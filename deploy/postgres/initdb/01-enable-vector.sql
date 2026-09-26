-- Runs on the first initialisation of the compose volume only.
--
-- The assistant's knowledge base is a pgvector column in the application's own database, so the extension
-- has to exist in `kada` before cmd/ai-ingest can create its table. CREATE EXTENSION is per database, not
-- per server, and the image (pgvector/pgvector:pg16) ships it - this is what enables it here.
--
-- On a host whose PostgreSQL was set up by hand, cmd/ai-ingest runs the same statement and says so if the
-- extension is not available at all.
CREATE EXTENSION IF NOT EXISTS vector;
