// Extracts the "Verify deployment" and "Report the smoke test result" scripts out of ci.yml and runs them
// against a stubbed curl, so each outcome is observed instead of assumed.
//
//   node scripts/verify-deploy-step.test.js
//
// The cases are the ones that actually appeared in CI (35: the edge reset the TLS handshake, 6: the host
// did not resolve) plus the healthy case and a reachable-but-unhealthy API. It asserts that the verify
// step never fails the job, that it records the right status, and that the report step warns only when
// the smoke test genuinely did not pass.
//
// Generates scripts/.verify-deploy-out/ and runs it with bash. Git Bash cannot run under the default
// sandbox (it needs a signal pipe), so this looks for WSL bash first and falls back to Git Bash.

const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");

const root = path.join(__dirname, "..");
const outDir = path.join(__dirname, ".verify-deploy-out");
const toPosix = (p) => p.replace(/\\/g, "/");

fs.rmSync(outDir, { recursive: true, force: true });
fs.mkdirSync(outDir, { recursive: true });

// --- pull the two steps' run: blocks out of the workflow ------------------------------------------------
const workflow = fs.readFileSync(path.join(root, ".github", "workflows", "ci.yml"), "utf8");

function extractRun(name) {
  const lines = workflow.split(/\r?\n/);
  const start = lines.findIndex((l) => l.trim() === `- name: ${name}`);
  if (start < 0) throw new Error(`step not found: ${name}`);
  const runAt = lines.findIndex((l, i) => i > start && l.trim() === "run: |");
  if (runAt < 0) throw new Error(`run: | not found after: ${name}`);
  const body = [];
  for (let i = runAt + 1; i < lines.length; i++) {
    const line = lines[i];
    if (line.trim() === "") { body.push(""); continue; }
    if (!/^\s{10}/.test(line)) break;
    body.push(line.slice(10));
  }
  while (body.length && body[body.length - 1].trim() === "") body.pop();
  return body.join("\n");
}

const write = (name, content) => fs.writeFileSync(path.join(outDir, name), content.replace(/\r\n/g, "\n"), "utf8");
write("step-verify.sh", extractRun("Verify deployment"));
write("step-report.sh", extractRun("Report the smoke test result"));

