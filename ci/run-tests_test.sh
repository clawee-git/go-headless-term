#!/usr/bin/env bash
set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUNNER="$HERE/run-tests.sh"
CHECKOUT="$(cd "$HERE/.." && pwd)"
WORK="$(mktemp -d "${TMPDIR:-/tmp}/run-tests-test.XXXXXX")"
PREP_SEED="/tmp/clawee-ght-rtt-$$"
trap 'rm -rf "$WORK" "$PREP_SEED" "$PREP_SEED".t-*' EXIT
FAILED=0
SEED=/tmp/clawee-ght-runtests-test
PRODUCT=clawee-go-headless-term

write_stubs() {
    mkdir -p "$WORK/bin"
    cat > "$WORK/bin/ssh" <<'STUB'
#!/usr/bin/env bash
cmd="${!#}"
n=$(( $(cat "$STUB_DIR/seq" 2>/dev/null || echo 0) + 1 ))
echo "$n" > "$STUB_DIR/seq"
printf '%s\n' "$cmd" > "$STUB_DIR/call.$n.ssh"
[ "$1" = -n ] || cat > "$STUB_DIR/call.$n.stdin"
case "$cmd" in
    true) exit 0 ;;
    *"kill -0"*) printf 'dead %s 0\n.' "$STUB_STATE"; exit 0 ;;
    *survivor*)
        s=$(( $(cat "$STUB_DIR/stops" 2>/dev/null || echo 0) + 1 ))
        echo "$s" > "$STUB_DIR/stops"
        [ "$s" -gt "${STUB_STOP_255:-0}" ] || exit 255
        [ -z "${STUB_STOP_FAIL:-}" ] || exit 1 ;;
esac
exit 0
STUB
    cat > "$WORK/bin/rsync" <<'STUB'
#!/usr/bin/env bash
n=$(( $(cat "$STUB_DIR/seq" 2>/dev/null || echo 0) + 1 ))
echo "$n" > "$STUB_DIR/seq"
printf '%s\n' "$@" > "$STUB_DIR/call.$n.rsync"
exit 0
STUB
    chmod +x "$WORK/bin/ssh" "$WORK/bin/rsync"
}

run_runner() {
    local state="$1"
    STUB_DIR="$WORK/run.$state"
    rm -rf "$STUB_DIR"; mkdir -p "$STUB_DIR"
    RC=0
    env PATH="$WORK/bin:$PATH" STUB_STOP_255="${STUB_STOP_255:-0}" STUB_STOP_FAIL="${STUB_STOP_FAIL:-}" STUB_DIR="$STUB_DIR" STUB_STATE="$state" CLAWEE_CI_POLL_S=1 \
        CLAWEE_CI_DIR="$SEED" CLAWEE_CI_LOCK_PROJECT=proj-x CLAWEE_CI_LOCK_SESSION=sess-y \
        perl -e 'alarm 30; exec @ARGV' "$RUNNER" ./internal/nothing \
        > "$STUB_DIR/out" 2> "$STUB_DIR/err" < /dev/null || RC=$?
}

fail() {
    printf 'FAIL %s: %s\n' "$1" "$2"
    FAILED=1
}

pass() {
    printf 'ok   %s\n' "$1"
}

launch_call() {
    grep -l 'setsid nohup' "$STUB_DIR"/call.*.ssh 2>/dev/null | head -1
}

test_launch_runs_under_product_lock() {
    local t=launch_runs_under_product_lock f
    run_runner 0
    f="$(launch_call)"
    [ -n "$f" ] || { fail $t "no launch command was sent"; return; }
    grep -Eq "ci-lock run $PRODUCT --timeout [0-9]+ --project [^ ]+ --session [^ ]+ --" "$f" ||
        { fail $t "launch lacks 'ci-lock run $PRODUCT --timeout <s> --project … --session … --': $(tr '\n' ' ' < "$f")"; return; }
    if ! grep -q 'proj-x' "$f" || ! grep -q 'sess-y' "$f"; then
        fail $t "launch does not pass CLAWEE_CI_LOCK_PROJECT / _SESSION"; return
    fi
    grep -q 'CLAWEE_CI_RUN_ID=' "$f" || { fail $t "launch does not export the run id"; return; }
    [ "$RC" = 0 ] || { fail $t "a passing suite exited $RC: $(tail -3 "$STUB_DIR/err")"; return; }
    pass $t
}

