#!/usr/bin/env bash
set -euo pipefail

PROG="ci/run-tests.sh"
MACHINE="${CI_MACHINE:-masdetta-ci}"
SRC="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

REMOTE_PREFIX="/tmp/clawee-ght-"
SEED_DIR="${CLAWEE_CI_DIR:-$REMOTE_PREFIX$(id -un)-$(printf '%s' "$SRC" | cksum | cut -d' ' -f1)}"
SEED_DEPS_DIR="$SEED_DIR.deps"
USER_TAG="$(id -un | tr -c 'A-Za-z0-9\n' '_')"
RUN_ID="$USER_TAG-$$-$(date +%s)"
case "$RUN_ID" in "$USER_TAG"-[0-9]*-[0-9]*) ;; *) echo "$PROG: could not form a run id ('$RUN_ID'); refusing to run" >&2; exit 1 ;; esac
REMOTE_DIR="$SEED_DIR.t-$RUN_ID"
DEPS_DIR="$REMOTE_DIR.deps"
RUN_DIR="$REMOTE_DIR.run"
RUN_BASE="$RUN_DIR/run"

CI_LOCK_PRODUCT=clawee-go-headless-term
CI_LOCK_BIN=/usr/local/bin/ci-lock
SUITE_BOUND_S=600
LOCK_PROJECT="${CLAWEE_CI_LOCK_PROJECT:-$(git -C "$SRC" rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)}"
LOCK_SESSION="${CLAWEE_CI_LOCK_SESSION:-unrecorded}"
POLL_S="${CLAWEE_CI_POLL_S:-3}"
FOLLOW_MAX_MISSES="${CLAWEE_CI_FOLLOW_MAX_MISSES:-40}"
DEAD_POLLS=3

EVIDENCE=0
ARTIFACTS_DIR=""
SHUFFLE=0
REPEAT=1
PACKAGES=(./...)
TREE_TOUCHED=0
LOCK_EXIT=""
WS_DIRS=()
WS_MODULES=()
WS_DESTS=()
WS_SEED_DESTS=()
WS_RSYNC_PATHS=()
FOLLOW_OUTCOME=""

IFS= read -r -d '' USAGE_TEXT <<'USAGE' || true
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
  CI_MACHINE                    the machine (default masdetta-ci)
  CI_NO_AUTOSTART               set: do not run ci-watch ensure before the first ssh
  CLAWEE_CI_DIR                 the checkout's seed tree (default
                                /tmp/clawee-ght-<user>-<cksum of this checkout>);
                                must be /tmp/clawee-ght-<name>, <name> of letters,
                                digits and ._- with no '..' and not ending in '.'.
                                Each run syncs into its own tree <seed>.t-<run id>,
                                seeded from the seed (rsync --copy-dest: own
                                files), and removes it when it ends
  CLAWEE_CI_LOCK_PROJECT        project recorded on the lock (default: the branch)
  CLAWEE_CI_LOCK_SESSION        session recorded on the lock (default: unrecorded)
  CLAWEE_CI_POLL_S              seconds between log polls (default 3)
  CLAWEE_CI_FOLLOW_MAX_MISSES   failed polls in a row before following gives up (default 40)
The two numbers must be whole numbers of at least 1; anything else is a
usage error before the machine is contacted.

The CI lock: the suite runs on the machine under
  ci-lock run clawee-go-headless-term --timeout 600 --project <p> --session <s> -- …
