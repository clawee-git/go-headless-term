#!/usr/bin/env bash
set -euo pipefail

PROG="ci/run-tests.sh"
MACHINE="${CLAWEE_CI_MACHINE:-burrowee-ci}"
SRC="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

REMOTE_PREFIX="/tmp/clawee-ght-"
REMOTE_DIR="${CLAWEE_CI_DIR:-$REMOTE_PREFIX$(id -un)-$(printf '%s' "$SRC" | cksum | cut -d' ' -f1)}"
DEPS_DIR="$REMOTE_DIR.deps"

LOCK_ROOT="/tmp/ci-lock"
LOCK="$LOCK_ROOT/clawee"
LOCK_HEARTBEAT_S="${CLAWEE_CI_LOCK_HEARTBEAT_S:-30}"
LOCK_STALE_MULTIPLE=4
LOCK_PROJECT="${CLAWEE_CI_LOCK_PROJECT:-$(git -C "$SRC" rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)}"
LOCK_SESSION="${CLAWEE_CI_LOCK_SESSION:-unrecorded}"
RUN_ID="$(id -un | tr -c 'A-Za-z0-9\n' '_')-$$-$(date +%s)"
RUN_BASE="$REMOTE_DIR.run.$RUN_ID"
POLL_S="${CLAWEE_CI_POLL_S:-3}"
FOLLOW_MAX_MISSES="${CLAWEE_CI_FOLLOW_MAX_MISSES:-40}"
DEAD_POLLS=3

EVIDENCE=0
ARTIFACTS_DIR=""
SHUFFLE=0
REPEAT=1
PACKAGES=(./...)
LOCK_STATE=""
WS_DIRS=()
WS_MODULES=()
WS_DESTS=()
WS_RSYNC_PATHS=()
FOLLOW_OUTCOME=""

usage() {
    cat <<'USAGE'
Usage:
  ci/run-tests.sh [options] [pkg...]   the suite; packages default to ./...
  ci/run-tests.sh -h|--help            this text

Options come before the packages. Any of them turns on the evidence mode:
go test -json, its readable lines streamed, then per-package and module test
and case counts (pass/fail/skip), each skipped and failed name, and each
shuffle seed. Without options the plain suite runs, unchanged.
  --artifacts <dir>   also -covermode=set -coverpkg=./... -coverprofile,
                      whatever packages are named, and copy test.json,
                      cover.out and covered.txt to <dir>. covered.txt is the
                      sorted set of covered blocks, without blocks in
                      `<pkg>_test/` support packages (lang/go.md). Named
                      packages with no test files are left out of the run and
                      listed. <dir> must not exist — evidence is never
                      overwritten — and its parent must.
  --shuffle           -shuffle=on; the seed per package is in the log
  --repeat <n>        -count=<n> instead of -count=1

Environment:
  CLAWEE_CI_MACHINE             the machine (default burrowee-ci)
  CLAWEE_CI_DIR                 remote tree (default /tmp/clawee-ght-<user>-<cksum
                                of this checkout>); must be /tmp/clawee-ght-<name>,
                                <name> of letters, digits and ._- with no '..'
                                and not ending in '.'
  CLAWEE_CI_LOCK_PROJECT        project id recorded on the lock (default: the branch)
  CLAWEE_CI_LOCK_SESSION        session id recorded on the lock (default: unrecorded)
  CLAWEE_CI_LOCK_HEARTBEAT_S    heartbeat interval in seconds (default 30); stale after 4
  CLAWEE_CI_POLL_S              seconds between log polls (default 3)
  CLAWEE_CI_FOLLOW_MAX_MISSES   failed polls in a row before following gives up (default 40)
The three numbers must be whole numbers of at least 1; anything else is a
usage error before the machine is contacted.

Exit status (a closed stderr never changes it):
  0        the build and every test passed, and this run's lock was released —
           or the release found the lock no longer naming this run (warned):
           this run then holds no lock
  1        the build or a test failed (any non-zero status from go is reported
           as 1, so 2 and 3 below always mean this script), or the machine
           could not be reached, the tree not synced, the run ended without a
           status, or its evidence could not be copied home — or the suite
           passed but this run's lock is or may be left held: its remote kill
           could not be confirmed, or its release went unanswered (the
           commands to clear it are on stderr)
  2        usage error, refused before any contact: bad option, option after
           packages, existing --artifacts directory, bad environment value
           (CLAWEE_CI_DIR outside /tmp/clawee-ght-<name>, or a <name> ending
           in '.'), a derived remote path the machine guard would refuse, or
           a local go.work whose `use` is not a directory or names a module
           that cannot be mirrored safely (empty, a '.' segment, '..')
  3        the Clawee CI lock is held by another run, was released while this
           run checked it, or its root is not writable — never waited for,
           never broken; a MISSING root is exit 1: it is the machine's to
           create, not this script's
  130/143/129  interrupted by INT/TERM/HUP; 141  stdout closed with SIGPIPE.
           The run is stopped and the lock released first. While the run is
           followed a signal is acted on at once, also when it is sent to this
           script's pid alone. These remote steps are bounded and finish
           before the signal is acted on: the probe, the lock take, the sync,
           the launch and the evidence copy-back. The stop that follows the
           signal can itself take up to about 22 s before the release.
Once the teardown (stop, then release) has begun, further signals are
ignored and the exit is the run's own status: a signal replaces the run's
status only when it lands before the teardown, the copy-back included.
A closed stdout WITHOUT SIGPIPE (`>&-`, or a caller that ignores SIGPIPE) does
not stop the run: messages continue on stderr and the status is the run's.
The lock outcome never replaces a failing status.
USAGE
}