test_no_command_names_the_brand_lock() {
    local t=no_command_names_the_brand_lock hits
    run_runner 0
    hits="$(grep -l -e '/tmp/ci-lock' -e 'heartbeat' "$STUB_DIR"/call.* 2>/dev/null)"
    [ -z "$hits" ] || { fail $t "machine commands name the hand-rolled lock: $hits"; return; }
    hits="$(grep -n -e '/tmp/ci-lock' -e 'heartbeat' -e 'HEARTBEAT' -e 'STALE' "$RUNNER")"
    [ -z "$hits" ] || { fail $t "the runner still names the hand-rolled lock: $(printf '%s' "$hits" | head -3 | tr '\n' ' ')"; return; }
    pass $t
}

test_help_describes_the_product_lock() {
    local t=help_describes_the_product_lock out
    out="$("$RUNNER" --help)"
    case "$out" in *CLAWEE_CI_LOCK_HEARTBEAT_S* | */tmp/ci-lock*) fail $t "--help still names the hand-rolled lock"; return ;; esac
    case "$out" in *"ci-lock run $PRODUCT"*) ;; *) fail $t "--help does not name ci-lock run $PRODUCT"; return ;; esac
    case "$out" in *"75 "*) ;; *) fail $t "--help does not list exit 75"; return ;; esac
    pass $t
}

test_lock_timeout_is_reported() {
    local t=lock_timeout_is_reported
    run_runner lock75
    [ "$RC" = 75 ] || { fail $t "a lock timeout exited $RC, not 75: $(tail -3 "$STUB_DIR/err")"; return; }
    grep -Eq "the $PRODUCT lock was not acquired within [0-9]+s" "$STUB_DIR/out" "$STUB_DIR/err" ||
        { fail $t "no timeout message: $(tail -3 "$STUB_DIR/err")"; return; }
    pass $t
}

test_unprovisioned_lock_names_install() {
    local t=unprovisioned_lock_names_install
    run_runner lock2
    [ "$RC" = 1 ] || { fail $t "an unprovisioned lock exited $RC, not 1 (2 is usage here): $(tail -3 "$STUB_DIR/err")"; return; }
    if ! grep -q "$PRODUCT is not provisioned on burrowee-ci" "$STUB_DIR/out" "$STUB_DIR/err" ||
        ! grep -q 'ci-lock install' "$STUB_DIR/out" "$STUB_DIR/err"; then
        fail $t "no not-provisioned message naming ci-lock install: $(tail -3 "$STUB_DIR/err")"; return
    fi
    pass $t
}

test_each_run_gets_its_own_tree() {
    local t=each_run_gets_its_own_tree f tree id
    run_runner 0
    f="$(grep -l -- "--copy-dest=$SEED\$" "$STUB_DIR"/call.*.rsync 2>/dev/null | head -1)"
    [ -n "$f" ] || { fail $t "the tree sync is not seeded with --copy-dest=$SEED"; return; }
    if grep -q -- '--link-dest' "$STUB_DIR"/call.* "$RUNNER"; then
        fail $t "a sync still hard-links with --link-dest; the seed and the run trees must not share inodes"; return
    fi
    tree="$(sed -n "s|^burrowee-ci:\\($SEED\\.t-[A-Za-z0-9_-]*\\)/\$|\\1|p" "$f")"
    [ -n "$tree" ] || { fail $t "the tree sync does not target a per-run tree $SEED.t-<run id>: $(tr '\n' ' ' < "$f")"; return; }
    id="${tree#"$SEED".t-}"
    grep -l -F "t=$tree id=$id;" "$STUB_DIR"/call.*.ssh | xargs grep -l -F 'rm -rf -- "$t" "$t.run" "$t.deps"' >/dev/null 2>&1 ||
        { fail $t "no removal call 't=$tree id=$id' with rm -rf -- \"\$t\" \"\$t.run\" \"\$t.deps\""; return; }
    pass $t
}

