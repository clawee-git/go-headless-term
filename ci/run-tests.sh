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
#     file saying who, and a `heartbeat` refreshed every CI_LOCK_HEARTBEAT_S
#     seconds while the lock is held. The exit trap releases it on success,
#     failure and interrupt. A held lock is reported — holder, heartbeat age and
#     the STALE threshold that judged it — and this script exits 3. It never
#     breaks a lock.
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
#   - A local go.work is mirrored (its `use` modules travel with the tree and a
#     go.work naming the remote copies is written there), exactly as `ci-test`
#     does. In the evidence mode every mirrored module also joins -coverpkg, so
#     code in another module reached by this module's tests lands in the
#     profile under that module's path.
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
#   --artifacts <dir>   also -covermode=set -coverpkg=./... (plus each go.work
#                       module) -coverprofile, whatever packages are named, and
#                       copy test.json, cover.out and covered.txt (the sorted
#                       covered blocks) to <dir>. <dir> must not exist — evidence
#                       is never overwritten — and its parent must.
#   --shuffle           -shuffle=on; the seed per package is in the log
#   --repeat <n>        -count=<n> instead of -count=1
#
# Environment:
#   CI_MACHINE            the machine (default burrowee-ci)
#   CI_TEST_DIR           remote tree (default /tmp/clawee-ght-<user>-<cksum of
#                         this checkout>); absolute, letters, digits and ._/- only
#   CI_LOCK_PROJECT       project id recorded on the lock (default: the branch)
#   CI_LOCK_SESSION       session id recorded on the lock (default: unrecorded)
#   CI_LOCK_HEARTBEAT_S   heartbeat interval in seconds (default 30); stale after 4
#   CI_POLL_S             seconds between log polls (default 3)
#   CI_FOLLOW_MAX_MISSES  failed polls in a row before following gives up (default 40)
#
# Exit status: the suite's (go build's, then go test's), 2 for a usage error,
# 3 when the CI lock is held or cannot be taken, 1 when the machine cannot be
# reached, the run ended without a status, or its evidence could not be copied
# home, 130/143/129 on INT/TERM/HUP.
set -euo pipefail

PROG="ci/run-tests.sh"
MACHINE="${CI_MACHINE:-burrowee-ci}"
SRC="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Keyed by the checkout, so a run from the project worktree and one from dev —
# or two terminals — never rsync over each other's tree mid-run.
REMOTE_DIR="${CI_TEST_DIR:-/tmp/clawee-ght-$(id -un)-$(printf '%s' "$SRC" | cksum | cut -d' ' -f1)}"
DEPS_DIR="$REMOTE_DIR.deps"

LOCK_ROOT="/tmp/ci-lock"
LOCK="$LOCK_ROOT/clawee"
LOCK_HEARTBEAT_S="${CI_LOCK_HEARTBEAT_S:-30}"
LOCK_STALE_MULTIPLE=4
LOCK_PROJECT="${CI_LOCK_PROJECT:-$(git -C "$SRC" rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)}"
LOCK_SESSION="${CI_LOCK_SESSION:-unrecorded}"
# Names THIS run: on the holder file, so only a lock this run took is refreshed
# or released, and in every remote run file.
RUN_ID="$(id -un | tr -c 'A-Za-z0-9\n' '_')-$$-$(date +%s)"
RUN_BASE="$REMOTE_DIR.run.$RUN_ID"
POLL_S="${CI_POLL_S:-3}"
FOLLOW_MAX_MISSES="${CI_FOLLOW_MAX_MISSES:-40}"
DEAD_POLLS=3

# Set by parse_args. EVIDENCE=0 is the plain suite.
EVIDENCE=0
ARTIFACTS_DIR=""
SHUFFLE=0
REPEAT=1
PACKAGES=(./...)
# Module paths of the go.work `use` entries mirrored by sync_tree.
WORKSPACE_MODULES=""
# How the run ended: status (the runner wrote one), lost, dead or unstarted.
FOLLOW_OUTCOME=""

