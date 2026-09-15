#!/usr/bin/env bash
# go-headless-term's suite — executed on the shared CI machine, never here.
#
# WHY this exists:
#   - Linux is where the suite runs. `go test` on the Darwin workstation executes
#     unsigned test binaries on the wrong platform, so the tree is synced to the
#     machine and built and tested there over ssh, addressed by name, never
#     through the virtualization CLI (the playbook's testing and vm-share
#     guidelines).
#   - Without options this runs exactly what `ci-test` ran before the repo had a
#     runner — `go build ./...`, then `go test -count=1 <pkg…>` — so a land gate
#     or a plain-run runtime quoted before this file existed still compares.
#   - One CI run per product at a time: the Clawee CI lock is the directory
#     /tmp/ci-lock/clawee ON THE MACHINE, taken with one atomic mkdir, a `holder`
#     file saying who, and a `heartbeat` refreshed every CLAWEE_CI_LOCK_HEARTBEAT_S
#     seconds while the lock is held. The exit trap releases it on success,
#     failure and interrupt — also when the take itself was interrupted or its
#     ssh dropped, since the mkdir may have landed; the release only ever removes
#     a lock whose holder names this run, and does it by rename then delete, so
#     a late heartbeat cannot leave a holderless directory behind. The one
#     exception is a run whose remote kill cannot be confirmed: its lock is left
#     held, with the commands to check and release it printed. A held lock is
#     reported — holder, heartbeat age and the STALE threshold that judged it —
#     and this script exits 3. It never waits for a lock and never breaks one.
#   - Long ssh sessions to the machine get dropped, and a dropped session (exit
#     255) is not a suite result. So the suite runs DETACHED there, writing its
#     log and exit status to files named for this run, and this script follows
#     the log over short reconnecting ssh calls. A run that ends without a
#     status has its process group killed, and the kill confirmed, before the
#     lock is released; if it cannot be confirmed the lock is left to go STALE.
#   - A test-suite review needs more than a green run: the covered set for the
#     whole module, test and case counts with skips, failure names, and
#     shuffled and repeated runs, all returned to the workstation. The options
#     below produce them through this same command.
#   - A local go.work is mirrored (its `use` modules travel with the tree, each
#     to a directory keyed by its full module path, and a go.work naming the
#     remote copies is written there), exactly as `ci-test` resolves one. It
#     never widens -coverpkg: this module imports no other Clawee module, so
#     cross-module coverage (an explicit --cover-module) belongs to the
#     consuming modules' runners, not this one.
#   - Every command that deletes or overwrites on the machine (rsync --delete,
#     rm, mv, the go.work write) is built once at start-up from validated
#     values, and refuses on the machine side unless its path is under this
#     runner's own /tmp/clawee-ght- prefix (or is the Clawee lock itself) and
#     holds no '..': a mistyped CLAWEE_CI_DIR=/tmp must never become
#     `rsync --delete … burrowee-ci:/tmp/`.
#   - No GitHub token is minted: every dependency of this module is public. A
#     go.work naming a private module fails the run loudly inside go.
#
# Usage:
#   ci/run-tests.sh [options] [pkg...]   the suite; packages default to ./...
#   ci/run-tests.sh -h|--help            this text
#
# Options come before the packages. Any of them turns on the evidence mode:
# go test -json, its readable lines streamed, then per-package and module test
# and case counts (pass/fail/skip), each skipped and failed name, and each
# shuffle seed. Without options the plain suite runs, unchanged.
#   --artifacts <dir>   also -covermode=set -coverpkg=./... -coverprofile,
#                       whatever packages are named, and copy test.json,
#                       cover.out and covered.txt to <dir>. covered.txt is the
#                       sorted set of covered blocks, without blocks in
#                       `<pkg>_test/` support packages (lang/go.md). Named
#                       packages with no test files are left out of the run and
#                       listed. <dir> must not exist — evidence is never
#                       overwritten — and its parent must.
#   --shuffle           -shuffle=on; the seed per package is in the log
#   --repeat <n>        -count=<n> instead of -count=1
#
# Environment:
#   CLAWEE_CI_MACHINE             the machine (default burrowee-ci)
#   CLAWEE_CI_DIR                 remote tree (default /tmp/clawee-ght-<user>-<cksum
#                                 of this checkout>); must be /tmp/clawee-ght-<name>,
#                                 <name> of letters, digits and ._- with no '..'
#   CLAWEE_CI_LOCK_PROJECT        project id recorded on the lock (default: the branch)
#   CLAWEE_CI_LOCK_SESSION        session id recorded on the lock (default: unrecorded)
#   CLAWEE_CI_LOCK_HEARTBEAT_S    heartbeat interval in seconds (default 30); stale after 4
#   CLAWEE_CI_POLL_S              seconds between log polls (default 3)
#   CLAWEE_CI_FOLLOW_MAX_MISSES   failed polls in a row before following gives up (default 40)
# The three numbers must be whole numbers of at least 1; anything else is a
# usage error before the machine is contacted.
#
# Exit status (a closed stderr never changes it):
#   0        the build and every test passed, and this run's lock was released —
#            or the release found the lock no longer naming this run (warned):
#            this run then holds no lock
#   1        the build or a test failed (any non-zero status from go is reported
#            as 1, so 2 and 3 below always mean this script), or the machine
#            could not be reached, the tree not synced, the run ended without a
#            status, or its evidence could not be copied home — or the suite
#            passed but this run's lock is or may be left held: its remote kill
#            could not be confirmed, or its release went unanswered (the
#            commands to clear it are on stderr)
#   2        usage error, refused before any contact: bad option, option after
#            packages, existing --artifacts directory, bad environment value
#            (CLAWEE_CI_DIR outside /tmp/clawee-ght-<name>), or a local go.work
#            naming a module that cannot be mirrored safely ('.', '..', empty)
#   3        the Clawee CI lock is held by another run, was released while this
#            run checked it, or its root is not writable — never waited for,
#            never broken; a MISSING root is exit 1: it is the machine's to
#            create, not this script's
#   130/143/129  interrupted by INT/TERM/HUP; 141  stdout closed with SIGPIPE.
#            The run is stopped and the lock released first. While the run is
#            followed a signal is acted on at once, also when it is sent to this
#            script's pid alone. These remote steps are bounded and finish
#            before the signal is acted on: the probe, the lock take, the sync,
#            the launch, the evidence copy-back, and the stop of a run whose
#            follow was lost, which can take up to about 22 s.
# A closed stdout WITHOUT SIGPIPE (`>&-`, or a caller that ignores SIGPIPE) does
# not stop the run: messages continue on stderr and the status is the run's.
# The lock outcome never replaces a failing status. A signal or SIGPIPE that
# arrives after the status is known does: the exit is the signal's.
set -euo pipefail