which holds the clawee-go-headless-term product lock and a shared hold on
clawee for as long as the run's processes live. A held lock is waited on, up
to the suite bound (go test's default -timeout, 600s); ci-lock's waiting and
acquired lines are in the followed log, and `ci-lock status
clawee-go-headless-term` on the machine names the holder. Other Clawee
products run beside it. The run gets its own user scope where the machine's
user manager runs; its end stops the group and the scope, then kills (TERM,
then KILL) every process still carrying CLAWEE_CI_RUN_ID=<run id>.

Exit status (a closed stderr never changes it):
  0        the build and every test passed
  1        the build or a test failed (any non-zero status from go is reported
           as 1, so 2 and 75 below always mean this script), or the machine
           could not be reached, the tree not synced, clawee-go-headless-term
           not provisioned on the machine (the operator runs `ci-lock
           install`), the run ended without a status, or its evidence could
           not be copied home — or the suite passed but its stop could not be
           confirmed (the check command is on stderr)
  2        usage error, refused before any contact: bad option, option after
           packages, existing --artifacts directory, bad environment value
           (CLAWEE_CI_DIR outside /tmp/clawee-ght-<name>, or a <name> ending
           in '.'), a derived remote path the machine guard would refuse, or
           a local go.work whose `use` is not a directory or names a module
           that cannot be mirrored safely (empty, a '.' segment, '..')
  75       the clawee-go-headless-term lock was not acquired within the suite
           bound
  130/143/129  interrupted by INT/TERM/HUP; 141  stdout closed with SIGPIPE.
           The run is stopped first. While the run is followed (the wait for
           the lock included) a signal is acted on at once, also when it is
           sent to this script's pid alone. These remote steps are bounded and
           finish before the signal is acted on: the probe, the stale-tree
           cleanup, the sync, the launch and the evidence copy-back. The stop
           that follows the signal can itself take up to about 35 s.
Once the teardown (stop, then the tree's removal) has begun, further signals
are ignored and the exit is the run's own status: a signal replaces the run's
status only when it lands before the teardown, the copy-back included.
A closed stdout WITHOUT SIGPIPE (`>&-`, or a caller that ignores SIGPIPE) does
not stop the run: messages continue on stderr and the status is the run's.
USAGE

usage() {
    printf '%s' "$USAGE_TEXT"
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
    local name="${SEED_DIR#$REMOTE_PREFIX}"
    case "$SEED_DIR" in
        $REMOTE_PREFIX?*) ;;
        *) usage_error "CLAWEE_CI_DIR must be ${REMOTE_PREFIX}<name>: '$SEED_DIR'" ;;
    esac
    case "$name" in
        *[!A-Za-z0-9._-]* | *..* | *.) usage_error "CLAWEE_CI_DIR's <name> may hold only letters, digits and ._-, no '..', and may not end in '.': '$SEED_DIR'" ;;
    esac
}