usage_error() {
    echo "$PROG: $1" >&2 2>/dev/null || echo >/dev/null
    usage >&2 2>/dev/null || true
    exit 2
}

say() {
    echo "$PROG: $*" 2>/dev/null && return 0
    stdout_closed
    echo "$PROG: $*" 2>/dev/null || echo >/dev/null
}

warn() {
    echo "$PROG: $*" >&2 2>/dev/null || echo >/dev/null
}

print_command() {
    echo "$*" >&2 2>/dev/null || echo >/dev/null
}

emit() {
    printf '%s' "$1" 2>/dev/null && return 0
    stdout_closed
    printf '%s' "$1" 2>/dev/null || echo >/dev/null
}

stdout_closed() {
    echo >/dev/null
    if : >&2; then
        exec 1>&2
        warn "stdout was closed — messages continue on stderr"
    else
        exec 1>/dev/null
    fi
}

parse_args() {
    local arg
    for arg in "$@"; do
        case "$arg" in -h|--help) usage; exit 0 ;; esac
    done
    while [ $# -gt 0 ]; do
        case "$1" in
            --artifacts) [ $# -ge 2 ] || usage_error "--artifacts needs a directory"; set_artifacts_dir "$2"; shift ;;
            --artifacts=*) set_artifacts_dir "${1#*=}" ;;
            --shuffle) SHUFFLE=1 ;;
            --repeat) [ $# -ge 2 ] || usage_error "--repeat needs a count"; set_repeat "$2"; shift ;;
            --repeat=*) set_repeat "${1#*=}" ;;
            -*) usage_error "unknown option '$1'" ;;
            *) break ;;
        esac
        EVIDENCE=1
        shift
    done
    [ $# -eq 0 ] || PACKAGES=("$@")
    for arg in "$@"; do
        case "$arg" in -*) usage_error "option '$arg' after the package list — options come first" ;; esac
    done
}

set_artifacts_dir() {
    local parent
    [ -n "$1" ] || usage_error "--artifacts needs a directory"
    if [ -e "$1" ] || [ -L "$1" ]; then
        usage_error "--artifacts: '$1' already exists; evidence is never overwritten, name a fresh directory"
    fi
    parent="$(dirname "$1")"
    [ -d "$parent" ] || usage_error "--artifacts: parent directory '$parent' does not exist"
    ARTIFACTS_DIR="$(cd "$parent" && pwd)/$(basename "$1")"
}

set_repeat() {
    case "$1" in
        '' | 0* | *[!0-9]*) usage_error "--repeat needs a whole number of at least 1: '$1'" ;;
    esac
    REPEAT="$1"
}

check_remote_dir() {
    local name="${REMOTE_DIR#$REMOTE_PREFIX}"
    case "$REMOTE_DIR" in
        $REMOTE_PREFIX?*) ;;
        *) usage_error "CLAWEE_CI_DIR must be ${REMOTE_PREFIX}<name>: '$REMOTE_DIR'" ;;
    esac
    case "$name" in
        *[!A-Za-z0-9._-]* | *..* | *.) usage_error "CLAWEE_CI_DIR's <name> may hold only letters, digits and ._-, no '..', and may not end in '.': '$REMOTE_DIR'" ;;
    esac
}

check_derived_paths() {
    local p i=0
    set -- "$REMOTE_DIR" "$REMOTE_DIR/go.work" "$DEPS_DIR" "$RUN_BASE" "$RUN_BASE.pid" "$RUN_BASE.log" "$RUN_BASE.rc" "$RUN_BASE.sh"
    while [ "$i" -lt "${#WS_DESTS[@]}" ]; do
        set -- "$@" "${WS_DESTS[$i]}"
        i=$((i + 1))
    done
    for p in "$@"; do
        case "$p" in
            $REMOTE_PREFIX?*) ;;
            *) usage_error "derived remote path is outside ${REMOTE_PREFIX}*: '$p'" ;;
        esac
        case "$p" in
            *..*) usage_error "derived remote path holds '..', which the machine refuses: '$p'" ;;
        esac
    done
}