usage() {
    awk '/^# Usage:/ { p = 1 } p && !/^#/ { exit } p { sub(/^# ?/, ""); print }' "$0"
}

usage_error() {
    echo "$PROG: $1" >&2
    usage >&2
    exit 2
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

check_remote_dir() {
    case "$REMOTE_DIR" in
        /*) ;;
        *) usage_error "CI_TEST_DIR must be an absolute path: '$REMOTE_DIR'" ;;
    esac
    case "$REMOTE_DIR" in
        *[!A-Za-z0-9._/-]*) usage_error "CI_TEST_DIR may hold only letters, digits and ._/-: '$REMOTE_DIR'" ;;
    esac
}

# A loaded machine times out the ssh banner exchange; that is a slow machine,
# not a failed suite, so every call waits longer than ssh's default.
remote() {
    ssh -o BatchMode=yes -o ConnectTimeout=30 -o ServerAliveInterval=15 "$MACHINE" "$@"
}

# Two failures with different owners: a stopped machine (only its owner starts
# it) and an account that is not enrolled (fixed by that account). Port 22
# answering tells them apart.
probe_machine() {
    remote true </dev/null 2>/dev/null && return 0
    echo "$PROG: $MACHINE is not answering ssh." >&2
    if (exec 3<>"/dev/tcp/$MACHINE/22") 2>/dev/null; then
        echo "$PROG: port 22 answered — the machine is up; it is loaded, or this account is not enrolled (vm enroll)." >&2
    else
        echo "$PROG: port 22 did not answer — the machine is stopped; only its owner can start it." >&2
    fi
    exit 1
}

# One atomic mkdir, so two runs cannot both succeed. The holder text travels on
# stdin, so a quote in a branch or session name cannot break the command.
# Remote exit: 0 taken, 3 held (holder and heartbeat age printed), 4 the lock
# root is not writable by this account.
take_lock() {
    printf 'project=%s\nsession=%s\nuser=%s\ntaken=%s\nrepo=%s\nrun=%s\nheartbeat_s=%s\n' "$LOCK_PROJECT" "$LOCK_SESSION" \
        "$(id -un)" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$SRC" "$RUN_ID" "$LOCK_HEARTBEAT_S" |
        remote "$(printf 'root=%q lock=%q; ' "$LOCK_ROOT" "$LOCK")"'
            [ -d "$root" ] || mkdir -m 1777 "$root" 2>/dev/null
            if mkdir "$lock" 2>/dev/null; then
                cat > "$lock/holder" && date -u +%Y-%m-%dT%H:%M:%SZ > "$lock/heartbeat" && exit 0
                rm -rf "$lock"; exit 4
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
            stat -c "root %n is %U:%G mode %a" "$root"; exit 4'
}

# Report a lock that could not be taken, and exit 3. Never breaks it. The
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
        echo "$PROG: cannot take $MACHINE:$LOCK — the lock root is not writable by $(id -un):" >&2
        printf '%s\n' "$out" | sed "s|^|$PROG:   |" >&2
        exit 3
    fi
    age="$(printf '%s\n' "$out" | sed -n 's/.* age=\([0-9-]*\)s$/\1/p')"
    echo "$PROG: the Clawee CI lock $MACHINE:$LOCK is held:" >&2
    printf '%s\n' "$out" | sed "s|^|$PROG:   |" >&2
    if [ -z "$age" ]; then
        echo "$PROG: no heartbeat recorded — its age cannot be judged ($rule). Ask the holder; breaking it is an operator decision." >&2
    elif [ "$age" -gt "$stale" ]; then
        echo "$PROG: STALE — no heartbeat for ${age}s, $rule." >&2
        echo "$PROG: breaking it is an operator decision; this script never does." >&2
    else
        echo "$PROG: live — heartbeat ${age}s old, $rule. Wait for it." >&2
    fi
    exit 3
}

heartbeat_cmd() {
    printf 'grep -qx %q %q && date -u +%%Y-%%m-%%dT%%H:%%M:%%SZ > %q' "run=$RUN_ID" "$LOCK/holder" "$LOCK/heartbeat"
}

# Refresh the heartbeat from here for as long as the lock is held — the sync and
# the teardown are gaps the detached runner does not cover. It refreshes only
# while this script is alive: a SIGKILLed script runs no trap, and a refresher
# that outlived it would keep the lock looking live.
start_heartbeat() {
    local cmd parent=$$
    cmd="$(heartbeat_cmd)"
    (
        while sleep "$LOCK_HEARTBEAT_S" && kill -0 "$parent" 2>/dev/null; do
            remote "$cmd" </dev/null >/dev/null 2>&1 || true
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

# Release only the lock this run took (its run id on the holder file).
release_lock() {
    if remote "$(printf 'grep -qx %q %q && rm -rf %q' "run=$RUN_ID" "$LOCK/holder" "$LOCK")" </dev/null 2>/dev/null; then
        echo "$PROG: released $MACHINE:$LOCK"
    else
        echo "$PROG: lock $MACHINE:$LOCK not released by this run (not ours, or unreachable) — check it" >&2
    fi
}

# --delete so a file removed locally cannot linger and keep a stale test green.
# go.work is excluded and rebuilt remotely: its `use` lines are relative to this
# checkout and resolve to nothing on the machine. rsync's --exclude keeps
# --delete from removing a remote go.work, so one left by an earlier run is
# removed here when this checkout has none. Every step is checked by hand: the
# caller runs this under `||`, where set -e does not apply.
sync_tree() {
    echo "$PROG: sync $SRC -> $MACHINE:$REMOTE_DIR"
    rsync -a --delete --exclude '.git' --exclude '.codegraph' --exclude '/dist' \
        --exclude 'go.work' --exclude 'go.work.sum' "$SRC/" "$MACHINE:$REMOTE_DIR/" || return 1
    if [ -f "$SRC/go.work" ]; then
        mirror_workspace || return 1
    else
        remote "$(printf 'rm -f %q %q' "$REMOTE_DIR/go.work" "$REMOTE_DIR/go.work.sum")" </dev/null || return 1
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

# Each `use` module other than this one goes to DEPS_DIR/<last path segment>,
# and a go.work naming those copies is written on the machine.
mirror_workspace() {
    local u dep module name uses="" go_directive
    go_directive="$(awk '$1 == "go" { print $2; exit }' "$SRC/go.work")"
    remote "$(printf 'mkdir -p %q' "$DEPS_DIR")" </dev/null || return 1
    for u in $(workspace_uses); do
        [ "$u" = "." ] && continue
        case "$u" in /*) dep="$u" ;; *) dep="$SRC/$u" ;; esac
        dep="$(cd "$dep" 2>/dev/null && pwd)" || { echo "$PROG: go.work names '$u', which is not a directory" >&2; return 1; }
        module="$(awk '$1 == "module" { print $2; exit }' "$dep/go.mod" 2>/dev/null)"
        [ -n "$module" ] || { echo "$PROG: $dep/go.mod declares no module path" >&2; return 1; }
        name="${module##*/}"
        rsync -a --delete --exclude '.git' --exclude '.codegraph' --exclude '/dist' \
            --exclude 'go.work' --exclude 'go.work.sum' "$dep/" "$MACHINE:$DEPS_DIR/$name/" || return 1
        uses="$uses	$DEPS_DIR/$name
"
        WORKSPACE_MODULES="$WORKSPACE_MODULES $module"
        echo "$PROG: go.work: $module <- $dep"
    done
    printf 'go %s\n\nuse (\n\t.\n%s)\n' "${go_directive:-1.25.1}" "$uses" |
        remote "$(printf 'cat > %q' "$REMOTE_DIR/go.work")"
}

