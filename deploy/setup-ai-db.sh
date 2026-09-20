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
# Name of a running PostgreSQL container, when PostgreSQL is containerised. Treated as a hint, not as the
# only possibility: see the detection below.
# Uses the "-" (unset-only) form so an explicit empty value forces the local psql.
PG_CONTAINER=${PG_CONTAINER-kada-postgres-1}

# The database name is passed to psql as a variable, and also appears in -d, so
# refuse anything that could not be a database name in the first place.
case "$AI_DB" in
  *[!A-Za-z0-9_]*) echo "invalid database name: $AI_DB" >&2; exit 1 ;;
esac

# PSQL is the command prefix without -d, so the same prefix can address the
# maintenance database and the new one.
#
# The container is looked up by its *image*, not only by the name above. The name depends on the compose
# project directory, so a host that runs the same image under a different name used to fall straight
# through to `sudo -u postgres psql` - a command that cannot work where PostgreSQL is containerised (there
# is no postgres system user, and the failure says nothing about the real problem).
pg_container_by_image() {
  docker ps --format '{{.Names}}	{{.Image}}' 2>/dev/null |
    awk -F'\t' 'tolower($2) ~ /postgres|pgvector/ { print $1; exit }'
}

if [ -z "${PSQL:-}" ]; then
  if [ -n "$PG_CONTAINER" ] && docker ps --format '{{.Names}}' 2>/dev/null | grep -qx "$PG_CONTAINER"; then
    PSQL="docker exec -i $PG_CONTAINER psql -U $PG_USER"
  else
    found=$(pg_container_by_image)
    if [ -n "$found" ]; then
      PSQL="docker exec -i $found psql -U $PG_USER"
    elif command -v psql >/dev/null 2>&1 && id postgres >/dev/null 2>&1; then
      PSQL="sudo -u postgres psql"
    else
      cat >&2 <<EOF
could not find a PostgreSQL server to prepare

Looked for, in order:
  1. a running container named "${PG_CONTAINER:-<none>}"           not running
  2. a running container with a postgres or pgvector image         none
  3. a local psql plus a "postgres" system user                    not available

Point this script at the right one:

  # container under another name or image - list what is running:
  docker ps --format '{{.Names}}\t{{.Image}}'
  PG_CONTAINER=<that name> bash deploy/setup-ai-db.sh

  # a local PostgreSQL where you reach the superuser another way:
  PSQL="sudo -u postgres psql" bash deploy/setup-ai-db.sh
  PSQL="psql -U postgres"      bash deploy/setup-ai-db.sh
EOF
      exit 1
    fi
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