check_numeric_env() {
    local pair name value
    for pair in "CLAWEE_CI_LOCK_HEARTBEAT_S=$LOCK_HEARTBEAT_S" "CLAWEE_CI_POLL_S=$POLL_S" \
        "CLAWEE_CI_FOLLOW_MAX_MISSES=$FOLLOW_MAX_MISSES"; do
        name="${pair%%=*}"
        value="${pair#*=}"
        case "$value" in
            '' | 0* | *[!0-9]*) usage_error "$name must be a whole number of at least 1: '$value'" ;;
        esac
    done
}

SSH_OPTS=(-o BatchMode=yes -o ConnectTimeout=30 -o ServerAliveInterval=15 -o ServerAliveCountMax=4)
RSYNC_SSH="ssh ${SSH_OPTS[*]}"

remote_n() {
    ssh -n "${SSH_OPTS[@]}" "$MACHINE" "$@"
}

remote_stdin() {
    ssh "${SSH_OPTS[@]}" "$MACHINE" "$@"
}

probe_machine() {
    remote_n true 2>/dev/null && return 0
    warn "$MACHINE is not answering ssh."
    if (exec 3<>"/dev/tcp/$MACHINE/22") 2>/dev/null; then
        warn "port 22 answered — the machine is up; it is loaded, or this account is not enrolled (vm enroll)."
    else
        warn "port 22 did not answer — the machine is stopped; only its owner can start it."
    fi
    exit 1
}

take_lock() {
    printf 'project=%s\nsession=%s\nuser=%s\ntaken=%s\nrepo=%s\nrun=%s\nheartbeat_s=%s\n' "$LOCK_PROJECT" "$LOCK_SESSION" \
        "$(id -un)" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$SRC" "$RUN_ID" "$LOCK_HEARTBEAT_S" |
        remote_stdin "$TAKE_CMD"
}

TAKE_BODY='
            [ -d "$root" ] || exit 5
            if mkdir "$lock" 2>/dev/null; then
                cat > "$lock/holder" && date -u +%Y-%m-%dT%H:%M:%SZ > "$lock/heartbeat" && exit 0
                rm -rf -- "$lock"; exit 4
            fi
            if [ -d "$lock" ]; then
                cat "$lock/holder" 2>/dev/null
                hb=$(cat "$lock/heartbeat" 2>/dev/null)
                if [ -n "$hb" ] && t=$(date -d "$hb" +%s 2>/dev/null); then
                    echo "heartbeat=$hb age=$(( $(date +%s) - t ))s"
                else
                    echo "heartbeat=none age=unknown"
                fi
                exit 3
            fi
            if [ -d "$root" ] && [ -w "$root" ]; then echo changed; exit 3; fi
            stat -c "root %n is %U:%G mode %a" "$root"; exit 4'

refuse_lock() {
    local rc="$1" out="$2" age interval source="holder's" stale rule
    interval="$(printf '%s\n' "$out" | sed -n 's/^heartbeat_s=\([0-9][0-9]*\)$/\1/p' | head -1)"
    [ -n "$interval" ] || { interval="$LOCK_HEARTBEAT_S"; source="this run's"; }
    stale=$((interval * LOCK_STALE_MULTIPLE))
    rule="stale after ${stale}s ($source interval ${interval}s x $LOCK_STALE_MULTIPLE)"
    if [ "$rc" != 3 ]; then
        warn "cannot take $MACHINE:$LOCK — the lock root is not writable by $(id -un):"
        printf '%s\n' "$out" | sed "s|^|$PROG:   |" >&2 2>/dev/null || true
        exit 3
    fi
    if [ "$out" = changed ]; then
        warn "the Clawee CI lock $MACHINE:$LOCK was released while this run checked it — re-run"
        exit 3
    fi
    age="$(printf '%s\n' "$out" | sed -n 's/.* age=\([0-9-]*\)s$/\1/p')"
    warn "the Clawee CI lock $MACHINE:$LOCK is held:"
    printf '%s\n' "$out" | sed "s|^|$PROG:   |" >&2 2>/dev/null || true
    if [ -z "$age" ]; then
        warn "no heartbeat recorded — its age cannot be judged ($rule). Ask the holder; breaking it is an operator decision."
    elif [ "$age" -gt "$stale" ]; then
        warn "STALE — no heartbeat for ${age}s, $rule."
        warn "breaking it is an operator decision; this script never does."
    else
        warn "live — heartbeat ${age}s old, $rule. Wait for it."
    fi
    exit 3
}

