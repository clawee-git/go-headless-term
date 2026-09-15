# go-headless-term

## Tests

The suite is **`ci/run-tests.sh [options] [pkg...]`**, and it runs on the shared CI machine
(`burrowee-ci`), never on the workstation. Packages default to `./...` — the root module:
`github.com/clawee-git/go-headless-term` and `internal/generate_width_table`. `wasm/` is a
separate `js/wasm` module with no tests and is not part of the run.

| Invocation | What it does |
|---|---|
| `ci/run-tests.sh` | the plain suite: `go build ./...`, then `go test -count=1 ./...` — land gates and the runtime of record |
| `ci/run-tests.sh ./internal/...` | the same, for named packages — the red/green loop |
| `ci/run-tests.sh --artifacts <dir>` | **evidence mode**: `-json` with a set-mode profile over `./...` (whatever packages are named); the log adds per-package and module test/case counts with skips, each skipped and failed name; `test.json`, `cover.out` and `covered.txt` (the sorted covered set) are copied to `<dir>`, which must not exist |
| `ci/run-tests.sh --shuffle --repeat <n> [--artifacts <dir>]` | evidence mode shuffled (`-shuffle=on`, seed per package in the log) and repeated (`-count=<n>`) |

- **The Clawee CI lock** (`/tmp/ci-lock/clawee` on the machine) is taken by every invocation and
  released on success, failure and interrupt. A held lock exits `3` with its holder, heartbeat age
  and stale threshold; the script never breaks one. Set `CI_LOCK_PROJECT` and `CI_LOCK_SESSION`
  so the holder names you.
- The suite runs detached on the machine and is followed over short ssh polls, so a dropped
  connection is not a result. Exit status: the suite's; `2` usage; `3` lock; `1` machine
  unreachable, no status, or evidence not copied home. `ci/run-tests.sh --help` has the rest.
- Cross-module coverage: a local `go.work` is mirrored to the machine, and in evidence mode each of
  its modules joins `-coverpkg`. This module imports no other Clawee module, so code here reached
  only by another module's tests (`cli` pins this one) is measured by **that** module's runner with
  this checkout in its `go.work`.
- Workstation checks before a run: `gofmt -s -w .` and `GOOS=linux go build ./...`.
