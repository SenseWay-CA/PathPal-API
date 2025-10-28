CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE Users (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    birth_date DATE NOT NULL,
    home_long DOUBLE PRECISION NOT NULL,
    home_lat DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);


CREATE TABLE Sessions (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES Users(user_id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    expires_at TIMESTAMP NOT NULL
);

CREATE TABLE Stats (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES Users(user_id) ON DELETE CASCADE,
    longitude DOUBLE PRECISION NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    battery INTEGER NOT NULL,
    heart_rate INTEGER NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE Events (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES Users(user_id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE Fences (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES Users(user_id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    longitude DOUBLE PRECISION NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    radius REAL NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE Invites (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES Users(user_id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    expires_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_users_email ON Users(email);


CREATE INDEX idx_sessions_user_id ON Sessions(user_id);
CREATE INDEX idx_sessions_token_hash ON Sessions(token_hash);


CREATE INDEX idx_stats_user_id ON Stats(user_id);
CREATE INDEX idx_stats_created_at ON Stats(created_at DESC);


CREATE INDEX idx_events_user_id ON Events(user_id);
CREATE INDEX idx_events_type ON Events(type);
CREATE INDEX idx_events_created_at ON Events(created_at DESC);


CREATE INDEX idx_fences_user_id ON Fences(user_id);


CREATE INDEX idx_invites_user_id ON Invites(user_id);
CREATE INDEX idx_invites_email ON Invites(email);
CREATE INDEX idx_invites_code ON Invites(code);

