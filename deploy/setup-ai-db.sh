#!/bin/bash
# Prepare the AI database on the PostgreSQL that already serves this host.
#
# deploy/postgres/initdb/01-create-ai-db.sql does the same thing, but files in
# docker-entrypoint-initdb.d only run when the data volume is created for the
# first time. On a host that is already running - the normal case when the AI
# service is added to an existing deployment - the statements have to be issued
# by hand, and this is that, made repeatable: the database is created only when
# it is missing and the extension is CREATE EXTENSION IF NOT EXISTS.
#
# Usage, on the server:
#   bash deploy/setup-ai-db.sh
#
# Overrides, when the defaults do not fit:
#   AI_DB=kada_ai_staging bash deploy/setup-ai-db.sh
#   PG_CONTAINER=my-postgres-1 bash deploy/setup-ai-db.sh
#   PG_CONTAINER= bash deploy/setup-ai-db.sh          # force the host's psql
#   PSQL="sudo -u postgres psql" bash deploy/setup-ai-db.sh
set -euo pipefail

AI_DB=${AI_DB:-kada_ai}
PG_USER=${PG_USER:-kada}
# Name of a running PostgreSQL container, when PostgreSQL is containerised.
# Uses the "-" (unset-only) form so an explicit empty value forces the local psql.
PG_CONTAINER=${PG_CONTAINER-kada-postgres-1}

# The database name is passed to psql as a variable, and also appears in -d, so
# refuse anything that could not be a database name in the first place.
case "$AI_DB" in
  *[!A-Za-z0-9_]*) echo "invalid database name: $AI_DB" >&2; exit 1 ;;
esac

# PSQL is the command prefix without -d, so the same prefix can address the
# maintenance database and the new one.
if [ -z "${PSQL:-}" ]; then
  if docker ps --format '{{.Names}}' 2>/dev/null | grep -qx "$PG_CONTAINER"; then
    PSQL="docker exec -i $PG_CONTAINER psql -U $PG_USER"
  else
    PSQL="sudo -u postgres psql"
  fi
fi
echo "Using: $PSQL"

# 1. The database itself. CREATE DATABASE cannot run inside a transaction, so
#    the statement is produced by a SELECT and executed with \gexec; :'db' is
#    psql-quoted, which keeps the name out of the SQL text.
$PSQL -d postgres -v ON_ERROR_STOP=1 -v db="$AI_DB" <<'SQL'
SELECT format('CREATE DATABASE %I', :'db')
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = :'db');
\gexec
SQL

# 2. pgvector, inside that database. The extension belongs to kada_ai, not to
#    the Go business database, and langchain-postgres does not install it.
if [ "$($PSQL -d "$AI_DB" -tAc "SELECT count(*) FROM pg_available_extensions WHERE name = 'vector'")" = "0" ]; then
  cat >&2 <<'EOF'
pgvector is not available to this PostgreSQL server, so the knowledge base
cannot be prepared. Install it first:

  native PostgreSQL (Ubuntu)  apt install postgresql-16-pgvector && systemctl restart postgresql
  Docker                      use the pgvector/pgvector:pg16 image (the root
                              docker-compose.yml already does)
EOF
  exit 1
fi
$PSQL -d "$AI_DB" -v ON_ERROR_STOP=1 -c 'CREATE EXTENSION IF NOT EXISTS vector;'

# 3. Read the result back instead of trusting the statements above.
if [ "$($PSQL -d "$AI_DB" -tAc "SELECT extname FROM pg_extension WHERE extname = 'vector'")" != "vector" ]; then
  echo "the vector extension could not be enabled in $AI_DB" >&2
  exit 1
fi

cat <<EOF
$AI_DB is ready with the vector extension.

Next:
  1. fill in /opt/kada/ai/ai.env (see deploy/ai.env.example)
  2. deploy the container:   bash /opt/kada/ai/deploy-ai.sh
  3. ingest the knowledge base:
       docker exec kada-ai python -m app.scripts.ingest_docs
     (rebuilds the collection from backend/ai/docs every run; without it the
      assistant still answers, just without platform references)
EOF