heartbeat_cmd() {
    printf 'grep -qx %q %q && date -u +%%Y-%%m-%%dT%%H:%%M:%%SZ > %q' "run=$RUN_ID" "$LOCK/holder" "$LOCK/heartbeat"
}

start_heartbeat() {
    local cmd parent=$$
    cmd="$(heartbeat_cmd)"
    (
        child=""
        trap '[ -z "$child" ] || kill "$child" 2>/dev/null; exit 0' TERM
        while :; do
            sleep "$LOCK_HEARTBEAT_S" & child=$!; wait "$child" || true
            kill -0 "$parent" 2>/dev/null || exit 0
            ssh -n "${SSH_OPTS[@]}" "$MACHINE" "$cmd" >/dev/null 2>&1 & child=$!; wait "$child" || true
        done
    ) &
    HEARTBEAT_PID=$!
}

stop_heartbeat() {
    if [ -n "${HEARTBEAT_PID:-}" ]; then
        kill "$HEARTBEAT_PID" 2>/dev/null || true
        wait "$HEARTBEAT_PID" 2>/dev/null || true
        HEARTBEAT_PID=""
    fi
}

release_lock() {
    local rc=0 KEPT=0
    remote_n "$RELEASE_CMD" 2>/dev/null || rc=$?
    case "$rc" in
        0) say "released $MACHINE:$LOCK" ;;
        3) [ "${1:-}" = quiet ] || warn "lock $MACHINE:$LOCK not released: its holder does not name run $RUN_ID — check it" ;;
        *) warn "could not release $MACHINE:$LOCK (ssh status $rc): it may still be held by run $RUN_ID"
           KEPT=1
           warn "  check the holder:"
           print_command "ssh $MACHINE cat $LOCK/holder"
           warn "  release, only if the holder names run $RUN_ID:"
           print_command "ssh $MACHINE '$RELEASE_CMD'" ;;
    esac
    [ "$KEPT" = 0 ]
}

sync_tree() {
    say "sync $SRC -> $MACHINE:$REMOTE_DIR"
    rsync -a -e "$RSYNC_SSH" --rsync-path="$TREE_RSYNC_PATH" --delete --exclude '.git' --exclude '.codegraph' \
        --exclude '/dist' --exclude 'go.work' --exclude 'go.work.sum' "$SRC/" "$MACHINE:$REMOTE_DIR/" ||
        { warn "rsync of $SRC to $MACHINE failed"; return 1; }
    if [ -f "$SRC/go.work" ]; then
        mirror_workspace || return 1
    else
        remote_n "$GOWORK_RM_CMD" ||
            { warn "could not remove a stale go.work on $MACHINE"; return 1; }
    fi
}

workspace_uses() {
    awk '
        /^[[:space:]]*use[[:space:]]*\(/ { inblock = 1; next }
        inblock && /^[[:space:]]*\)/     { inblock = 0; next }
        inblock                          { print $1; next }
        /^[[:space:]]*use[[:space:]]+/   { print $2 }
    ' "$SRC/go.work"
}

workspace_module() {
    local module
    module="$(awk '$1 == "module" { print $2; exit }' "$1/go.mod" 2>/dev/null)"
    case "$module" in
        '') warn "$1/go.mod declares no module path"; return 1 ;;
        *[!A-Za-z0-9._~/-]*) warn "$1/go.mod declares an unusable module path: '$module'"; return 1 ;;
    esac
    case "/$module/" in
        *//* | */./* | *..*) warn "$1/go.mod declares an unusable module path: '$module'"; return 1 ;;
    esac
    printf '%s' "$module"
}