check_derived_paths() {
    local p i=0
    set -- "$SEED_DIR" "$SEED_DEPS_DIR" "$REMOTE_DIR" "$REMOTE_DIR/go.work" "$DEPS_DIR" "$RUN_DIR" "$RUN_BASE.sh"
    while [ "$i" -lt "${#WS_DESTS[@]}" ]; do
        set -- "$@" "${WS_DESTS[$i]}" "${WS_SEED_DESTS[$i]}"
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
    for pair in "CLAWEE_CI_POLL_S=$POLL_S" "CLAWEE_CI_FOLLOW_MAX_MISSES=$FOLLOW_MAX_MISSES"; do
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

ensure_machine() {
    [ -z "${CI_NO_AUTOSTART:-}" ] || return 0
    ci-watch ensure "$MACHINE" || { warn "ci-watch could not bring $MACHINE up."; exit 1; }
}

probe_machine() {
    ensure_machine
    remote_n true 2>/dev/null && return 0
    warn "$MACHINE is not answering ssh."
    if (exec 3<>"/dev/tcp/$MACHINE/22") 2>/dev/null; then
        warn "port 22 answered — the machine is up; it is loaded, or this account is not enrolled (vm enroll)."
    else
        warn "port 22 did not answer — the machine is stopped; only its owner can start it."
    fi
    exit 1
}

sync_tree() {
    say "sync $SRC -> $MACHINE:$REMOTE_DIR"
    rsync -a -e "$RSYNC_SSH" --rsync-path="$TREE_RSYNC_PATH" --delete --exclude '.git' --exclude '.codegraph' \
        --copy-dest="$SEED_DIR" --exclude '/dist' --exclude 'go.work' --exclude 'go.work.sum' "$SRC/" "$MACHINE:$REMOTE_DIR/" ||
        { warn "rsync of $SRC to $MACHINE failed"; return 1; }
    if [ -f "$SRC/go.work" ]; then
        mirror_workspace || return 1
    fi
    remote_n "$SEED_REFRESH_CMD" ||
        warn "could not refresh the seed $MACHINE:$SEED_DIR from this run's tree; the next run copies more, nothing else changes"
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
        WS_SEED_DESTS+=("$SEED_DEPS_DIR/${dest##*/}")
        WS_RSYNC_PATHS+=("$(remote_guard "$dest" "$REMOTE_PREFIX?*")rsync")
    done
}

mirror_workspace() {
    local i=0 uses="" go_directive
    go_directive="$(awk '$1 == "go" { print $2; exit }' "$SRC/go.work")"
    remote_n "$(printf 'mkdir -p %q' "$DEPS_DIR")" || { warn "could not create $DEPS_DIR on $MACHINE"; return 1; }
    while [ "$i" -lt "${#WS_DIRS[@]}" ]; do
        rsync -a -e "$RSYNC_SSH" --rsync-path="${WS_RSYNC_PATHS[$i]}" --delete --exclude '.git' --exclude '.codegraph' \
            --copy-dest="${WS_SEED_DESTS[$i]}" --exclude '/dist' --exclude 'go.work' --exclude 'go.work.sum' "${WS_DIRS[$i]}/" "$MACHINE:${WS_DESTS[$i]}/" ||
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
    remote_guard "$RUN_DIR" "$REMOTE_PREFIX?*"
    printf '\n'
    printf ': > %q\n' "$RUN_DIR/started"
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
    printf 'echo $rc > %q && mv -- %q %q\nexit $rc\n' "$RUN_BASE.rc.tmp" "$RUN_BASE.rc.tmp" "$RUN_BASE.rc"
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

LOCKED_RUN="$CI_LOCK_BIN run $CI_LOCK_PRODUCT --timeout $SUITE_BOUND_S --project \"\$1\" --session \"\$2\" -- bash \"\$3/run.sh\"; c=\$?; [ -f \"\$3/run.rc\" ] || [ -f \"\$3/started\" ] || { echo \"\$c\" > \"\$3/lock_rc.tmp\" && mv -- \"\$3/lock_rc.tmp\" \"\$3/lock_rc\"; }; exit \$c"
EMPTY_ID_REFUSAL='[ -n "$id" ] || { echo "ci/run-tests.sh: refusing to act on an empty run id" >&2; exit 5; }; '
LAUNCH_BODY='
            rm -rf -- "$run" && mkdir -m 700 "$run" && echo "$id" > "$run/id" || exit 1
            cat > "$run/run.sh" || exit 1
            export CLAWEE_CI_RUN_ID="$id"
            scope=""
            case "$(systemctl --user is-system-running 2>/dev/null)" in
                running | degraded) command -v systemd-run >/dev/null && scope="clawee-go-headless-term-run-$id.scope" ;;
            esac
            printf "%s\n" "$scope" > "$run/scope"
            setsid nohup ${scope:+systemd-run --user --scope --quiet --collect --unit="$scope" --} bash -c "$locked" ci-lock-run "$project" "$session" "$run" > "$run/run.log" 2>&1 < /dev/null &
            echo $! > "$run/run.pid"'

poll_cmd() {
    printf 'run=%q off=%q; ' "$RUN_DIR" "$1"
    printf '%s' 'alive=dead; [ -f "$run/run.pid" ] && kill -0 "$(cat "$run/run.pid")" 2>/dev/null && alive=alive; '
    printf '%s' 'rc=$(cat "$run/run.rc" 2>/dev/null); [ -n "$rc" ] || { [ -f "$run/lock_rc" ] && rc="lock$(cat "$run/lock_rc")"; }; '
    printf '%s' 'size=$(stat -c %s "$run/run.log" 2>/dev/null || echo 0); '
    printf '%s' 'echo "$alive $rc $size"; tail -c +$((off + 1)) "$run/run.log" 2>/dev/null | head -c $((size - off)); printf .'
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
            case "$rc" in lock*) FOLLOW_OUTCOME=lock; LOCK_EXIT="${rc#lock}"; return 1 ;; esac
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
    remote_guard "$RUN_DIR" "$REMOTE_PREFIX?*"
    printf 'run=%q id=%q; %s' "$RUN_DIR" "$RUN_ID" "$EMPTY_ID_REFUSAL"
    printf '%s' '
proc=/proc
survivors() { grep -lzxF -- "CLAWEE_CI_RUN_ID=$id" "$proc"/[0-9]*/environ 2>/dev/null | sed "s|^$proc/||; s|/environ\$||" | grep -vx "$$"; }
unconfirmed=0
if grep -qxF -- "$id" "$run/id" 2>/dev/null; then
    pg=$(cat "$run/run.pid" 2>/dev/null)
    case "$pg" in "" | 0* | *[!0-9]* | 1) pg="" ;; esac
    if [ -n "$pg" ]; then
        pgrep -g "$pg" >/dev/null; r=$?
        if [ "$r" -eq 0 ]; then
            echo "ci/run-tests.sh: stopping the run on the machine (process group $pg)"
            kill -TERM -- "-$pg" 2>/dev/null
            for i in $(seq 1 20); do pgrep -g "$pg" >/dev/null; r=$?; [ "$r" -eq 0 ] || break; sleep 1; done
            if [ "$r" -eq 0 ]; then kill -KILL -- "-$pg" 2>/dev/null; sleep 1; pgrep -g "$pg" >/dev/null; r=$?; fi
        fi
        [ "$r" -eq 1 ] || { echo "ci/run-tests.sh: process group $pg is not confirmed gone (pgrep $r)"; unconfirmed=1; }
    fi
    unit=$(cat "$run/scope" 2>/dev/null)
    [ "$unit" != "clawee-go-headless-term-run-$id.scope" ] || systemctl --user stop "$unit" 2>/dev/null
