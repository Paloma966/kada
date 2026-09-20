-- AI service database, kept separate from the Go business database `kada`.
--
-- docker-postgres only runs files in docker-entrypoint-initdb.d when the data
-- volume is initialised for the first time. On a host that already has a
-- `pgdata` volume, run these statements manually once:
--     CREATE DATABASE kada_ai;
--     \c kada_ai
--     CREATE EXTENSION IF NOT EXISTS vector;

CREATE DATABASE kada_ai;

\connect kada_ai

-- pgvector powers the RAG knowledge base; the pgvector/pgvector:pg16 image
-- ships the extension, it only has to be enabled inside this database.
-- The AI service creates its own tables (conversations, messages, vector
-- store) on startup / ingestion, so nothing else is defined here.
CREATE EXTENSION IF NOT EXISTS vector;