PROG="ci/run-tests.sh"
MACHINE="${CLAWEE_CI_MACHINE:-burrowee-ci}"
SRC="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Keyed by the checkout, so a run from the project worktree and one from dev —
# or two terminals — never rsync over each other's tree mid-run.
# A literal: the local check and every machine-side guard compare against it.
REMOTE_PREFIX="/tmp/clawee-ght-"
REMOTE_DIR="${CLAWEE_CI_DIR:-$REMOTE_PREFIX$(id -un)-$(printf '%s' "$SRC" | cksum | cut -d' ' -f1)}"
DEPS_DIR="$REMOTE_DIR.deps"

LOCK_ROOT="/tmp/ci-lock"
LOCK="$LOCK_ROOT/clawee"
LOCK_HEARTBEAT_S="${CLAWEE_CI_LOCK_HEARTBEAT_S:-30}"
LOCK_STALE_MULTIPLE=4
LOCK_PROJECT="${CLAWEE_CI_LOCK_PROJECT:-$(git -C "$SRC" rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)}"
LOCK_SESSION="${CLAWEE_CI_LOCK_SESSION:-unrecorded}"
# Names THIS run: on the holder file, so only a lock this run took is refreshed
# or released, and in every remote run file.
RUN_ID="$(id -un | tr -c 'A-Za-z0-9\n' '_')-$$-$(date +%s)"
RUN_BASE="$REMOTE_DIR.run.$RUN_ID"
POLL_S="${CLAWEE_CI_POLL_S:-3}"
FOLLOW_MAX_MISSES="${CLAWEE_CI_FOLLOW_MAX_MISSES:-40}"
DEAD_POLLS=3

