CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE user_role AS ENUM (
    'Cane_User',
    'Caregiver'
);

CREATE TYPE event_type AS ENUM (
    'SOS',
    'Fall',
    'Low_Battery',
    'Geofence_Exit',
    'Geofence_Enter'
);

CREATE TABLE Users (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL,
    type user_role NOT NULL,
    birth_date DATE NOT NULL,
    home_long DOUBLE PRECISION NOT NULL,
    home_lat DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


CREATE TABLE Sessions (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES Users(user_id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE Stats (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES Users(user_id) ON DELETE CASCADE,
    longitude DOUBLE PRECISION NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    battery SMALLINT NOT NULL,
    heart_rate SMALLINT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE Events (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES Users(user_id) ON DELETE CASCADE,
    type event_type NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE Fences (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES Users(user_id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    longitude DOUBLE PRECISION NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    radius REAL NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE Invites (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES Users(user_id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE Guardians (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    cane_user_id UUID NOT NULL REFERENCES Users(user_id) ON DELETE CASCADE,
    caregiver_user_id UUID NOT NULL REFERENCES Users(user_id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),    
    UNIQUE(cane_user_id, caregiver_user_id)
);


CREATE INDEX idx_sessions_user_id ON Sessions(user_id);

CREATE INDEX idx_stats_user_id ON Stats(user_id);
CREATE INDEX idx_stats_created_at ON Stats(created_at DESC);

CREATE INDEX idx_events_user_id ON Events(user_id);
CREATE INDEX idx_events_type ON Events(type);
CREATE INDEX idx_events_created_at ON Events(created_at DESC);

CREATE INDEX idx_fences_user_id ON Fences(user_id);

CREATE INDEX idx_invites_user_id ON Invites(user_id);
CREATE INDEX idx_invites_email ON Invites(email);

CREATE INDEX idx_guardians_cane_user_id ON Guardians(cane_user_id);
CREATE INDEX idx_guardians_caregiver_user_id ON Guardians(caregiver_user_id);