# What the covered set spans: the whole module, plus every go.work module.
coverpkg() {
    local list="./..." module
    for module in $WORKSPACE_MODULES; do
        list="$list,$module/..."
    done
    printf '%s' "$list"
}

go_test_flags() {
    local flags="-count=$REPEAT"
    [ "$SHUFFLE" = 0 ] || flags="$flags -shuffle=on"
    [ "$EVIDENCE" = 0 ] || flags="$flags -json"
    if [ -n "$ARTIFACTS_DIR" ]; then
        flags="$flags -covermode=set -coverpkg=$(coverpkg) -coverprofile=$(printf %q "$RUN_BASE.cover.out")"
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
    printf 'kill $beat 2>/dev/null\necho $rc > %q && mv %q %q\nrm -f %q\nexit $rc\n' \
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
# shuffle seed; then, with --artifacts, the covered set and its size. A missing
# profile fails the run. Expects $rc.
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
    printf 'awk %q %q | LC_ALL=C sort -u > %q\n' 'FNR > 1 && $NF > 0 {print $1}' "$RUN_BASE.cover.out" "$RUN_BASE.covered.txt"
    printf 'echo "=== covered $(wc -l < %q) blocks"\n' "$RUN_BASE.covered.txt"
    printf 'else\necho "=== no coverage profile"; [ $rc -ne 0 ] || rc=1\nfi\n'
}

