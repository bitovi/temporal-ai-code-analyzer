CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS documents (
	id SERIAL PRIMARY KEY,
	repository TEXT,
	key TEXT,
	embedding vector(1536)
);