# Set by parse_args. EVIDENCE=0 is the plain suite.
EVIDENCE=0
ARTIFACTS_DIR=""
SHUFFLE=0
REPEAT=1
PACKAGES=(./...)
# Where the lock stands: empty (not tried), trying (the take's ssh has not
# answered, so the mkdir may have landed) or taken.
LOCK_STATE=""
# The go.work mirror plan, validated before any contact (plan_workspace).
WS_DIRS=()
WS_MODULES=()
WS_DESTS=()
WS_RSYNC_PATHS=()
# How the run ended: status (the runner wrote one), lost, dead or unstarted.
FOLLOW_OUTCOME=""

usage() {
    awk '/^# Usage:/ { p = 1 } p && !/^#/ { exit } p { sub(/^# ?/, ""); print }' "$0"
}

# A closed stderr never changes the status: the refusal is still a 2.
usage_error() {
    echo "$PROG: $1" >&2 2>/dev/null || echo >/dev/null
    usage >&2 2>/dev/null || true
    exit 2
}

# Messages once the run is under way. A write to a closed stdout or stderr does
# not always raise SIGPIPE (`>&-`, or a caller that ignores SIGPIPE), and bash
# 3.2 keeps the text of the failed write in its output buffer: the next $(...)
# flushes it into the captured value — a remote script, or a poll's answer. So a
# failed write is flushed into /dev/null at once (`echo >/dev/null` flushes it;
# `printf ''` does not), and a closed stdout is pointed at stderr, or at
# /dev/null when stderr is closed too.
say() {
    echo "$PROG: $*" 2>/dev/null && return 0
    stdout_closed
    echo "$PROG: $*" 2>/dev/null || echo >/dev/null
}

warn() {
    echo "$PROG: $*" >&2 2>/dev/null || echo >/dev/null
}

# A recovery command, alone on its line on stderr with no prefix, so the whole
# line pastes into a shell.
print_command() {
    echo "$*" >&2 2>/dev/null || echo >/dev/null
}

# Raw remote log bytes, streamed while following.
emit() {
    printf '%s' "$1" 2>/dev/null && return 0
    stdout_closed
    printf '%s' "$1" 2>/dev/null || echo >/dev/null
}

# The stderr test is a bare `: >&2`: wrapping it in 2>/dev/null would test
# /dev/null, and an `exec 1>&2` onto a closed stderr ends the shell.
stdout_closed() {
    echo >/dev/null
    if : >&2; then
        exec 1>&2
        warn "stdout was closed — messages continue on stderr"
    else
        exec 1>/dev/null
    fi
}

# Options first, then packages. An option after a package is refused rather
# than handed to go as a package name.
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

# Refused before anything runs, so an "after" run can never land on a baseline.
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

# The remote tree is deleted into (rsync --delete) and globbed for removal
# (<dir>.run.*), so it is accepted only as this runner's own prefix plus a name.
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

# Every path a machine-side guard will check, tested here with the guard's own
# two tests before any contact. The value checks alone were not enough: a
# <name> ending in '.' passed them, but its run base <dir>.run.<id> holds '..',
# so the machine refused the launch AND the stop, and the lock was kept.
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

# The numeric settings, refused before any contact unless a whole number of at
# least 1. A heartbeat of 0 or "abc" makes the remote refresher loop without
# sleeping on the shared machine and records a heartbeat_s every sibling
# misjudges; a bad poll count breaks the follower's give-up test.
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

# A loaded machine times out the ssh banner exchange; that is a slow machine,
# not a failed suite, so every call waits longer than ssh's default.
# remote_n carries no stdin (ssh -n); remote_stdin is for the three calls that
# pipe something in on purpose: the holder text, the run script, the go.work.
# Every ssh carries ServerAliveInterval=15 with ServerAliveCountMax=4: a
# half-open connection is given up after about a minute instead of parking the
# follow, the heartbeat or the stop.
SSH_OPTS=(-o BatchMode=yes -o ConnectTimeout=30 -o ServerAliveInterval=15 -o ServerAliveCountMax=4)
# rsync's own ssh gets the same options, so a half-open connection cannot park
# a sync either.
RSYNC_SSH="ssh ${SSH_OPTS[*]}"

remote_n() {
    ssh -n "${SSH_OPTS[@]}" "$MACHINE" "$@"
}