# Write the script and start it detached, in ONE ssh. setsid makes the runner
# its own process group leader, and its pid is written in the same breath, so a
# runner is never alive without a pid file even if this ssh drops right after.
launch() {
    local script
    script="$(remote_suite)" || return 1
    printf '%s\n' "$script" |
        remote "$(printf 'base=%q; ' "$RUN_BASE")"'
            rm -f "$base".* || exit 1
            cat > "$base.sh" || exit 1
            setsid nohup bash "$base.sh" > "$base.log" 2>&1 < /dev/null &
            echo $! > "$base.pid"'
}

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

# Follow the remote log until the status file appears. A failed poll is the
# transport, not the suite: retried up to FOLLOW_MAX_MISSES in a row. A runner
# gone without a status is reported after DEAD_POLLS polls. Sets
# FOLLOW_OUTCOME; returns the status, or 1 without one.
follow() {
    local offset=0 misses=0 dead=0 out header alive rc size
    while :; do
        if out="$(remote "$(poll_cmd "$offset")" </dev/null)"; then
            misses=0
            header="${out%%$'\n'*}"
            out="${out#*$'\n'}"
            printf '%s' "${out%.}"
            read -r alive rc size <<<"$header"
            if [ -z "$size" ]; then size="$rc"; rc=""; fi
            offset="$size"
            if [ -n "$rc" ]; then FOLLOW_OUTCOME=status; return "$rc"; fi
            if [ "$alive" = dead ]; then dead=$((dead + 1)); else dead=0; fi
            if [ "$dead" -ge "$DEAD_POLLS" ]; then
                echo "$PROG: the runner on $MACHINE is gone and wrote no status — killed? log $RUN_BASE.log" >&2
                FOLLOW_OUTCOME=dead; return 1
            fi
        else
            misses=$((misses + 1))
            echo "$PROG: lost contact with $MACHINE (poll $misses) — the run continues there" >&2
            if [ "$misses" -ge "$FOLLOW_MAX_MISSES" ]; then
                echo "$PROG: giving up following; log $MACHINE:$RUN_BASE.log" >&2
                FOLLOW_OUTCOME=lost; return 1
            fi
        fi
        sleep "$POLL_S"
    done
}

