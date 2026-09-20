#!/bin/bash
# Deploy or refresh the Kada AI service container on the production host.
#
# Runs ON THE SERVER. CI copies this file to /opt/kada/ai/ and invokes it over SSH, so the
# pipeline and a hand-run deploy cannot drift. The image is built here rather than shipped,
# because the AI service is a whole Python environment and not a single binary; only the
# source changes between deploys, so the pip layer of the image stays cached.
#
# Expects:
#   /opt/kada/ai/ai.env             secrets and connection strings (deploy/ai.env.example)
#   /opt/kada/ai/docker-compose.yml deploy/docker-compose.ai.yml
#   /tmp/kada-ai-src.tar.gz         the backend/ai source tree
#
# It refuses to run when a provider key is empty: CI syncs those from GitHub Secrets first
# (deploy/upsert-env.sh), so an empty value means the secret is missing, not that the file is.
#
# Produces a container named kada-ai listening on 127.0.0.1:8000, which is where
# the Go gateway looks for it (config.AIBaseURL defaults to http://127.0.0.1:8000).
set -euo pipefail

AI_DIR=${AI_DIR:-/opt/kada/ai}
TARBALL=${TARBALL:-/tmp/kada-ai-src.tar.gz}
COMPOSE_FILE="$AI_DIR/docker-compose.yml"
ENV_FILE="$AI_DIR/ai.env"

[ -f "$TARBALL" ] || { echo "missing source tarball: $TARBALL" >&2; exit 1; }
[ -f "$COMPOSE_FILE" ] || { echo "missing compose file: $COMPOSE_FILE" >&2; exit 1; }

if [ ! -f "$ENV_FILE" ]; then
  cat >&2 <<'EOF'
missing /opt/kada/ai/ai.env

Create it from deploy/ai.env.example (chmod 600) and fill in:
  POSTGRES_URL, REDIS_URL, KADA_API_BASE, DEEPSEEK_API_KEY, aliyun, AI_INTERNAL_SECRET

This refuses to continue instead of deploying a service that starts, passes its
health check and then fails every chat request on the provider side.
EOF
  exit 1
fi

# The provider keys are the difference between a service that answers and a service that starts, passes
# its health check, and then fails every single question - which is the hardest possible way to find out
# that a key is missing. CI writes them from GitHub Secrets before this script runs (deploy/upsert-env.sh),
# so an empty one here means the secret was never set, not that the file is wrong.
missing_keys=""
for key in DEEPSEEK_API_KEY aliyun; do
  if ! grep -qE "^[[:space:]]*${key}=[^[:space:]]" "$ENV_FILE"; then
    missing_keys="$missing_keys $key"
  fi
done
if [ -n "$missing_keys" ]; then
  cat >&2 <<EOF
missing required value(s) in $ENV_FILE:$missing_keys

  DEEPSEEK_API_KEY  the chat model key
  aliyun            the Aliyun Bailian (DashScope) embedding key - the variable name is literally "aliyun"

Set the matching repository secret (DEEPSEEK_API_KEY / DASHSCOPE_API_KEY) and re-run the
deployment; CI writes it into this file. Refusing to build an image whose chat requests
would all fail.
EOF
  exit 1
fi

# The guard on the AI service's own port is only as good as this value: without
# it, any process on this host can send its own X-Kada-User-ID. Warn rather than
# refuse, because the rest of the service works while the secret is being set up.
if ! grep -qE '^[[:space:]]*AI_INTERNAL_SECRET=[^[:space:]]' "$ENV_FILE"; then
  echo "warning: AI_INTERNAL_SECRET is empty in $ENV_FILE - the AI service will accept" >&2
  echo "         requests from any process on this host, not just the Go gateway" >&2
fi

# The kada_ai database must exist before the container starts: init_db() runs in
# the lifespan hook, so a missing database is a crash loop rather than a bad
# health check. deploy/setup-ai-db.sh prepares it.
echo "Unpacking the AI service source"
rm -rf "$AI_DIR/src"
mkdir -p "$AI_DIR/src"
tar xzf "$TARBALL" -C "$AI_DIR/src"

cd "$AI_DIR"

# Keep the image that is running now, so a failed start can be rolled back.
if docker image inspect kada-ai:deploy >/dev/null 2>&1; then
  docker tag kada-ai:deploy kada-ai:previous
fi

echo "Building the AI image"
docker compose -f "$COMPOSE_FILE" build

echo "Restarting the AI container"
docker compose -f "$COMPOSE_FILE" up -d

# The service answers /healthz only after init_db() succeeded, so this also
# covers "the database is missing or unreachable".
echo "Waiting for the AI service on 127.0.0.1:8000"
for _ in $(seq 1 30); do
  if curl -fsS --max-time 3 http://127.0.0.1:8000/healthz >/dev/null 2>&1; then
    echo "AI service is healthy"
    docker compose -f "$COMPOSE_FILE" ps
    exit 0
  fi
  sleep 2
done

echo "the AI service did not become healthy within 60s" >&2
docker logs --tail 60 kada-ai >&2 || true

if docker image inspect kada-ai:previous >/dev/null 2>&1; then
  echo "rolling back to the previous image" >&2
  docker tag kada-ai:previous kada-ai:deploy
  # --force-recreate: the tag now points at the older image, and the running
  # container has to be rebuilt from it rather than left as it is.
  docker compose -f "$COMPOSE_FILE" up -d --force-recreate || true
fi
exit 1