remote_stdin() {
    ssh "${SSH_OPTS[@]}" "$MACHINE" "$@"
}

# Two failures with different owners: a stopped machine (only its owner starts
# it) and an account that is not enrolled (fixed by that account). Port 22
# answering tells them apart.
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

# One atomic mkdir, so two runs cannot both succeed. The holder text travels on
# stdin, so a quote in a branch or session name cannot break the command.
# Remote exit: 0 taken, 3 held (holder and heartbeat age printed) or released
# between the mkdir and the check (`changed`), 4 the lock root is not writable
# by this account, 5 the lock root is missing. The root is never created here:
# the machine's tmpfiles.d entry makes it 1777 at boot (local overlay,
# machine.md), and a root made by whichever account ran first is owned by that
# account with whatever mode it chose.
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

# Report a lock that could not be taken, and exit 3 — also when stderr is
# closed, so every write here tolerates failing. Never breaks it. The
# holder's own heartbeat interval judges it when its holder file records one;
# a holder written by another tool is judged by this run's interval, and the
# printed rule says which.
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

# Refresh the heartbeat from here for as long as the lock is held — the sync and
# the teardown are gaps the detached runner does not cover. It refreshes only
# while this script is alive: a SIGKILLed script runs no trap, and a refresher
# that outlived it would keep the lock looking live. Its sleep and its ssh run
# as a child it waits on, and TERM kills that child: an ssh left running past
# stop_heartbeat could write a heartbeat into the lock while it is released.
# The ssh is started directly, not through remote_n: a backgrounded function
# is a subshell, and killing it leaves its ssh running. Both waits are guarded:
# the subshell inherits set -e, and one failed ssh (the machine drops sessions)
# must not end the refreshing for the rest of the run.
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

# Release only the lock this run took (its run id on the holder file), by an
# atomic rename and then a delete: the lock path is free the moment the rename
# lands, whatever a late write does to the renamed directory. The remote side
# exits 3 when the holder does not name this run. `quiet` is for a take that
# never answered: there, "not ours" is the normal case and says nothing, but a
# machine that cannot be reached still warns, because the take may have landed.
# Non-zero only when the lock may still be held by this run.
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

# --delete so a file removed locally cannot linger and keep a stale test green.
# go.work is excluded and rebuilt remotely: its `use` lines are relative to this
# checkout and resolve to nothing on the machine. rsync's --exclude keeps
# --delete from removing a remote go.work, so one left by an earlier run is
# removed here when this checkout has none. Every step is checked by hand: the
# caller runs this under `||`, where set -e does not apply.
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

# Every `use` path of the local go.work, from both spellings: the parenthesised
# block and a single-line `use ./path`.
workspace_uses() {
    awk '
        /^[[:space:]]*use[[:space:]]*\(/ { inblock = 1; next }
        inblock && /^[[:space:]]*\)/     { inblock = 0; next }
        inblock                          { print $1; next }
        /^[[:space:]]*use[[:space:]]+/   { print $2 }
    ' "$SRC/go.work"
}

# The go.work module in directory $1: its module path, refused unless it holds
# only Go's module-path characters, no empty or '.' segment and no '..'
# anywhere (the machine guard refuses any path holding '..', so a Go-valid
# `example.com/a..b` must stop here, before contact), since it names a
# directory the machine deletes into.
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

# Each `use` module other than this one goes to DEPS_DIR/<module path with / as
# _> — keyed by the whole path, so two modules sharing a last segment never
# share a directory. Planned before any contact: every module is validated and
# every rsync --delete destination and its machine-side guard are built here.
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

# Mirror the planned modules and write a go.work naming the copies.
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

# The script the machine runs. Every value is quoted with %q, so a package
# argument reaches go as the one word it was given. It records its pid (the
# process group an interrupt kills), refreshes the lock heartbeat while live,
# builds, tests, and writes its status last, by rename.
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

# Shell for the machine, evidence mode: the packages to test into "$@", then
# the events teed to $RUN_BASE.json while the readable lines stream into the
# log; go's status, not the filter's; then the summary.
#
# Under --artifacts, named packages WITHOUT test files are left out of go test
# and listed in the log. With -coverpkg, go runs its covdata tool for such a
# package, and the go1.25.1 toolchain GOTOOLCHAIN=auto fetches ships no
# prebuilt covdata: the run failed with `go: no such tool "covdata"` for
# examples/basic while every test passed. A package with no tests covers
# nothing, so the covered set is the same without it; -coverpkg still spans
# the whole module.
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