# Kill the run's process group and confirm nothing in it survives. The group
# comes from the pid file, or, when there is none, from a process still running
# the script: a missing pid file alone is not proof the runner is gone.
# Non-zero when that cannot be confirmed.
stop_runner() {
    remote "$(printf 'pidf=%q script=%q; ' "$RUN_BASE.pid" "$RUN_BASE.sh")"'
        if [ -f "$pidf" ]; then
            pg=$(cat "$pidf")
        else
            pid=$(pgrep -f -- "^bash $script\$" | head -1)
            [ -n "$pid" ] || exit 0
            pg=$(ps -o pgid= -p "$pid" | tr -d " ")
            [ -n "$pg" ] || exit 0
        fi
        kill -TERM -- "-$pg" 2>/dev/null
        for i in $(seq 1 20); do
            pgrep -g "$pg" >/dev/null || { rm -f "$pidf"; exit 0; }
            sleep 1
        done
        kill -KILL -- "-$pg" 2>/dev/null; sleep 1
        pgrep -g "$pg" >/dev/null && exit 1
        rm -f "$pidf"' </dev/null 2>/dev/null
}

# Copy the evidence home over the runner's own ssh. Each file lands under a
# .part name and is renamed, so a dropped copy never leaves a truncated file
# that reads as a whole one. Every file is tried; non-zero, naming each, when
# any copy failed.
fetch_artifacts() {
    local pair from to failed=0
    mkdir "$ARTIFACTS_DIR" || { echo "$PROG: could not create $ARTIFACTS_DIR" >&2; return 1; }
    for pair in json:test.json cover.out:cover.out covered.txt:covered.txt; do
        from="$RUN_BASE.${pair%%:*}"
        to="$ARTIFACTS_DIR/${pair#*:}"
        if remote "$(printf 'cat -- %q' "$from")" </dev/null >"$to.part" && mv "$to.part" "$to"; then
            continue
        fi
        rm -f "$to.part"
        echo "$PROG: could not copy $MACHINE:$from to $to" >&2
        failed=1
    done
    [ "$failed" = 0 ] || return 1
    echo "$PROG: evidence copied to $ARTIFACTS_DIR:"
    (cd "$ARTIFACTS_DIR" && wc -c test.json cover.out covered.txt | sed "s|^|$PROG:   |")
}

# Exit trap. Repeated signals are ignored while tearing down, so a second
# Ctrl-C cannot skip stopping the runner or releasing the lock.
cleanup() {
    local rc=$?
    trap '' INT TERM HUP
    trap - EXIT
    if [ "${LAUNCHED:-0}" = 1 ] && [ "$FOLLOW_OUTCOME" != status ] && ! stop_runner; then
        stop_heartbeat
        echo "$PROG: could not confirm the runner on $MACHINE is gone — $LOCK is left for it (it goes STALE); not released" >&2
        exit "$(( rc == 0 ? 1 : rc ))"
    fi
    stop_heartbeat
    if [ "${LOCK_TAKEN:-0}" = 1 ]; then
        release_lock
    fi
    exit "$rc"
}

# Launch, follow, and copy the evidence home. Returns the run's status.
run_suite() {
    local rc=0
    remote "$(printf 'rm -f -- %q.run.*' "$REMOTE_DIR")" </dev/null || true
    sync_tree || { echo "$PROG: could not sync the tree to $MACHINE" >&2; return 1; }
    LAUNCHED=1
    if ! launch; then
        echo "$PROG: could not start the suite on $MACHINE" >&2
        FOLLOW_OUTCOME=unstarted
        return 1
    fi
    echo "$PROG: started on $MACHINE (log $RUN_BASE.log)"
    follow || rc=$?
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
    probe_machine
    trap cleanup EXIT
    trap 'exit 130' INT
    trap 'exit 143' TERM
    trap 'exit 129' HUP
    lock_out="$(take_lock)" || lock_rc=$?
    case "$lock_rc" in
        0) LOCK_TAKEN=1; echo "$PROG: took $MACHINE:$LOCK (project $LOCK_PROJECT, session $LOCK_SESSION)" ;;
        3|4) refuse_lock "$lock_rc" "$lock_out" ;;
        *) echo "$PROG: could not reach $MACHINE to take the lock (ssh $lock_rc)" >&2; exit 1 ;;
    esac
    start_heartbeat
    run_suite || rc=$?
    exit "$rc"
}

main "$@"