test_product_name_matches_registry() {
    local t=product_name_matches_registry want got
    if ! command -v ci-lock >/dev/null || [ ! -f "${CODING_ROOT:-/nonexistent}/PROJECTS.md" ]; then
        printf 'skip %s: ci-lock is not on PATH or $CODING_ROOT/PROJECTS.md is missing; owner: the operator (Workstation bin on PATH, CODING_ROOT set); re-enable: both present\n' $t
        return
    fi
    want="$(sed -n 's/^CI_LOCK_PRODUCT=\([a-z0-9-]*\)$/\1/p' "$RUNNER")"
    got="$(ci-lock products --path "$CHECKOUT")"
    [ -n "$want" ] && [ "$want" = "$got" ] ||
        { fail $t "CI_LOCK_PRODUCT '$want' != ci-lock products --path: '$got'"; return; }
    pass $t
}

stop_call() {
    grep -l 'survivor' "$STUB_DIR"/call.*.ssh 2>/dev/null | head -1
}

sweep_setup() {
    local f
    run_runner 0
    f="$(stop_call)"
    [ -n "$f" ] || return 1
    SWEEP_ID="$(sed -n 's/.* id=\([A-Za-z0-9_-]*\);.*/\1/p' "$f" | head -1)"
    FAKE="$WORK/proc.$1"
    rm -rf "$FAKE"; mkdir -p "$FAKE/101" "$FAKE/102" "$FAKE/103"
    printf 'A=1\0CLAWEE_CI_RUN_ID=%s\0' "$SWEEP_ID" > "$FAKE/101/environ"
    printf 'CLAWEE_CI_RUN_ID=%sx\0' "$SWEEP_ID" > "$FAKE/102/environ"
    printf 'B=2\0' > "$FAKE/103/environ"
    printf 'ssh\0-N\0' > "$FAKE/101/cmdline"
    sed "s|^proc=/proc\$|proc=$FAKE|" "$f" > "$WORK/stop.$1.sh"
    grep -q "^proc=$FAKE\$" "$WORK/stop.$1.sh"
}

run_sweep() {
    local dies_on="$1" script="$2"
    SWEEP_RC=0
    SWEEP_OUT="$(DIES_ON="$dies_on" FAKE="$FAKE" bash -c '
        kill() { echo "kill $*" >> "$FAKE/kills"; local p; for p in "$@"; do case "$p" in -*) [ "$p" = "$DIES_ON" ] || return 0 ;; *) rm -rf "${FAKE:?}/$p" ;; esac; done; }
        sleep() { :; }
        systemctl() { :; }
        pgrep() { return 1; }
        source "$0"' "$script" 2>&1)" || SWEEP_RC=$?
}

test_stop_sweeps_survivors_by_run_id() {
    local t=stop_sweeps_survivors_by_run_id
    sweep_setup term || { fail $t "no stop call with a run-id survivor sweep is sent after a passing run"; return; }
    run_sweep -TERM "$WORK/stop.term.sh"
    [ "$SWEEP_RC" = 0 ] || { fail $t "sweep exited $SWEEP_RC: $SWEEP_OUT"; return; }
    [ "$(cat "$FAKE/kills")" = "kill -TERM 101" ] || { fail $t "kills were: $(tr '\n' ';' < "$FAKE/kills")"; return; }
    [ -d "$FAKE/102" ] && [ -d "$FAKE/103" ] || { fail $t "a process of another run was killed"; return; }
    case "$SWEEP_OUT" in *"pid 101"*ssh*) ;; *) fail $t "the sweep does not report what it killed: $SWEEP_OUT"; return ;; esac
    pass $t
}