plan_workspace() {
    local u dep module dest
    [ -f "$SRC/go.work" ] || return 0
    for u in $(workspace_uses); do
        [ "$u" = "." ] && continue
        case "$u" in /*) dep="$u" ;; *) dep="$SRC/$u" ;; esac
        dep="$(cd "$dep" 2>/dev/null && pwd)" || { warn "go.work names '$u', which is not a directory"; return 1; }
        module="$(workspace_module "$dep")" || return 1
        dest="$DEPS_DIR/$(printf '%s' "$module" | tr '/' '_')"
        WS_DIRS+=("$dep"); WS_MODULES+=("$module"); WS_DESTS+=("$dest")
        WS_RSYNC_PATHS+=("$(remote_guard "$dest" "$REMOTE_PREFIX?*")rsync")
    done
}

mirror_workspace() {
    local i=0 uses="" go_directive
    go_directive="$(awk '$1 == "go" { print $2; exit }' "$SRC/go.work")"
    remote_n "$(printf 'mkdir -p %q' "$DEPS_DIR")" || { warn "could not create $DEPS_DIR on $MACHINE"; return 1; }
    while [ "$i" -lt "${#WS_DIRS[@]}" ]; do
        rsync -a -e "$RSYNC_SSH" --rsync-path="${WS_RSYNC_PATHS[$i]}" --delete --exclude '.git' --exclude '.codegraph' \
            --exclude '/dist' --exclude 'go.work' --exclude 'go.work.sum' "${WS_DIRS[$i]}/" "$MACHINE:${WS_DESTS[$i]}/" ||
            { warn "rsync of ${WS_DIRS[$i]} to $MACHINE failed"; return 1; }
        uses="$uses	${WS_DESTS[$i]}
"
        say "go.work: ${WS_MODULES[$i]} <- ${WS_DIRS[$i]}"
        i=$((i + 1))
    done
    printf 'go %s\n\nuse (\n\t.\n%s)\n' "${go_directive:-1.25.1}" "$uses" |
        remote_stdin "$GOWORK_WRITE_CMD" ||
        { warn "could not write go.work on $MACHINE"; return 1; }
}

go_test_flags() {
    local flags="-count=$REPEAT"
    [ "$SHUFFLE" = 0 ] || flags="$flags -shuffle=on"
    [ "$EVIDENCE" = 0 ] || flags="$flags -json"
    if [ -n "$ARTIFACTS_DIR" ]; then
        flags="$flags -covermode=set -coverpkg=./... -coverprofile=$(printf %q "$RUN_BASE.cover.out")"
    fi
    printf '%s' "$flags"
}

remote_suite() {
    local pkgs flags
    pkgs="$(printf '%q ' "${PACKAGES[@]}")"
    flags="$(go_test_flags)"
    remote_guard "$RUN_BASE" "$REMOTE_PREFIX?*"
    printf '\n'
    printf 'echo $$ > %q\n' "$RUN_BASE.pid"
    printf '( while grep -qx %q %q 2>/dev/null; do date -u +%%Y-%%m-%%dT%%H:%%M:%%SZ > %q; sleep %q; done ) &\nbeat=$!\n' \
        "run=$RUN_ID" "$LOCK/holder" "$LOCK/heartbeat" "$LOCK_HEARTBEAT_S"
    printf 'export GOTOOLCHAIN=auto TMPDIR=/tmp\nrc=0\n'
    printf 'if ! cd %q; then echo "=== cannot cd to %q"; rc=1\n' "$REMOTE_DIR" "$REMOTE_DIR"
    printf 'elif [ %q = 1 ] && ! command -v jq >/dev/null; then echo "=== jq is not installed; the evidence mode needs it"; rc=1\n' "$EVIDENCE"
    printf 'else\necho "=== go build ./..."\n'
    printf 'if go build ./...; then\n'
    if [ "$EVIDENCE" = 0 ]; then
        printf 'echo %q\n' "=== go test $flags $pkgs"
        printf 'go test %s %s || { rc=$?; echo "=== test FAILED ($rc)"; }\n' "$flags" "$pkgs"
    else
        evidence_test_cmd "$flags" "$pkgs"
    fi
    printf 'else rc=$?; echo "=== build FAILED ($rc)"\nfi\nfi\n'
    printf 'kill $beat 2>/dev/null\necho $rc > %q && mv -- %q %q\nrm -f -- %q\nexit $rc\n' \
        "$RUN_BASE.rc.tmp" "$RUN_BASE.rc.tmp" "$RUN_BASE.rc" "$RUN_BASE.pid"
}

evidence_test_cmd() {
    local flags="$1" pkgs="$2" has_tests='{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}'
    if [ -z "$ARTIFACTS_DIR" ]; then
        printf 'set -- %s\n' "$pkgs"
    else
        printf 'notest=$(go list -f %q %s | grep . | tr "\\n" " ")\n' '{{if not (or .TestGoFiles .XTestGoFiles)}}{{.ImportPath}}{{end}}' "$pkgs"
        printf '[ -z "$notest" ] || echo "=== no test files, left out of the coverage run: $notest"\n'
        printf 'set -- $(go list -f %q %s)\n' "$has_tests" "$pkgs"
    fi
    printf 'if [ $# -eq 0 ]; then echo "=== no named package has test files"; rc=1\nelse\n'
    printf 'echo "=== go test %s $*"\n' "$flags"
    printf 'go test %s "$@" </dev/null | tee %q | %s\n' "$flags" "$RUN_BASE.json" "$(evidence_filter_cmd)"
    printf 'rc=${PIPESTATUS[0]}; [ $rc -eq 0 ] || echo "=== test FAILED ($rc)"\n'
    evidence_summary_cmd
    printf 'fi\n'
}

evidence_filter_cmd() {
    printf 'jq --unbuffered -Rrj %q' '(fromjson? // {Action: "output", Output: (. + "\n")})
        | select(.Action == "output" or .Action == "build-output") | .Output
        | select(test("^\\s*(=== (RUN|PAUSE|CONT|NAME)|--- PASS)") | not)'
}

evidence_summary_cmd() {
    local program
    program='def n($x; $a): $x | map(select(.Action == $a)) | length;
        def tally($x): "\($x | length) (pass \(n($x; "pass")), fail \(n($x; "fail")), skip \(n($x; "skip")))";
        def line($x; $label): "=== \($label): tests \(tally($x | map(select(.Test | contains("/") | not)))) · cases \(tally($x | map(select(.Test | contains("/")))))";
        [inputs | fromjson? | select(type == "object")] as $all
        | ($all | map(select(.Test != null and (.Action == "pass" or .Action == "fail" or .Action == "skip")))) as $e
        | ($e | group_by(.Package)[] | line(.; .[0].Package)),
          line($e; "module"),
          ($e[] | select(.Action == "skip") | "=== skip " + .Package + " " + .Test),
          ($e[] | select(.Action == "fail") | "=== fail " + .Package + " " + .Test),
          ($all[] | select(.Action == "output" and .Test == null and (.Output | startswith("-test.shuffle ")))
            | "=== shuffle " + .Package + " " + (.Output | rtrimstr("\n")))'
    printf 'echo "=== summary"\njq -Rrn %q < %q\n' "$program" "$RUN_BASE.json"
    [ -n "$ARTIFACTS_DIR" ] || return 0
    printf 'if [ -s %q ]; then\n' "$RUN_BASE.cover.out"
    printf 'awk %q %q | LC_ALL=C sort -u > %q\n' 'FNR > 1 && $NF > 0 && $1 !~ /_test\// {print $1}' "$RUN_BASE.cover.out" "$RUN_BASE.covered.txt"
    printf 'echo "=== covered $(wc -l < %q) blocks"\n' "$RUN_BASE.covered.txt"
    printf 'else\necho "=== no coverage profile"; [ $rc -ne 0 ] || rc=1\nfi\n'
}

launch() {
    printf '%s\n' "$SUITE_SCRIPT" | remote_stdin "$LAUNCH_CMD"
}

LAUNCH_BODY='
            rm -f -- "$base".* || exit 1
            cat > "$base.sh" || exit 1
            setsid nohup bash "$base.sh" > "$base.log" 2>&1 < /dev/null &
            echo $! > "$base.pid"'

poll_cmd() {
    printf 'base=%q off=%q; ' "$RUN_BASE" "$1"
    printf '%s' 'alive=dead; [ -f "$base.pid" ] && kill -0 "$(cat "$base.pid")" 2>/dev/null && alive=alive; '
    printf '%s' 'rc=$(cat "$base.rc" 2>/dev/null); size=$(stat -c %s "$base.log" 2>/dev/null || echo 0); '
    printf '%s' 'echo "$alive $rc $size"; tail -c +$((off + 1)) "$base.log" 2>/dev/null | head -c $((size - off)); printf .'
}

follow() {
    local offset=0 misses=0 dead=0 out header alive rc size cmd polled
    while :; do
        cmd="$(poll_cmd "$offset")"
        ssh -n "${SSH_OPTS[@]}" "$MACHINE" "$cmd" >"$POLL_OUT" 2>/dev/null & FOLLOW_CHILD=$!
        polled=0; wait "$FOLLOW_CHILD" || polled=$?
        FOLLOW_CHILD=""
        if [ "$polled" = 0 ] && out="$(cat "$POLL_OUT")"; then
            misses=0
            header="${out%%$'\n'*}"
            out="${out#*$'\n'}"
            emit "${out%.}"
            read -r alive rc size <<<"$header"
            if [ -z "$size" ]; then size="$rc"; rc=""; fi
            offset="$size"
            if [ -n "$rc" ]; then FOLLOW_OUTCOME=status; return "$rc"; fi
            if [ "$alive" = dead ]; then dead=$((dead + 1)); else dead=0; fi
            if [ "$dead" -ge "$DEAD_POLLS" ]; then
                warn "the runner on $MACHINE is gone and wrote no status — killed? log $RUN_BASE.log"
                FOLLOW_OUTCOME=dead; return 1
            fi
        else
            misses=$((misses + 1))
            warn "lost contact with $MACHINE (poll $misses) — the run continues there"
            if [ "$misses" -ge "$FOLLOW_MAX_MISSES" ]; then
                warn "giving up following; log $MACHINE:$RUN_BASE.log"
                FOLLOW_OUTCOME=lost; return 1
            fi
        fi
        sleep "$POLL_S" & FOLLOW_CHILD=$!
        wait "$FOLLOW_CHILD" || true
        FOLLOW_CHILD=""
    done
}

stop_runner_cmd() {
    remote_guard "$RUN_BASE.pid" "$REMOTE_PREFIX?*"
    printf 'pidf=%q script=%q; ' "$RUN_BASE.pid" "$RUN_BASE.sh"
    printf '%s' '
        if [ -f "$pidf" ]; then
            pg=$(cat "$pidf")
        else
            pid=$(pgrep -f -- "^bash $script\$"); r=$?
            [ "$r" -eq 1 ] && exit 0
            [ "$r" -eq 0 ] || exit 1
            pg=$(ps -o pgid= -p "${pid%%[!0-9]*}" | tr -d " ")
        fi
        case "$pg" in ""|0*|*[!0-9]*|1) echo unusable process group: "[$pg]" >&2; exit 1 ;; esac
        kill -TERM -- "-$pg" 2>/dev/null
        for i in $(seq 1 20); do
            pgrep -g "$pg" >/dev/null; r=$?
            [ "$r" -eq 1 ] && { rm -f -- "$pidf"; exit 0; }
            [ "$r" -eq 0 ] || exit 1
            sleep 1
        done
        kill -KILL -- "-$pg" 2>/dev/null; sleep 1
        pgrep -g "$pg" >/dev/null; [ "$?" -eq 1 ] || exit 1
        rm -f -- "$pidf"'
}

stop_runner() {
    remote_n "$STOP_CMD" 2>/dev/null
}

remote_guard() {
    printf 'case %q in %s) ;; *) echo refused, outside the runner prefix: %q >&2; exit 64 ;; esac; ' "$1" "$2" "$1"
    printf 'case %q in *..*) echo refused, path holds dot-dot: %q >&2; exit 64 ;; esac; ' "$1" "$1"
}

build_teardown_cmds() {
    local gone="$LOCK.released.$RUN_ID" run_pattern="$REMOTE_PREFIX?*"
    STOP_CMD="$(stop_runner_cmd)"
    RELEASE_CMD="$(remote_guard "$LOCK" /tmp/ci-lock/clawee)$(remote_guard "$gone" '/tmp/ci-lock/clawee.released.?*')"
    RELEASE_CMD="$RELEASE_CMD$(printf 'grep -qx %q %q 2>/dev/null || exit 3; mv -- %q %q && rm -rf -- %q' \
        "run=$RUN_ID" "$LOCK/holder" "$LOCK" "$gone" "$gone")"
    TAKE_CMD="$(remote_guard "$LOCK" /tmp/ci-lock/clawee)$(printf 'root=%q lock=%q; ' "$LOCK_ROOT" "$LOCK")$TAKE_BODY"
    CLEAN_CMD="$(remote_guard "$REMOTE_DIR" "$run_pattern")rm -f -- $(printf %q "$REMOTE_DIR").run.*"
    GOWORK_RM_CMD="$(remote_guard "$REMOTE_DIR" "$run_pattern")$(printf 'rm -f -- %q %q' "$REMOTE_DIR/go.work" "$REMOTE_DIR/go.work.sum")"
    GOWORK_WRITE_CMD="$(remote_guard "$REMOTE_DIR/go.work" "$run_pattern")$(printf 'cat > %q' "$REMOTE_DIR/go.work")"
    TREE_RSYNC_PATH="$(remote_guard "$REMOTE_DIR" "$run_pattern")rsync"
    LAUNCH_CMD="$(remote_guard "$RUN_BASE" "$run_pattern")$(printf 'base=%q; ' "$RUN_BASE")$LAUNCH_BODY"
    SUITE_SCRIPT="$(remote_suite)"
    RUN_CHECK_CMD="$(printf 'ls -l %q.pid %q.rc %q.log; tail -5 %q.log; if [ -f %q.pid ]; then pg=$(cat %q.pid); case "$pg" in ""|0*|*[!0-9]*|1) echo "unusable process group: [$pg]"; pgrep -af "^bash %q.sh\\$" ;; *) pgrep -ag "$pg" ;; esac; else pgrep -af "^bash %q.sh\\$"; fi' \
        "$RUN_BASE" "$RUN_BASE" "$RUN_BASE" "$RUN_BASE" "$RUN_BASE" "$RUN_BASE" "$RUN_BASE" "$RUN_BASE")"
}

fetch_artifacts() {
    local pair from to failed=0
    mkdir "$ARTIFACTS_DIR" || { warn "could not create $ARTIFACTS_DIR"; return 1; }
    for pair in json:test.json cover.out:cover.out covered.txt:covered.txt; do
        from="$RUN_BASE.${pair%%:*}"
        to="$ARTIFACTS_DIR/${pair#*:}"
        if (remote_n "$(printf 'cat -- %q' "$from")") >"$to.part" && mv "$to.part" "$to"; then
            continue
        fi
        rm -f "$to.part"
        warn "could not copy $MACHINE:$from to $to"
        failed=1
    done
    [ "$failed" = 0 ] || return 1
    say "evidence copied to $ARTIFACTS_DIR:"
    (cd "$ARTIFACTS_DIR" && wc -c test.json cover.out covered.txt | sed "s|^|$PROG:   |") 2>/dev/null || true
}

report_kept_lock() {
    warn "could not confirm the run on $MACHINE is gone — $MACHINE:$LOCK is left held (it goes STALE); not released"
    warn "  run id:  $RUN_ID"
    warn "  files:   $MACHINE:$RUN_BASE.{pid,log,rc}"
    warn "  check what is left of the run:"
    print_command "ssh $MACHINE '$RUN_CHECK_CMD'"
    warn "  release, only once the check shows nothing left running:"
    print_command "ssh $MACHINE '$RELEASE_CMD'"
}

cleanup() {
    local rc=$?
    echo >/dev/null
    set +e
    trap '' INT TERM HUP PIPE
    trap - EXIT
    [ "$rc" != 141 ] || stdout_closed
    [ -z "${FOLLOW_CHILD:-}" ] || kill "$FOLLOW_CHILD" 2>/dev/null
    [ -z "${POLL_OUT:-}" ] || rm -f "$POLL_OUT"
    if [ "${LAUNCHED:-0}" = 1 ] && [ "$FOLLOW_OUTCOME" != status ] && ! stop_runner; then
        stop_heartbeat
        report_kept_lock
        exit "$(( rc == 0 ? 1 : rc ))"
    fi
    stop_heartbeat
    case "$LOCK_STATE" in
        taken) release_lock || [ "$rc" -ne 0 ] || rc=1 ;;
        trying) release_lock quiet || [ "$rc" -ne 0 ] || rc=1 ;;
    esac
    exit "$rc"
}

run_suite() {
    local rc=0
    remote_n "$CLEAN_CMD" || true
    sync_tree || return 1
    LAUNCHED=1
    if ! launch; then
        warn "could not start the suite on $MACHINE"
        FOLLOW_OUTCOME=unstarted
        return 1
    fi
    say "started on $MACHINE (log $RUN_BASE.log)"
    follow || rc=$?
    [ "$rc" -eq 0 ] || rc=1
    if [ "$FOLLOW_OUTCOME" = status ] && [ -n "$ARTIFACTS_DIR" ] && ! fetch_artifacts; then
        [ "$rc" -ne 0 ] || rc=1
    fi
    [ "$FOLLOW_OUTCOME" = status ] || [ "$rc" -ne 0 ] || rc=1
    return "$rc"
}

main() {
    local lock_out lock_rc=0 rc=0
    parse_args "$@"
    check_remote_dir
    check_numeric_env
    plan_workspace || exit 2
    check_derived_paths
    build_teardown_cmds
    POLL_OUT="$(mktemp "${TMPDIR:-/tmp}/ght-ci-poll.XXXXXX")" || { warn "could not create a temporary file"; exit 1; }
    trap 'rm -f "$POLL_OUT"' EXIT
    trap 'exit 130' INT
    trap 'exit 143' TERM
    trap 'exit 129' HUP
    trap 'exit 141' PIPE
    probe_machine
    trap cleanup EXIT
    LOCK_STATE=trying
    lock_out="$(take_lock)" || lock_rc=$?
    case "$lock_rc" in
        0) LOCK_STATE=taken; say "took $MACHINE:$LOCK (project $LOCK_PROJECT, session $LOCK_SESSION)" ;;
        3|4) LOCK_STATE=""; refuse_lock "$lock_rc" "$lock_out" ;;
        5) LOCK_STATE=""
           warn "$MACHINE:$LOCK_ROOT does not exist — the machine's tmpfiles.d entry creates it; this script never does. Ask the machine's owner to run systemd-tmpfiles --create."
           exit 1 ;;
        *) warn "could not reach $MACHINE to take the lock (ssh $lock_rc)"; exit 1 ;;
    esac
    start_heartbeat
    run_suite || rc=$?
    exit "$rc"
}

main "$@"