fi
left=$(survivors)
[ -n "$left" ] || exit "$unconfirmed"
for p in $left; do echo "ci/run-tests.sh: survivor of run $id outside its group: pid $p $(tr "\0" " " < "$proc/$p/cmdline" 2>/dev/null)"; done
kill -TERM $left 2>/dev/null
for i in $(seq 1 10); do left=$(survivors); [ -n "$left" ] || break; sleep 1; done
if [ -n "$left" ]; then kill -KILL $left 2>/dev/null; sleep 1; left=$(survivors); fi
[ -z "$left" ] || { echo "ci/run-tests.sh: run $id still has processes after KILL: $left"; exit 1; }
echo "ci/run-tests.sh: killed the survivors of run $id (TERM, then KILL after 10s)"
exit "$unconfirmed"'
}

prep_cmd() {
    remote_guard "$SEED_DIR" "$REMOTE_PREFIX?*"
    remote_guard "$SEED_DEPS_DIR" "$REMOTE_PREFIX?*"
    printf '\nseed=%q seeddeps=%q me=%q\n' "$SEED_DIR" "$SEED_DEPS_DIR" "$USER_TAG"
    printf '%s' 'mkdir -p -- "$seed" "$seeddeps" || exit 1
proc=/proc
for t in "$seed".t-?*; do
    id=${t#"$seed".t-}
    case "$id" in *[!A-Za-z0-9_-]*) continue ;; "$me"-[0-9]*-[0-9]*) ;; *) continue ;; esac
    [ -d "$t" ] && [ -O "$t" ] || continue
    grep -qxF -- "$id" "$t.run/id" 2>/dev/null || continue
    [ -n "$(find "$t.run/id" -mmin +1 2>/dev/null)" ] || continue
    grep -qzxF -- "CLAWEE_CI_RUN_ID=$id" "$proc"/[0-9]*/environ 2>/dev/null && continue
    rm -rf -- "$t" "$t.run" "$t.deps" && echo "ci/run-tests.sh: removed the finished run tree $t (run $id) and its .run / .deps"
done
exit 0'
}

stop_runner() {
    local attempt rc
    for attempt in 1 2 3; do
        rc=0
        remote_n "$STOP_CMD" || rc=$?
        [ "$rc" = 255 ] || return "$rc"
        warn "$MACHINE refused the stop (attempt $attempt of 3)"
        [ "$attempt" = 3 ] || sleep 10
    done
    return 1
}

remove_tree() {
    local attempt
    for attempt in 1 2 3; do
        remote_n "$REMOVE_CMD" && return 0
        [ "$attempt" = 3 ] || sleep 5
    done
    warn "could not remove this run's tree on $MACHINE; remove it by hand:"
    print_command "ssh $MACHINE '$REMOVE_CMD'"
}

lock_failed() {
    case "$1" in
        75) warn "the $CI_LOCK_PRODUCT lock was not acquired within ${SUITE_BOUND_S}s; ssh $MACHINE ci-lock status $CI_LOCK_PRODUCT names the holder"
            return 75 ;;
        2) warn "$CI_LOCK_PRODUCT is not provisioned on $MACHINE; the operator runs \`ci-lock install\`" ;;
        *) warn "ci-lock exited $1 before the suite ran; its lines are in the log above" ;;
    esac
    return 1
}

remote_guard() {
    printf 'case %q in %s) ;; *) echo refused, outside the runner prefix: %q >&2; exit 64 ;; esac; ' "$1" "$2" "$1"
    printf 'case %q in *..*) echo refused, path holds dot-dot: %q >&2; exit 64 ;; esac; ' "$1" "$1"
}