test_stop_kills_a_survivor_that_ignores_term() {
    local t=stop_kills_a_survivor_that_ignores_term
    sweep_setup kill || { fail $t "no stop call with a run-id survivor sweep is sent after a passing run"; return; }
    run_sweep -KILL "$WORK/stop.kill.sh"
    [ "$SWEEP_RC" = 0 ] || { fail $t "sweep exited $SWEEP_RC: $SWEEP_OUT"; return; }
    grep -qx 'kill -KILL 101' "$FAKE/kills" || { fail $t "no KILL after TERM was ignored: $(tr '\n' ';' < "$FAKE/kills")"; return; }
    pass $t
}

test_stop_reports_a_survivor_that_outlives_kill() {
    local t=stop_reports_a_survivor_that_outlives_kill
    sweep_setup immortal || { fail $t "no stop call with a run-id survivor sweep is sent after a passing run"; return; }
    run_sweep -NONE "$WORK/stop.immortal.sh"
    [ "$SWEEP_RC" = 1 ] || { fail $t "a survivor of KILL exited $SWEEP_RC, not 1: $SWEEP_OUT"; return; }
    pass $t
}

test_stop_refuses_an_empty_run_id() {
    local t=stop_refuses_an_empty_run_id
    sweep_setup empty || { fail $t "no stop call with a run-id survivor sweep is sent after a passing run"; return; }
    sed -i.bak "s/ id=$SWEEP_ID;/ id=;/" "$WORK/stop.empty.sh"
    run_sweep -TERM "$WORK/stop.empty.sh"
    [ "$SWEEP_RC" != 0 ] && [ ! -e "$FAKE/kills" ] || { fail $t "an empty id was not refused (exit $SWEEP_RC)"; return; }
    pass $t
}

test_launch_contains_the_run_in_a_user_scope() {
    local t=launch_contains_the_run_in_a_user_scope f
    run_runner 0
    f="$(launch_call)"
    grep -q 'systemd-run --user --scope' "$f" || { fail $t "the launch does not use a per-run user scope"; return; }
    grep -q 'CLAWEE_CI_RUN_DIR' "$f" && { fail $t "the launch exports a run-dir variable into the suite"; return; }
    pass $t
}

prep_tree() {
    local id="$1" age="$2"
    [ -e "$PREP_SEED.t-$id" ] || mkdir -p "$PREP_SEED.t-$id"
    [ "$age" = none ] && return
    mkdir -p "$PREP_SEED.t-$id.run"
    printf '%s\n' "$id" > "$PREP_SEED.t-$id.run/id"
    [ "$age" = old ] && touch -t 202001010000 "$PREP_SEED.t-$id.run/id"
}

test_prep_removes_only_proven_own_trees() {
    local t=prep_removes_only_proven_own_trees f me kept
    run_runner 0
    f="$(grep -l 'removed the finished run tree' "$STUB_DIR"/call.*.ssh 2>/dev/null | head -1)"
    [ -n "$f" ] || { fail $t "no stale-tree cleanup call is sent before the sync"; return; }
    me="$(id -un | tr -c 'A-Za-z0-9\n' '_')"
    rm -rf "$PREP_SEED" "$PREP_SEED".t-*
    prep_tree "$me-1-100" old
    prep_tree "$me-2-200" young
    prep_tree "$me-3-300" none
    prep_tree "$me-4-400" old
    prep_tree "other-5-500" old
    ln -s / "$PREP_SEED.t-$me-6-600"
    prep_tree "$me-6-600" old
    mkdir -p "$WORK/pproc/77"
    printf 'CLAWEE_CI_RUN_ID=%s\0' "$me-4-400" > "$WORK/pproc/77/environ"
    sed -e "s|^seed=$SEED |seed=$PREP_SEED |; s| seeddeps=$SEED.deps | seeddeps=$PREP_SEED.deps |; s|^proc=/proc\$|proc=$WORK/pproc|" "$f" > "$WORK/prep.sh"
    grep -q "^proc=$WORK/pproc\$" "$WORK/prep.sh" || { fail $t "the cleanup call has no proc=/proc line to point at a fake /proc"; return; }
    bash "$WORK/prep.sh" > "$WORK/prep.out" 2>&1 || { fail $t "the cleanup exited non-zero: $(cat "$WORK/prep.out")"; return; }
    [ ! -e "$PREP_SEED.t-$me-1-100" ] && [ ! -e "$PREP_SEED.t-$me-1-100.run" ] ||
        { fail $t "a finished own tree with its id file was not removed: $(cat "$WORK/prep.out")"; return; }
    for kept in "$me-2-200" "$me-3-300" "$me-4-400" "other-5-500" "$me-6-600"; do
        [ -e "$PREP_SEED.t-$kept" ] || { fail $t "tree $kept was removed without proof: $(cat "$WORK/prep.out")"; return; }
    done
    pass $t
}

