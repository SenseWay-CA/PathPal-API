package main

// ─── Pi Camera Streaming Subsystem ──────────────────────────────────────────
//
// Architecture:
//   Pi (rpicam-vid | ffmpeg → MPEGTS/UDP) → udp://SERVER:8554
//   Server FFmpeg decodes H264 → JPEG frames → WebSocket clients (/ws/stream)
//
// WebSocket protocol:
//   binary message = raw JPEG bytes (one frame per message)
//   text message   = JSON status   { "type": "stream_status", "streaming": bool }
//
// Env vars:
//   STREAM_UDP_ADDR  — where to listen for the Pi stream (default udp://0.0.0.0:8554)
//   STREAM_FPS       — output frame rate                  (default 15)
//   STREAM_QUALITY   — ffmpeg -q:v 1-31, lower=better    (default 5)

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/net/websocket"
)

// ─── Client / Hub ────────────────────────────────────────────────────────────

type streamClient struct {
	ws   *websocket.Conn
	send chan []byte // buffered; frames are dropped if full (slow client)
}

type streamHub struct {
	mu         sync.RWMutex
	clients    map[*streamClient]bool
	register   chan *streamClient
	unregister chan *streamClient
	broadcast  chan []byte

	lastFrameMu sync.RWMutex
	lastFrame   []byte // most recent frame, sent immediately to new clients

	streaming atomic.Bool // true while FFmpeg has a live Pi connection
}

var hub = &streamHub{
	clients:    make(map[*streamClient]bool),
	register:   make(chan *streamClient, 16),
	unregister: make(chan *streamClient, 16),
	broadcast:  make(chan []byte, 30), // ~2 s of frames at 15 fps
}

// hubRun dispatches frames to all connected WebSocket clients.
// Call once in a goroutine at startup.
func hubRun() {
	for {
		select {

		case c := <-hub.register:
			hub.mu.Lock()
			hub.clients[c] = true
			hub.mu.Unlock()
			// Give new client the last frame immediately so the screen isn't blank
			hub.lastFrameMu.RLock()
			if hub.lastFrame != nil {
				select {
				case c.send <- hub.lastFrame:
				default:
				}
			}
			hub.lastFrameMu.RUnlock()
			sendStatus(c)
			log.Printf("[stream] client connected, total=%d", hubClientCount())

		case c := <-hub.unregister:
			hub.mu.Lock()
			if _, ok := hub.clients[c]; ok {
				delete(hub.clients, c)
				close(c.send)
			}
			hub.mu.Unlock()
			log.Printf("[stream] client disconnected, total=%d", hubClientCount())

		case frame := <-hub.broadcast:
			hub.lastFrameMu.Lock()
			hub.lastFrame = frame
			hub.lastFrameMu.Unlock()

			hub.mu.RLock()
			for c := range hub.clients {
				select {
				case c.send <- frame:
				default:
					// Slow client — drop frame rather than block everyone else
				}
			}
			hub.mu.RUnlock()
		}
	}
}

func hubClientCount() int {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	return len(hub.clients)
}

// sendStatus pushes a JSON status message into a single client's send queue
func sendStatus(c *streamClient) {
	msg, _ := json.Marshal(map[string]any{
		"type":      "stream_status",
		"streaming": hub.streaming.Load(),
		"clients":   hubClientCount(),
	})
	select {
	case c.send <- msg:
	default:
	}
}

// broadcastStatus sends a JSON status message to all connected clients
func broadcastStatus() {
	msg, _ := json.Marshal(map[string]any{
		"type":      "stream_status",
		"streaming": hub.streaming.Load(),
		"clients":   hubClientCount(),
	})
	hub.mu.RLock()
	for c := range hub.clients {
		select {
		case c.send <- msg:
		default:
		}
	}
	hub.mu.RUnlock()
}

// ─── WebSocket handler ───────────────────────────────────────────────────────

// wsServer uses golang.org/x/net/websocket (already an indirect dep — no go get needed).
// Handshake accepts all origins so Android apps connect without an Origin header.
var wsServer = websocket.Server{
	Handshake: func(_ *websocket.Config, _ *http.Request) error { return nil },
	Handler:   websocket.Handler(streamWSConn),
}

// streamWSHandler is the Echo handler for GET /ws/stream
func streamWSHandler(c echo.Context) error {
	wsServer.ServeHTTP(c.Response(), c.Request())
	return nil
}

// streamWSConn is called once per connected client. Blocks until disconnected.
func streamWSConn(ws *websocket.Conn) {
	client := &streamClient{
		ws:   ws,
		send: make(chan []byte, 90), // ~6 s buffer at 15 fps
	}
	hub.register <- client

	go client.writePump()
	client.readPump() // blocks

	hub.unregister <- client
}