# The readable log from go test -json: every output line except the RUN / PAUSE
# / CONT / NAME and PASS chatter, so ok/FAIL, --- FAIL and --- SKIP lines with
# their messages, and shuffle seeds, remain. A line that is not JSON passes
# through as it is.
evidence_filter_cmd() {
    printf 'jq --unbuffered -Rrj %q' '(fromjson? // {Action: "output", Output: (. + "\n")})
        | select(.Action == "output" or .Action == "build-output") | .Output
        | select(test("^\\s*(=== (RUN|PAUSE|CONT|NAME)|--- PASS)") | not)'
}

# Shell for the machine, after the run: test and case counts by outcome per
# package and for the module, a line per skipped and failed test and per
# shuffle seed; then, with --artifacts, the covered set and its size. Blocks in
# `<pkg>_test/` support packages are dropped from the set (lang/go.md, "The
# covered set"): an audit's EXTRACT moves code into them, so their block keys
# change without any production change. A missing profile fails the run.
# Expects $rc.
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

# Write the script and start it detached, in ONE ssh. setsid makes the runner
# its own process group leader, and its pid is written in the same breath, so a
# runner is never alive without a pid file even if this ssh drops right after.
launch() {
    printf '%s\n' "$SUITE_SCRIPT" | remote_stdin "$LAUNCH_CMD"
}

LAUNCH_BODY='
            rm -f -- "$base".* || exit 1
            cat > "$base.sh" || exit 1
            setsid nohup bash "$base.sh" > "$base.log" 2>&1 < /dev/null &
            echo $! > "$base.pid"'

# One poll's remote half. Prints "<alive> <rc-or-empty> <log size>", a newline,
# the log bytes from offset $1 up to that size, and a closing "." — the
# sentinel keeps command substitution from eating trailing newlines. The status
# is read BEFORE the size, so once a status is seen the bytes are the whole log.
poll_cmd() {
    printf 'base=%q off=%q; ' "$RUN_BASE" "$1"
    printf '%s' 'alive=dead; [ -f "$base.pid" ] && kill -0 "$(cat "$base.pid")" 2>/dev/null && alive=alive; '
    printf '%s' 'rc=$(cat "$base.rc" 2>/dev/null); size=$(stat -c %s "$base.log" 2>/dev/null || echo 0); '
    printf '%s' 'echo "$alive $rc $size"; tail -c +$((off + 1)) "$base.log" 2>/dev/null | head -c $((size - off)); printf .'
}

# Follow the remote log until the status file appears. The poll's ssh and the
# pause between polls run as background children the loop waits on: bash defers
# a trap until a foreground command ends, so a TERM sent to this script alone
# waited out the pause (CLAWEE_CI_POLL_S) or a slow poll before the stop and
# release. `wait` returns at once on a trapped signal, and cleanup kills the
# child. The ssh is backgrounded directly, not through remote_n: killing a
# backgrounded function kills its subshell and leaves the ssh running. A failed
# poll is the transport, not the suite: retried up to FOLLOW_MAX_MISSES in a
# row. A runner
# gone without a status is reported after DEAD_POLLS polls. Sets
# FOLLOW_OUTCOME; returns the status, or 1 without one.
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

# Kill the run's process group and confirm nothing in it survives. The group
# comes from the pid file, or, when there is none, from a process still running
# the script: a missing pid file alone is not proof the runner is gone.
# Non-zero when that cannot be confirmed. Built once, before the run, into
# STOP_CMD (see build_teardown_cmds).
# The process-group id must be a number above 1 before it reaches kill or
# pgrep -g: an empty one (a pid file caught mid-write) makes `pgrep -g ""`
# fail, which the loop used to read as "nothing left" — a confirmed stop that
# confirmed nothing. Only pgrep's status 1 (no match) counts as gone; any other
# failure is an unconfirmed stop, so the lock stays held and the recovery
# commands are printed.
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
        case "$pg" in ""|*[!0-9]*|0|1) echo unusable process group: "[$pg]" >&2; exit 1 ;; esac
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

