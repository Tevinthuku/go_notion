
create extension if not exists "uuid-ossp";

CREATE TABLE IF NOT EXISTS pages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title JSONB NOT NULL,
    content JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by SERIAL REFERENCES users(id),
    position FLOAT NOT NULL DEFAULT 0
);

CREATE INDEX ON pages (created_by);

CREATE UNIQUE INDEX ON pages (created_by, position);