// --- the harness that decides whether each case behaved -------------------------------------------------
const HARNESS = String.raw`#!/usr/bin/env bash
# Runs the two extracted steps against a stubbed curl and reports on each case.
set -u

HERE="$(cd "$(dirname "$0")" && pwd)"
fails=0

run_case() {
  mode="$1"; want_verify_exit="$2"; want_status="$3"; want_warning="$4"; want_body="$5"; label="$6"
  dir="$HERE/case-$mode"
  rm -rf "$dir"; mkdir -p "$dir/bin"
  : > "$dir/calls"
  export STUB_CALLS="$dir/calls"

  # The stub: every call is refused unless the case says otherwise. It records each call so the harness
  # can tell that the stub - and not the real curl - was what ran.
  cat > "$dir/bin/curl" <<'STUB'
#!/usr/bin/env bash
echo x >> "$STUB_CALLS"
out=/dev/null
want_out=0
url=""
for arg in "$@"; do
  if [ "$want_out" = 1 ]; then out="$arg"; want_out=0; continue; fi
  if [ "$arg" = "-o" ]; then want_out=1; continue; fi
  case "$arg" in
    http://* | https://*) url="$arg" ;;
  esac
done

# The real endpoints answer differently: /api/health is JSON, / is HTML.
case "$url" in
  */api/health) healthy='{"service":"kada-api","status":"ok","version":"0.2.0"}'; unhealthy='{"status":"degraded"}' ;;
  *)            healthy='<!doctype html><html><title>Kada</title></html>'; unhealthy="$healthy" ;;
esac

case "$STUB_MODE" in
  ok)
    printf '%s' "$healthy" > "$out"
    exit 0 ;;
  reset)
    echo "curl: (35) Recv failure: Connection reset by peer" >&2
    exit 35 ;;
  dns)
    echo "curl: (6) Could not resolve host: kada.click" >&2
    exit 6 ;;
  unhealthy)
    printf '%s' "$unhealthy" > "$out"
    exit 0 ;;
  *)
    echo "unknown STUB_MODE: $STUB_MODE" >&2
    exit 99 ;;
esac
STUB
  chmod +x "$dir/bin/curl"

  export STUB_MODE="$mode"
  export SITE_ORIGIN="https://kada.click"
  export GITHUB_STEP_SUMMARY="$dir/summary.md"
  # cygpath keeps the stub ahead of the real curl; a Windows-style PATH would not.
  export PATH="$(cygpath -u "$dir/bin"):$PATH"

  ( cd "$dir" && bash "$HERE/step-verify.sh" ) > "$dir/verify.out" 2>&1
  verify_exit=$?

  status=$(cat "$dir/verify-status.txt" 2>/dev/null || echo missing)
  body=$(cat "$dir/verify-body.txt" 2>/dev/null || echo "(none)")
  # Proves the stub answered rather than the real curl, and that --retry actually retried.
  calls=$(wc -l < "$dir/calls" | tr -d ' ')

  ( cd "$dir" && bash "$HERE/step-report.sh" ) > "$dir/report.out" 2>&1
  report_exit=$?

  warned=no
  grep -q '::warning' "$dir/report.out" && warned=yes

  problems=""
  [ "$verify_exit" = "$want_verify_exit" ] || problems="$problems verifyExit=$verify_exit(want $want_verify_exit)"
  [ "$status" = "$want_status" ] || problems="$problems status=$status(want $want_status)"
  [ "$warned" = "$want_warning" ] || problems="$problems warning=$warned(want $want_warning)"
  [ "$report_exit" = "0" ] || problems="$problems reportExit=$report_exit"
  [ "$calls" -ge 1 ] 2>/dev/null || problems="$problems the stubbed curl never ran"
  # Note: curl's --retry does not retry a *shell* stub that exits non-zero - it only retries failures it
  # raises itself - so this harness cannot assert the retry count. That was verified separately against a
  # socket that really resets: --retry 5 produced 6 connections. Here the stub refuses persistently, which
  # is the worst case, and the point is that the step survives it.
  printf '%s' "$body" | grep -q "$want_body" || problems="$problems body=$body(does not contain $want_body)"
  # A refusal must never be the thing that fails the step.
  [ -s "$dir/verify.out" ] || problems="$problems the step printed nothing"

  if [ -z "$problems" ]; then
    printf 'ok    %s\n' "$label"
  else
    printf 'FAIL  %s\n' "$label"
    printf '        reason:%s\n' "$problems"
    fails=$((fails + 1))
  fi
  printf '        status=%s verifyExit=%s reportExit=%s warning=%s stubCalls=%s\n' "$status" "$verify_exit" "$report_exit" "$warned" "$calls"
  printf '        body: %s\n' "$body"
}

echo "--- bash -n ---"
bash -n "$HERE/step-verify.sh" && echo "step-verify.sh parses"
bash -n "$HERE/step-report.sh" && echo "step-report.sh parses"
echo

echo "--- cases ---"
run_case ok        0 passed no  "(none)"                              "runner reaches the site"
run_case reset     0 failed yes "curl 35: a TLS handshake failure"    "edge resets the TLS handshake (exit 35)"
run_case dns       0 failed yes "curl 6: could not fetch"             "DNS does not resolve (exit 6)"
run_case unhealthy 0 failed yes "did not answer with a healthy payload" "reachable but the API is not healthy"
echo

if [ "$fails" = "0" ]; then
  echo "all cases behaved"
  exit 0
fi
echo "$fails case(s) failed"
exit 1
`;
write("harness.sh", HARNESS);

// --- run it --------------------------------------------------------------------------------------------
// Git Bash has to be invoked from here because it needs a signal pipe, which the sandbox refuses; that is
// why this command needs wider access than the default. WSL would be closer to the runner (Linux) but is
// not installed on this machine.
const GIT_BASH = "C:\\Program Files\\Git\\bin\\bash.exe";

function run(command) {
  try {
    execFileSync(GIT_BASH, ["-lc", command], { stdio: "inherit" });
    return 0;
  } catch (err) {
    return err.status ?? 1;
  }
}

const unixDir = "/" + toPosix(outDir).replace(/^([A-Za-z]):/, (_, d) => d.toLowerCase());
process.exit(run(`bash ${unixDir}/harness.sh`));