build_teardown_cmds() {
    local run_pattern="$REMOTE_PREFIX?*"
    STOP_CMD="$(stop_runner_cmd)"
    PREP_CMD="$(prep_cmd)"
    REMOVE_CMD="$(printf 't=%q id=%q; %s' "$REMOTE_DIR" "$RUN_ID" "$EMPTY_ID_REFUSAL")$(remote_guard "$REMOTE_DIR" "$run_pattern")"
    REMOVE_CMD="$REMOVE_CMD"'case "$t" in *.t-"$id") ;; *) echo "ci/run-tests.sh: $t is not the tree of run $id" >&2; exit 5 ;; esac; rm -rf -- "$t" "$t.run" "$t.deps"'
    SEED_REFRESH_CMD="$(remote_guard "$REMOTE_DIR" "$run_pattern")$(remote_guard "$SEED_DIR" "$run_pattern")$(remote_guard "$SEED_DEPS_DIR" "$run_pattern")"
    SEED_REFRESH_CMD="$SEED_REFRESH_CMD$(printf 't=%q seed=%q seeddeps=%q; ' "$REMOTE_DIR" "$SEED_DIR" "$SEED_DEPS_DIR")"'rsync -a --delete -- "$t/" "$seed/" || exit 1; [ ! -d "$t.deps" ] || rsync -a --delete -- "$t.deps/" "$seeddeps/"'
    GOWORK_WRITE_CMD="$(remote_guard "$REMOTE_DIR/go.work" "$run_pattern")$(printf 'cat > %q' "$REMOTE_DIR/go.work")"
    TREE_RSYNC_PATH="$(remote_guard "$REMOTE_DIR" "$run_pattern")rsync"
    LAUNCH_CMD="$(printf 'run=%q id=%q project=%q session=%q; %s' "$RUN_DIR" "$RUN_ID" "$LOCK_PROJECT" "$LOCK_SESSION" "$EMPTY_ID_REFUSAL")"
    LAUNCH_CMD="$LAUNCH_CMD$(remote_guard "$RUN_DIR" "$run_pattern")"$'\n'"locked='$LOCKED_RUN'$LAUNCH_BODY"
    SUITE_SCRIPT="$(remote_suite)"
    RUN_CHECK_CMD="$(printf 'cat %q %q; tail -5 %q; pg=$(cat %q); case "$pg" in ""|0*|*[!0-9]*|1) echo "no usable process group: [$pg]" ;; *) pgrep -ag "$pg" ;; esac; grep -lzxF CLAWEE_CI_RUN_ID=%q /proc/[0-9]*/environ' \
        "$RUN_BASE.rc" "$RUN_DIR/lock_rc" "$RUN_BASE.log" "$RUN_BASE.pid" "$RUN_ID")"
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

report_unstopped() {
    warn "could not confirm the run on $MACHINE stopped. While any of its processes live they hold the $CI_LOCK_PRODUCT lock, and its tree is left in place."
    warn "  run id:  $RUN_ID"
    warn "  files:   $MACHINE:$RUN_DIR/{run.pid,run.log,run.rc,lock_rc}"
    warn "  check what is left of the run:"
    print_command "ssh $MACHINE '$RUN_CHECK_CMD'"
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
    if [ "${LAUNCHED:-0}" = 1 ] && ! stop_runner; then
        report_unstopped
        exit "$(( rc == 0 ? 1 : rc ))"
    fi
    [ "$TREE_TOUCHED" = 0 ] || remove_tree
    exit "$rc"
}

run_suite() {
    local rc=0
    sync_tree || return 1
    LAUNCHED=1
    if ! launch; then
        warn "could not start the suite on $MACHINE"
        FOLLOW_OUTCOME=unstarted
        return 1
    fi
    say "started on $MACHINE (log $RUN_BASE.log)"
    follow || rc=$?
    if [ "$FOLLOW_OUTCOME" = lock ]; then
        lock_failed "$LOCK_EXIT"
        return
    fi
    [ "$rc" -eq 0 ] || rc=1
    if [ "$FOLLOW_OUTCOME" = status ] && [ -n "$ARTIFACTS_DIR" ] && ! fetch_artifacts; then
        [ "$rc" -ne 0 ] || rc=1
    fi
    [ "$FOLLOW_OUTCOME" = status ] || [ "$rc" -ne 0 ] || rc=1
    return "$rc"
}

main() {
    local prep_out rc=0
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
    TREE_TOUCHED=1
    prep_out="$(remote_n "$PREP_CMD")" || { warn "could not prepare $MACHINE:$SEED_DIR"; exit 1; }
    [ -z "$prep_out" ] || say "$prep_out"
    run_suite || rc=$?
    exit "$rc"
}

main "$@"