wrapper_run() {
    local mode="$1" suite="$2" f locked
    run_runner 0
    f="$(launch_call)"
    locked="$(sed -n "s/^locked='\\(.*\\)'\$/\\1/p" "$f")"
    [ -n "$locked" ] || return 1
    mkdir -p "$WORK/fakebin"
    printf '#!/usr/bin/env bash\ncase "$FAKE_MODE" in 75) exit 75 ;; 2) exit 2 ;; esac\nwhile [ "$1" != -- ]; do shift; done; shift; exec "$@"\n' > "$WORK/fakebin/ci-lock"
    chmod +x "$WORK/fakebin/ci-lock"
    WRAP_DIR="$WORK/wrap.$mode"
    rm -rf "$WRAP_DIR"; mkdir -p "$WRAP_DIR"
    printf '%s\n' "$suite" > "$WRAP_DIR/run.sh"
    WRAP_RC=0
    FAKE_MODE="$mode" bash -c "${locked//\/usr\/local\/bin\/ci-lock/$WORK/fakebin/ci-lock}" ci-lock-run p s "$WRAP_DIR" || WRAP_RC=$?
}

test_wrapper_records_lock_exit_only_when_the_suite_never_started() {
    local t=wrapper_records_lock_exit_only_when_the_suite_never_started mode
    for mode in 75 2; do
        wrapper_run "$mode" 'exit 0' || { fail $t "the launch has no locked='…' wrapper line"; return; }
        [ "$WRAP_RC" = "$mode" ] && [ "$(cat "$WRAP_DIR/lock_rc" 2>/dev/null)" = "$mode" ] ||
            { fail $t "ci-lock exit $mode: wrapper exit $WRAP_RC, lock_rc '$(cat "$WRAP_DIR/lock_rc" 2>/dev/null)'"; return; }
    done
    wrapper_run run 'd=$(dirname "$0"); : > "$d/started"; echo 0 > "$d/run.rc"'
    [ "$WRAP_RC" = 0 ] && [ ! -e "$WRAP_DIR/lock_rc" ] ||
        { fail $t "a suite that ran: wrapper exit $WRAP_RC, lock_rc present: $([ -e "$WRAP_DIR/lock_rc" ] && echo yes)"; return; }
    wrapper_run died 'd=$(dirname "$0"); : > "$d/started"; exit 2'
    [ "$WRAP_RC" = 2 ] && [ ! -e "$WRAP_DIR/lock_rc" ] ||
        { fail $t "a suite that died without rc: wrapper exit $WRAP_RC, lock_rc '$(cat "$WRAP_DIR/lock_rc" 2>/dev/null)'"; return; }
    pass $t
}

test_suite_script_marks_started_first() {
    local t=suite_script_marks_started_first f first
    run_runner 0
    f="$(launch_call)"
    f="${f%.ssh}.stdin"
    first="$(grep -v '^case ' "$f" | grep -v '^$' | head -1)"
    case "$first" in ": > "*"/started") ;; *) fail $t "the suite's first command is not the started marker: '$first'"; return ;; esac
    pass $t
}

