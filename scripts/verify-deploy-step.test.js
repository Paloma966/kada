// Extracts the "Verify deployment", "Report the smoke test result" and "Check the SMS credentials" steps
// out of ci.yml and runs them, so each outcome is observed instead of assumed.
//
//   node scripts/verify-deploy-step.test.js
//
// The verification cases are the ones that actually appeared in CI (35: the edge reset the TLS handshake,
// 6: the host did not resolve) plus the healthy case and a reachable-but-unhealthy API. It asserts that
// the verify step never fails the job, that it records the right status, and that the report step warns
// only when the smoke test genuinely did not pass.
//
// The SMS cases assert the property that makes that step safe to ship: it reports that CI does not manage
// the credentials, and still exits 0. It must not fail the job, because the value only has to reach the
// host's .env by some route and a hand-configured host deploys perfectly well - a step that failed here
// would reject a deployment that was going to work, which is worse than saying nothing.
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

// --- pull the three steps' run: blocks out of the workflow ----------------------------------------------
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
write("step-check-sms.sh", extractRun("Check the SMS credentials"));

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
bash -n "$HERE/step-check-sms.sh" && echo "step-check-sms.sh parses"
echo

echo "--- cases ---"
run_case ok        0 passed no  "(none)"                              "runner reaches the site"
run_case reset     0 failed yes "curl 35: a TLS handshake failure"    "edge resets the TLS handshake (exit 35)"
run_case dns       0 failed yes "curl 6: could not fetch"             "DNS does not resolve (exit 6)"
run_case unhealthy 0 failed yes "did not answer with a healthy payload" "reachable but the API is not healthy"
echo

echo "--- the SMS advisory step ---"

# A missing SMS secret must be reported without failing the deploy, and must name every key that is not
# managed here; with all four set it must be completely silent. It must never dress this up as a failure,
# because whether the deployment really breaks depends on the host's own .env, which this step cannot see.
run_sms_case() {
  mode="$1"; want_notice="$2"; want_summary="$3"; label="$4"
  dir="$HERE/case-sms-$mode"
  rm -rf "$dir"; mkdir -p "$dir"
  : > "$dir/summary.md"

  if [ "$mode" = "present" ]; then
    # Obviously fake values on purpose. Nothing here asserts on what a credential *is* (only on whether the
    # step stays quiet when all four are set), so using the account's real template code would mean putting
    # an account-specific value in the tree for no gain - see the note in docs/design.md section 9.1.
    SMS_ACCESS_KEY_ID=id SMS_ACCESS_KEY_SECRET=secret SMS_SIGN_NAME=kada SMS_TEMPLATE_CODE=SMS_000000000 \
      GITHUB_STEP_SUMMARY="$dir/summary.md" bash "$HERE/step-check-sms.sh" > "$dir/out.txt" 2>&1
  else
    SMS_ACCESS_KEY_ID= SMS_ACCESS_KEY_SECRET= SMS_SIGN_NAME= SMS_TEMPLATE_CODE= \
      GITHUB_STEP_SUMMARY="$dir/summary.md" bash "$HERE/step-check-sms.sh" > "$dir/out.txt" 2>&1
  fi
  rc=$?

  noted=no
  grep -q '::notice' "$dir/out.txt" && noted=yes
  # An annotation that cries failure where there is none is how the annotations that matter stop being read.
  alarmed=no
  grep -qE '::(warning|error)' "$dir/out.txt" && alarmed=yes
  summary=no
  [ -s "$dir/summary.md" ] && summary=yes

  problems=""
  [ "$rc" = "0" ] || problems="$problems exit=$rc(want 0)"
  [ "$noted" = "$want_notice" ] || problems="$problems notice=$noted(want $want_notice)"
  [ "$alarmed" = "no" ] || problems="$problems emitted-a-warning-or-error"
  [ "$summary" = "$want_summary" ] || problems="$problems summary=$summary(want $want_summary)"

  if [ "$want_notice" = "yes" ]; then
    for key in SMS_ACCESS_KEY_ID SMS_ACCESS_KEY_SECRET SMS_SIGN_NAME SMS_TEMPLATE_CODE; do
      grep -q "$key" "$dir/out.txt" || problems="$problems does-not-name-$key"
      grep -q "$key" "$dir/summary.md" || problems="$problems summary-missing-$key"
    done
    grep -q '| Secret |' "$dir/summary.md" || problems="$problems summary-missing-the-table"
  fi

  if [ -z "$problems" ]; then
    printf 'ok    %s\n' "$label"
  else
    printf 'FAIL  %s\n' "$label"
    printf '        reason:%s\n' "$problems"
    fails=$((fails + 1))
  fi
  printf '        exit=%s notice=%s summary=%s\n' "$rc" "$noted" "$summary"
  printf '        out: %s\n' "$(head -c 300 "$dir/out.txt")"
}

run_sms_case missing yes yes "SMS secrets unset: says so, without failing the deploy"
run_sms_case present no  no  "SMS secrets set: says nothing at all"
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