// writePump drains the send channel and writes to the WebSocket.
// Frames starting with '{' are sent as text (JSON); all others as binary (JPEG).
func (c *streamClient) writePump() {
	defer c.ws.Close() // close conn so readPump also unblocks on write failure
	for msg := range c.send {
		var err error
		if len(msg) > 0 && msg[0] == '{' {
			err = websocket.Message.Send(c.ws, string(msg)) // text frame
		} else {
			err = websocket.Message.Send(c.ws, msg) // binary frame
		}
		if err != nil {
			return
		}
	}
}

// readPump reads incoming messages until the client disconnects.
func (c *streamClient) readPump() {
	defer c.ws.Close()
	for {
		var msg []byte
		if err := websocket.Message.Receive(c.ws, &msg); err != nil {
			break
		}
		// Future: handle detection result JSON sent back from the client
		_ = msg
	}
}

// ─── REST status endpoint ─────────────────────────────────────────────────────

// streamStatusHandler handles GET /stream/status
func streamStatusHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{
		"streaming": hub.streaming.Load(),
		"clients":   hubClientCount(),
		"udp_addr":  piUDPAddr(),
		"fps":       streamFPS(),
	})
}

// ─── FFmpeg ingestion ─────────────────────────────────────────────────────────

func piUDPAddr() string {
	if v := os.Getenv("STREAM_UDP_ADDR"); v != "" {
		return v
	}
	return "udp://0.0.0.0:8554"
}

func streamFPS() string {
	if v := os.Getenv("STREAM_FPS"); v != "" {
		return v
	}
	return "15"
}

func streamQuality() string {
	if v := os.Getenv("STREAM_QUALITY"); v != "" {
		return v
	}
	return "5" // ffmpeg -q:v: 1=best quality, 31=worst; 5 is a good balance
}

// StartStream runs forever — reconnects automatically when the Pi drops.
// Call once in a goroutine at startup.
func StartStream() {
	for {
		if err := runFFmpeg(); err != nil {
			log.Printf("[stream] ffmpeg stopped: %v", err)
		}
		hub.streaming.Store(false)
		broadcastStatus()
		log.Printf("[stream] retrying in 3 s...")
		time.Sleep(3 * time.Second)
	}
}

// runFFmpeg spawns an FFmpeg process that:
//  1. Listens for MPEGTS/H264 from the Pi on UDP
//  2. Decodes every frame — smooth playback, not just I-frames
//  3. Throttles to STREAM_FPS and outputs continuous MJPEG on stdout
//
// Blocks until FFmpeg exits.
func runFFmpeg() error {
	addr := piUDPAddr()
	fps := streamFPS()
	quality := streamQuality()

	log.Printf("[stream] starting ffmpeg → src=%s fps=%s q=%s", addr, fps, quality)

	cmd := exec.Command("ffmpeg",
		"-loglevel", "error",
		"-fflags", "nobuffer+discardcorrupt",
		"-flags", "low_delay",
		"-err_detect", "ignore_err",
		"-i", addr,
		"-vf", "fps="+fps+",scale=640:480",
		"-f", "image2pipe",
		"-vcodec", "mjpeg",
		"-q:v", quality,
		"pipe:1",
	)
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	hub.streaming.Store(true)
	broadcastStatus()
	log.Printf("[stream] ffmpeg running (pid %d)", cmd.Process.Pid)

	parseJPEGStream(stdout) // blocks until the pipe closes

	return cmd.Wait()
}


// JPEG framing: SOI = 0xFF 0xD8 … EOI = 0xFF 0xD9
func parseJPEGStream(r io.Reader) {
	const maxBuf = 8 * 1024 * 1024 // 8 MB safety cap; reset on overflow
	soi := []byte{0xFF, 0xD8}
	eoi := []byte{0xFF, 0xD9}

	var buf []byte
	tmp := make([]byte, 64*1024)

	for {
		n, err := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			return
		}

		for {
			start := bytes.Index(buf, soi)
			if start == -1 {
				buf = nil
				break
			}
			eoiOffset := bytes.Index(buf[start+2:], eoi)
			if eoiOffset == -1 {
				if start > 0 {
					buf = buf[start:]
				}
				break
			}
			end := start + 2 + eoiOffset + 2

			frame := make([]byte, end-start)
			copy(frame, buf[start:end])
			buf = buf[end:]

			select {
			case hub.broadcast <- frame:
			default:
				// Broadcast channel full — drop frame
			}
		}

		if len(buf) > maxBuf {
			log.Printf("[stream] buffer overflow — resetting")
			buf = nil
		}
	}
}