test_suite_dead_without_status_is_not_a_lock_failure() {
    local t=suite_dead_without_status_is_not_a_lock_failure
    run_runner ''
    [ "$RC" = 1 ] || { fail $t "a suite gone without a status exited $RC, not 1"; return; }
    grep -q 'wrote no status' "$STUB_DIR/err" || { fail $t "no 'wrote no status' message: $(tail -3 "$STUB_DIR/err")"; return; }
    if grep -q -e 'not provisioned' -e 'not acquired' -e 'ci-lock exited' "$STUB_DIR/out" "$STUB_DIR/err"; then
        fail $t "a dead suite is reported as a lock failure"; return
    fi
    pass $t
}

test_stop_terms_the_group_first() {
    local t=stop_terms_the_group_first f id run
    run_runner 0
    f="$(stop_call)"
    [ -n "$f" ] || { fail $t "no stop call"; return; }
    id="$(sed -n 's/.* id=\([A-Za-z0-9_-]*\);.*/\1/p' "$f" | head -1)"
    run="$(sed -n 's/.*run=\([^ ]*\) id=.*/\1/p' "$f" | head -1)"
    FAKE="$WORK/proc.group"
    rm -rf "$FAKE" "$run"; mkdir -p "$FAKE" "$run"
    printf '%s\n' "$id" > "$run/id"
    echo 4242 > "$run/run.pid"
    sed "s|^proc=/proc\$|proc=$FAKE|" "$f" > "$WORK/stop.group.sh"
    SWEEP_RC=0
    SWEEP_OUT="$(FAKE="$FAKE" bash -c '
        kill() { echo "kill $*" >> "$FAKE/kills"; }
        sleep() { :; }
        systemctl() { :; }
        pgrep() { grep -q -- "-TERM -- -4242" "$FAKE/kills" 2>/dev/null && return 1; return 0; }
        source "$0"' "$WORK/stop.group.sh" 2>&1)" || SWEEP_RC=$?
    rm -rf "$run"
    [ "$SWEEP_RC" = 0 ] || { fail $t "stop exited $SWEEP_RC: $SWEEP_OUT"; return; }
    [ "$(cat "$FAKE/kills")" = "kill -TERM -- -4242" ] || { fail $t "kills were: $(tr '\n' ';' < "$FAKE/kills")"; return; }
    pass $t
}

test_stop_retries_a_refused_connection() {
    local t=stop_retries_a_refused_connection
    STUB_STOP_255=1 run_runner 0
    [ "$(cat "$STUB_DIR/stops" 2>/dev/null)" = 2 ] || { fail $t "the stop was sent $(cat "$STUB_DIR/stops" 2>/dev/null) times after one 255, not 2"; return; }
    [ "$RC" = 0 ] || { fail $t "a passing run exited $RC after one refused stop"; return; }
    pass $t
}

test_unconfirmed_stop_keeps_the_tree_and_fails_the_run() {
    local t=unconfirmed_stop_keeps_the_tree_and_fails_the_run
    STUB_STOP_FAIL=1 run_runner 0
    [ "$RC" = 1 ] || { fail $t "a passing run whose stop failed exited $RC, not 1"; return; }
    grep -q 'could not confirm the run on burrowee-ci stopped' "$STUB_DIR/err" && grep -q 'run id:' "$STUB_DIR/err" ||
        { fail $t "no unconfirmed-stop report: $(tail -3 "$STUB_DIR/err")"; return; }
    if grep -l -F 'is not the tree of run' "$STUB_DIR"/call.*.ssh >/dev/null 2>&1; then
        fail $t "the tree of an unconfirmed run was removed"; return
    fi
    pass $t
}

