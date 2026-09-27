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

- **It runs under the product lock `clawee-go-headless-term`.** The detached launch on the
  machine is `/usr/local/bin/ci-lock run clawee-go-headless-term --timeout 600 --project
  <CLAWEE_CI_LOCK_PROJECT> --session <CLAWEE_CI_LOCK_SESSION> -- bash <run script>`: the product
  lock exclusive and a shared hold on `clawee`, both kernel locks held by the run's processes and
  released when the last one exits — no take, heartbeat or release step, nothing stale. Other
  Clawee products run beside it. The name is the constant `CI_LOCK_PRODUCT`, and
  `ci/run-tests_test.sh` asserts it equals `ci-lock products --path <checkout>`. A held lock is
  **waited on**, up to the suite bound (600 s, go test's default `-timeout`); ci-lock's `waiting`
  / `acquired` lines are in the followed log, and `ssh burrowee-ci ci-lock status
  clawee-go-headless-term` names the holder. Not acquired in time → **exit 75**; not provisioned
  → exit 1, naming `ci-lock install` (the operator's). The run script writes a `started` marker
  first, so a suite that dies without a status is reported as that, never as a lock failure.
- **A tree per run.** `CLAWEE_CI_DIR` (default `/tmp/clawee-ght-<user>-<cksum of checkout>`) is
  the checkout's SEED; each run syncs into its own `<seed>.t-<run id>` (plus `.run/` for its
  files and `.deps` for go.work modules) with `rsync --copy-dest=<seed>` (never `--link-dest`:
  every tree holds its own files), refreshes the seed from it, and removes it when it ends.
  Before syncing, a run removes earlier trees of this checkout only when they are PROVABLY its
  account's finished runs: owned by this account, named for this user's run id, whose `.run/id`
  holds that id (written over a minute ago), and with no process on the machine carrying that
  `CLAWEE_CI_RUN_ID`. An empty run id is refused on the workstation and the machine.
- **Containment.** The run id is exported on the machine as `CLAWEE_CI_RUN_ID` to every process
  of the run, and nothing else of the runner's is. Where the user manager runs (`systemctl
  --user is-system-running` says running or degraded) and `systemd-run` exists, the run starts in
  its own user scope `clawee-go-headless-term-run-<id>.scope`. The stop runs at the end of every
  launched run (retried on ssh 255, 3 times): it TERMs the process group (KILL after 20 s), stops
  that scope, then SWEEPS by exact run id — every process whose `/proc/<pid>/environ` holds
  `CLAWEE_CI_RUN_ID=<id>` gets TERM, and KILL after 10 s, each named on stderr. Without a user
  manager the group kill and the sweep remain; a helper that also drops the variable is the
  stated gap. A stop that cannot be confirmed is reported with the run id, its files and a check
  command; its tree is left, and while any of its processes live they hold the lock — never
  break it, stop the run.
- Environment: `CLAWEE_CI_MACHINE`, `CLAWEE_CI_DIR`, `CLAWEE_CI_LOCK_PROJECT`,
  `CLAWEE_CI_LOCK_SESSION`, `CLAWEE_CI_POLL_S`, `CLAWEE_CI_FOLLOW_MAX_MISSES`; the numbers must
  be whole numbers of at least 1, and `CLAWEE_CI_DIR` must be `/tmp/clawee-ght-<name>` (letters,
  digits, `._-`, no `..`, not ending in `.`). Every command that deletes or overwrites on the
  machine refuses any path outside that prefix.
- The suite runs detached on the machine and is followed over short ssh polls, so a dropped
  connection is not a result. A signal (INT/TERM/HUP, also to the script's pid alone) stops the
  remote run at once while the run is followed, the wait for the lock included. These remote
  steps are bounded and finish before the signal is acted on: the probe, the stale-tree cleanup,
  the sync, the launch and the evidence copy-back. The stop can take up to ~35 s. Once the
  teardown (stop, then the tree's removal) has begun, further signals are ignored and the exit is
  the run's own status.
  Exit status, unchanged by a closed stderr:
  - `0`: build and tests passed;
  - `1`: build or test failure, machine unreachable, `clawee-go-headless-term` not provisioned on
    the machine, no status, evidence not copied home — **or a passing run whose stop could not
    be confirmed** (the check command is on stderr);
  - `2`: usage, refused before any contact — including a bad environment value (`CLAWEE_CI_DIR`
    outside `/tmp/clawee-ght-<name>`, or a `<name>` ending in `.`), a derived remote path the
    machine guard would refuse, and a local `go.work` whose `use` is not a directory or names a
    module that cannot be mirrored safely (empty, a `.` segment, `..` anywhere);
  - `75`: the `clawee-go-headless-term` lock was not acquired within the suite bound;
  - `130`/`143`/`129`: interrupted; `141`: stdout closed with SIGPIPE — after the stop.

  A signal or SIGPIPE replaces the run's status only when it lands before the teardown (the
  copy-back included); during the teardown it is ignored. A closed stdout without SIGPIPE
  (`>&-`, or SIGPIPE ignored) does not stop the run: messages continue on stderr and the status
  is the run's. `ci/run-tests.sh --help` has the rest.
- The runner's own suite is `bash ci/run-tests_test.sh`: it runs on the workstation with `ssh`
  and `rsync` stubbed on `PATH`, runs the real wrapper, stop and stale-tree scripts against a
  fake `ci-lock`, `/proc` and seed, and never contacts the machine.
- Cross-module coverage: none from this runner. A local `go.work` is mirrored to the machine, but
  never widens `-coverpkg`. This module imports no other Clawee module; code here reached only by
  another module's tests (`cli` pins this one) is measured by **that** module's runner, with
  `--cover-module github.com/clawee-git/go-headless-term` and this checkout in its `go.work`.
- Workstation checks before a run: `gofmt -s -w .`, `goimports -w .` and `GOOS=linux go build ./...`.