# Machine-side refusal, placed first in every command that deletes or
# overwrites there: $1 must match the shell pattern $2 and hold no '..'. The
# messages carry no quotes or glob characters: rsync re-splits --rsync-path, and
# a quoted pattern came out of it glob-expanded.
remote_guard() {
    printf 'case %q in %s) ;; *) echo refused, outside the runner prefix: %q >&2; exit 64 ;; esac; ' "$1" "$2" "$1"
    printf 'case %q in *..*) echo refused, path holds dot-dot: %q >&2; exit 64 ;; esac; ' "$1" "$1"
}

# Every remote command the teardown sends, built at start-up while stdout is
# known to be healthy. A $(...) run during teardown can pick up the text of an
# earlier failed write (see say); a prebuilt string cannot.
build_teardown_cmds() {
    local gone="$LOCK.released.$RUN_ID" run_pattern="$REMOTE_PREFIX?*"
    STOP_CMD="$(stop_runner_cmd)"
    # The lock patterns are literals, not derived from $LOCK: a guard built from
    # the value it checks would accept anything.
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
    # The check lists what is left of the run: by its process group while the
    # pid file exists, and otherwise by the script's command line — the same
    # fallback stop_runner uses, since a missing pid file is not proof the run
    # is gone. It is printed inside single quotes, so it holds none.
    RUN_CHECK_CMD="$(printf 'ls -l %q.pid %q.rc %q.log; tail -5 %q.log; if [ -f %q.pid ]; then pgrep -ag "$(cat %q.pid)"; else pgrep -af "^bash %q.sh\\$"; fi' \
        "$RUN_BASE" "$RUN_BASE" "$RUN_BASE" "$RUN_BASE" "$RUN_BASE" "$RUN_BASE" "$RUN_BASE")"
}

# Copy the evidence home over the runner's own ssh. Each file lands under a
# .part name and is renamed, so a dropped copy never leaves a truncated file
# that reads as a whole one. Every file is tried; non-zero, naming each, when
# any copy failed.
fetch_artifacts() {
    local pair from to failed=0
    mkdir "$ARTIFACTS_DIR" || { warn "could not create $ARTIFACTS_DIR"; return 1; }
    for pair in json:test.json cover.out:cover.out covered.txt:covered.txt; do
        from="$RUN_BASE.${pair%%:*}"
        to="$ARTIFACTS_DIR/${pair#*:}"
        # In a subshell: bash runs a trap while a redirected function call still
        # holds its redirection, so a signal here wrote cleanup's "released"
        # into the evidence file.
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

# The one exit that deliberately keeps the lock: the run's process group could
# not be confirmed dead, so the lock stays with it and goes STALE rather than
# let a second run start beside a live one. Everything needed to finish the job
# by hand is printed. Each command is alone on its line, so the whole line
# pastes into a shell.
report_kept_lock() {
    warn "could not confirm the run on $MACHINE is gone — $MACHINE:$LOCK is left held (it goes STALE); not released"
    warn "  run id:  $RUN_ID"
    warn "  files:   $MACHINE:$RUN_BASE.{pid,log,rc}"
    warn "  check what is left of the run:"
    print_command "ssh $MACHINE '$RUN_CHECK_CMD'"
    warn "  release, only once the check shows nothing left running:"
    print_command "ssh $MACHINE '$RELEASE_CMD'"
}

# Exit trap. Repeated signals are ignored while tearing down, so a second
# Ctrl-C cannot skip stopping the runner or releasing the lock. PIPE is ignored
# and errexit is off too: a reader that closed stdout (`ci/run-tests.sh | head`)
# makes every later echo fail, and without this the first one would end the
# teardown before the lock is released. The first write flushes whatever a
# failed write left in bash's buffer into /dev/null (say), and every remote
# command sent from here was built at start-up (build_teardown_cmds).
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
    # A passing run whose lock may still be held (the release went unanswered)
    # is not a clean exit; a non-zero status already says the run failed.
    case "$LOCK_STATE" in
        taken) release_lock || [ "$rc" -ne 0 ] || rc=1 ;;
        trying) release_lock quiet || [ "$rc" -ne 0 ] || rc=1 ;;
    esac
    exit "$rc"
}

# Launch, follow, and copy the evidence home. Returns the run's status.
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
