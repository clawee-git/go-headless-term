# go-headless-term

## Tests

The suite is **`ci/run-tests.sh [options] [pkg...]`**, and it runs on the shared CI machine
(`burrowee-ci`), never on the workstation. Packages default to `./...` — the root module's three:
`github.com/clawee-git/go-headless-term`, `internal/generate_width_table`, and `examples/basic`
(no test files). `wasm/` is a separate `js/wasm` module with no tests and is not part of the run.

| Invocation | What it does |
|---|---|
| `ci/run-tests.sh` | the plain suite: `go build ./...`, then `go test -count=1 ./...` — land gates and the runtime of record |
| `ci/run-tests.sh ./internal/...` | the same, for named packages — the red/green loop |
| `ci/run-tests.sh --artifacts <dir>` | **evidence mode**: `-json` with a set-mode profile over `./...` (whatever packages are named); the log adds per-package and module test/case counts with skips, each skipped and failed name; `test.json`, `cover.out` and `covered.txt` are copied to `<dir>`, which must not exist. `covered.txt` is the sorted covered set without `<pkg>_test/` support packages. Named packages without test files (`examples/basic`) are left out of this run and listed in the log |
| `ci/run-tests.sh --shuffle --repeat <n> [--artifacts <dir>]` | evidence mode shuffled (`-shuffle=on`, seed per package in the log) and repeated (`-count=<n>`) |

- **The Clawee CI lock** (`/tmp/ci-lock/clawee` on the machine) is taken by every invocation and
  released on success, failure, interrupt and a closed stdout. **One exception:** when the remote
  run's process group cannot be confirmed dead, the lock is deliberately left held (it goes STALE)
  so no second run starts beside a live one; the script prints the run id, the run's
  `pid`/`log`/`rc` files on the machine, a check command, and the release command guarded by that
  run id — run the release only once the check shows nothing running. A held lock exits `3` with its holder, heartbeat age
  and stale threshold; the script never waits for or breaks one. Set `CLAWEE_CI_LOCK_PROJECT` and
  `CLAWEE_CI_LOCK_SESSION` so the holder names you.
- Environment: `CLAWEE_CI_MACHINE`, `CLAWEE_CI_DIR`, `CLAWEE_CI_LOCK_PROJECT`,
  `CLAWEE_CI_LOCK_SESSION`, `CLAWEE_CI_LOCK_HEARTBEAT_S`, `CLAWEE_CI_POLL_S`,
  `CLAWEE_CI_FOLLOW_MAX_MISSES`; the numbers must be whole numbers of at least 1, and
  `CLAWEE_CI_DIR` must be `/tmp/clawee-ght-<name>` (letters, digits, `._-`, no `..`). Every command
  that deletes or overwrites on the machine refuses any path outside that prefix or the lock.
- The suite runs detached on the machine and is followed over short ssh polls, so a dropped
  connection is not a result. A signal (INT/TERM/HUP, also to the script's pid alone) stops the
  remote run and releases the lock, at once while the run is followed. These remote steps are
  bounded and finish before the signal is acted on: the probe, the lock take, the sync, the launch
  and the evidence copy-back. The stop that follows the signal can take up to ~22 s before the
  release. Once that teardown (stop, then release) has begun, further signals are ignored and the
  exit is the run's own status.
  Exit status, unchanged by a closed stderr:
  - `0`: build and tests passed **and** this run's lock was released — or the release found the
    lock no longer naming this run (warned), so this run holds none;
  - `1`: build or test failure, machine unreachable, `/tmp/ci-lock` missing on the machine (never
    created here), no status, evidence not copied home — **or a passing run whose lock is or may be left held** (unconfirmed remote kill,
    unanswered release; the recovery commands are on stderr);
  - `2`: usage, refused before any contact — including a bad environment value (`CLAWEE_CI_DIR`
    outside `/tmp/clawee-ght-<name>`, or a `<name>` ending in `.`), a derived remote path the
    machine guard would refuse, and a local `go.work` whose `use` is not a directory or names a
    module that cannot be mirrored safely (empty, a `.` segment, `..` anywhere);
  - `3`: the lock is held by another run, was released while this run checked it, or its root is
    not writable;
  - `130`/`143`/`129`: interrupted; `141`: stdout closed with SIGPIPE — after the stop and release.

  The lock outcome never replaces a failing status. A signal or SIGPIPE replaces the run's status
  only when it lands before the teardown (the copy-back included); during the teardown it is
  ignored. A closed stdout without SIGPIPE (`>&-`, or SIGPIPE ignored) does
  not stop the run: messages continue on stderr and the status is the run's. The printed release
  exits `3` when the lock is not this run's and `0` when it released it. `ci/run-tests.sh --help`
  has the rest.
- Cross-module coverage: none from this runner. A local `go.work` is mirrored to the machine, but
  never widens `-coverpkg`. This module imports no other Clawee module; code here reached only by
  another module's tests (`cli` pins this one) is measured by **that** module's runner, with
  `--cover-module github.com/clawee-git/go-headless-term` and this checkout in its `go.work`.
- Workstation checks before a run: `gofmt -s -w .`, `goimports -w .` and `GOOS=linux go build ./...`.
