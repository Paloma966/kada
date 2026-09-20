#!/bin/bash
# Set or refresh KEY=VALUE pairs in a runtime env file, on the server.
#
#   bash upsert-env.sh /opt/kada/ai/ai.env \
#     --set DEEPSEEK_API_KEY="$DEEPSEEK_API_KEY" \
#     --set aliyun="$DASHSCOPE_API_KEY" \
#     --require DEEPSEEK_API_KEY --require aliyun
#
# Why this exists: the AI keys and the SMS credentials used to be typed into a file on the server by hand,
# and nothing anywhere failed when they were forgotten. The service started, passed its health check, and
# then failed every request that mattered - "the AI page is broken" and "the SMS code never arrives" were
# both, in the end, an empty line in a file nobody had looked at. The values now come from GitHub Secrets
# through CI, and --require turns "forgot to configure it" into a failed deployment that says which key is
# missing, before any service on the host has been touched.
#
# --set with an EMPTY value is a no-op: it leaves whatever the file already holds. That is what makes it
# safe to run on a host that was configured by hand before the secrets existed - and --require is what
# stops "empty means skip" from quietly passing when nothing is there at all.
#
# --require and --warn draw the line between the two kinds of missing value, which is a decision about the
# product rather than about this script:
#
#   --require  the deployment cannot produce a working system without it, so fail. The AI keys are in this
#              group: a service that starts, passes /healthz and then fails every question is worse to
#              diagnose than a refused deploy.
#   --warn     the deployment is still meaningful without it, but the gap must be impossible to miss. The
#              SMS credentials are in this group: the signing name and template have to be approved in the
#              Aliyun console, which takes days, and blocking every deploy until then would stop unrelated
#              fixes from shipping. The warning says exactly what is broken, in the log and - from CI's own
#              check on the runner - as an annotation and a job-summary entry.
#
# The value is never passed through sed: an API key can contain `/`, `&` or `\`, every one of which sed
# would treat as part of its own syntax. The line is removed with grep and appended with printf instead.
set -euo pipefail

usage() {
  cat >&2 <<'EOF'
usage: upsert-env.sh FILE [--set KEY=VALUE]... [--require KEY]... [--warn KEY]...

  --set KEY=VALUE   Write KEY=VALUE into FILE, replacing any existing KEY= line.
                    An empty VALUE leaves the current line untouched.
  --require KEY     Fail unless FILE ends up containing a non-empty KEY=.
  --warn KEY        Warn loudly, but continue, when KEY ends up empty.
EOF
}

file=""
sets=()
requires=()
warns=()

while [ $# -gt 0 ]; do
  case "$1" in
    --set)
      [ $# -ge 2 ] || { echo "--set needs KEY=VALUE" >&2; usage; exit 2; }
      sets+=("$2")
      shift 2
      ;;
    --require)
      [ $# -ge 2 ] || { echo "--require needs KEY" >&2; usage; exit 2; }
      requires+=("$2")
      shift 2
      ;;
    --warn)
      [ $# -ge 2 ] || { echo "--warn needs KEY" >&2; usage; exit 2; }
      warns+=("$2")
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      if [ -z "$file" ]; then
        file="$1"
      else
        echo "unexpected argument: $1" >&2
        usage
        exit 2
      fi
      shift
      ;;
  esac
done

if [ -z "$file" ]; then
  usage
  exit 2
fi

if [ ! -f "$file" ]; then
  # 600 from the moment it exists: these files hold provider keys and the JWT secret, and a file that is
  # world-readable for even a second is a file that has leaked. The umask covers the creation itself; the
  # explicit chmod makes the intent independent of whatever umask the calling shell happens to have.
  mkdir -p "$(dirname "$file")"
  (umask 077; : > "$file")
  chmod 600 "$file"
  echo "created $file (mode 600)"
fi

for pair in ${sets[@]+"${sets[@]}"}; do
  key=${pair%%=*}
  value=${pair#*=}

  # The key is interpolated into a grep pattern, so it has to be boring: letters, digits and underscore.
  case "$key" in
    ''|*[!A-Za-z0-9_]*)
      echo "refusing to write an invalid key: $key" >&2
      exit 2
      ;;
  esac

  if [ -z "$value" ]; then
    echo "keeping the existing $key in $file (no value given)"
    continue
  fi

  tmp=$(mktemp)
  # `|| true`: grep exits 1 when it removes nothing, which under `set -e` would abort the script.
  grep -v -E "^[[:space:]]*${key}=" "$file" > "$tmp" || true
  printf '%s=%s\n' "$key" "$value" >> "$tmp"
  cat "$tmp" > "$file"
  rm -f "$tmp"
  echo "set $key in $file"
done

missing=""
for key in ${requires[@]+"${requires[@]}"}; do
  if ! grep -qE "^[[:space:]]*${key}=[^[:space:]]" "$file"; then
    missing="$missing $key"
  fi
done

if [ -n "$missing" ]; then
  cat >&2 <<EOF
missing required value(s) in $file:$missing

Add them as repository secrets (Settings -> Secrets and variables -> Actions): the
deployment refuses to continue rather than start a service that cannot work, and it
refuses before touching anything on this host.

Expected secrets:
  DEEPSEEK_API_KEY    chat model key           -> DEEPSEEK_API_KEY in ai.env
  DASHSCOPE_API_KEY   Bailian embedding key    -> aliyun in ai.env
  AI_INTERNAL_SECRET  gateway shared secret    -> both ai.env and backend/.env (host side generated)
  SMS_ACCESS_KEY_ID, SMS_ACCESS_KEY_SECRET, SMS_SIGN_NAME, SMS_TEMPLATE_CODE -> backend/.env
      (warned about rather than required, but without them NOBODY CAN SIGN IN:
       phone + SMS code is the only sign-in method)
EOF
  exit 1
fi

# Advisory keys: the deployment proceeds, and says out loud what will not work.
empty=""
for key in ${warns[@]+"${warns[@]}"}; do
  if ! grep -qE "^[[:space:]]*${key}=[^[:space:]]" "$file"; then
    empty="$empty $key"
  fi
done

if [ -n "$empty" ]; then
  # The ::warning:: form is a GitHub Actions workflow command; parsed when this runs inside a job, and
  # harmless plain text otherwise (a hand-run deploy on the host). The same message is repeated as
  # ordinary output because the annotation is best-effort once the text has travelled back over SSH.
  echo "::warning title=$file has empty values::$empty is empty in $file"
  cat >&2 <<EOF
================================================================================
WARNING: empty value(s) in $file:$empty

The deployment continues, but whatever needs these will fail at request time.
For the SMS credentials that means NOBODY CAN SIGN IN: phone + SMS code is the
only sign-in method, and in release mode the code is not written to the log.
================================================================================
EOF
fi

exit 0