group_stop_with_pgrep() {
    local code="$1" f id run
    run_runner 0
    f="$(stop_call)"
    [ -n "$f" ] || return 1
    id="$(sed -n 's/.* id=\([A-Za-z0-9_-]*\);.*/\1/p' "$f" | head -1)"
    run="$(sed -n 's/.*run=\([^ ]*\) id=.*/\1/p' "$f" | head -1)"
    FAKE="$WORK/proc.pgrep$code"
    rm -rf "$FAKE" "$run"; mkdir -p "$FAKE" "$run"
    printf '%s\n' "$id" > "$run/id"
    echo 4242 > "$run/run.pid"
    printf 'clawee-go-headless-term-run-%s.scope\n' "$id" > "$run/scope"
    sed "s|^proc=/proc\$|proc=$FAKE|" "$f" > "$WORK/stop.pgrep$code.sh"
    SWEEP_RC=0
    SWEEP_OUT="$(FAKE="$FAKE" CODE="$code" bash -c '
        kill() { echo "kill $*" >> "$FAKE/kills"; }
        sleep() { :; }
        systemctl() { echo "systemctl $*" >> "$FAKE/kills"; }
        pgrep() { [ -s "$FAKE/kills" ] && return "$CODE"; return 0; }
        source "$0"' "$WORK/stop.pgrep$code.sh" 2>&1)" || SWEEP_RC=$?
    rm -rf "$run"
}

test_stop_counts_only_pgrep_1_as_gone() {
    local t=stop_counts_only_pgrep_1_as_gone code
    for code in 2 3; do
        group_stop_with_pgrep "$code" || { fail $t "no stop call"; return; }
        [ "$SWEEP_RC" = 1 ] || { fail $t "pgrep exit $code after TERM: stop exited $SWEEP_RC, not 1"; return; }
        case "$SWEEP_OUT" in *"not confirmed gone"*) ;; *) fail $t "no 'not confirmed gone' line: $SWEEP_OUT"; return ;; esac
    done
    pass $t
}

test_stop_stops_the_run_scope_after_proof() {
    local t=stop_stops_the_run_scope_after_proof
    group_stop_with_pgrep 1 || { fail $t "no stop call"; return; }
    [ "$SWEEP_RC" = 0 ] || { fail $t "stop exited $SWEEP_RC: $SWEEP_OUT"; return; }
    grep -q '^systemctl --user stop clawee-go-headless-term-run-.*\.scope$' "$FAKE/kills" ||
        { fail $t "the run's scope was not stopped: $(tr '\n' ';' < "$FAKE/kills")"; return; }
    pass $t
}

test_go_failure_codes_exit_1() {
    local t=go_failure_codes_exit_1 code
    for code in 2 75; do
        run_runner "$code"
        [ "$RC" = 1 ] || { fail $t "a suite rc $code exited $RC, not 1"; return; }
        if grep -q -e 'not provisioned' -e 'not acquired' -e 'ci-lock exited' "$STUB_DIR/out" "$STUB_DIR/err"; then
            fail $t "a suite rc $code is reported as a lock failure"; return
        fi
    done
    pass $t
}

write_stubs
test_launch_runs_under_product_lock
test_no_command_names_the_brand_lock
test_help_describes_the_product_lock
test_lock_timeout_is_reported
test_unprovisioned_lock_names_install
test_each_run_gets_its_own_tree
test_product_name_matches_registry
test_stop_sweeps_survivors_by_run_id
test_stop_kills_a_survivor_that_ignores_term
test_stop_reports_a_survivor_that_outlives_kill
test_stop_refuses_an_empty_run_id
test_launch_contains_the_run_in_a_user_scope
test_prep_removes_only_proven_own_trees
test_wrapper_records_lock_exit_only_when_the_suite_never_started
test_suite_script_marks_started_first
test_suite_dead_without_status_is_not_a_lock_failure
test_stop_terms_the_group_first
test_stop_retries_a_refused_connection
test_unconfirmed_stop_keeps_the_tree_and_fails_the_run
test_stop_counts_only_pgrep_1_as_gone
test_stop_stops_the_run_scope_after_proof
test_go_failure_codes_exit_1
exit "$FAILED"
