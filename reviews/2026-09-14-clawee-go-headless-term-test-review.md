# Test-suite review — go-headless-term

Module: `github.com/clawee-git/go-headless-term` · baseline commit `a2b682e` · after commit `b2dd063` (test-code head; the report and evidence commits follow it) · verified at `32e1182` · suite command: `ci/run-tests.sh [options] [pkg...]` · target: `burrowee-ci` · runtime = plain uninstrumented run · covered = set mode profile

**Revision 2 (2026-09-17)** — this report was reviewed at `a2b682e..c412993`
(`../reviews/2026-09-17-clawee-go-headless-term-test-suite-review-02-review.md` in the
resources repo) and came back NOT CLEAN with twelve findings. All twelve are carried out here:
three MERGE mutations taken at head, the non-compiling commit window ruled on, the `After`
table added with the honest line delta, the two false statements corrected, seventeen missing
verdict rows added, and every row that was recorded `TABLE` but executed as a named subtest
func either folded into a real data table or re-verdicted `RENAME`. Revision-2 commits are
`1c024cb`, `f8679b4`, `b2dd063` and this one.

**Revision 3 (2026-09-17)** — `burrowee-ci` came back, and the verify phase revision 2 could not
run was run: one control evidence run, **six** mutation runs (the three MERGE rows, two forms
each — drop-the-case for coverage, break-the-code for behaviour) and the three verify runs
(plain, `--shuffle --repeat 3`, `--repeat 3`), all at head `32e1182`. Every §5 gate is recomputed
against those runs and quoted in `Gates`; the `After` table is filled from them; both "STALE —
re-run pending" markers are gone. `TestMiddlewareMergeSetUserVar` (M3) had never been mutated by
anyone and is now measured: it behaves exactly as M1 does, for the same production reason. No
gate failed and no verdict moved.

## Rulings (finishing session, 2026-09-15)

**Branch-local gofmt commit excised.** The branch as left by the prior workers opened with
`9289383 "style: apply gofmt to existing production code"` (providers.go, terminal.go) between the
approved 01 head `a2b682e` and the audit commits. `origin/dev` (`d68b769`) still carries the same
unformatted tree, so the unformatted state is a pre-existing dev condition, not something this
audit introduced or may fix: the spec gate requires `git diff --stat <from>..<to> -- ':!*_test.go'
':!**/testdata/**' ':!reviews/**'` to be empty, and a "documented environment prerequisite"
justification would still leave a production-file diff in the range. The commit was excised
(`git rebase --onto a2b682e 9289383`; nothing was pushed, the rebase applied cleanly because every
audit commit touches only `*_test.go`). The baseline covered set was re-taken at `a2b682e` because
`9289383`'s blank-line removal in `providers.go` had shifted that file's block keys down one line
relative to dev (baseline entries at `providers.go:125+`); the old `9289383`-era artifacts are kept
under `reviews/2026-09-14-clawee-go-headless-term-coverage/` for provenance. The mutation evidence
(mutate-clipboard, mutate-charset) is unaffected: its lost blocks are in `terminal.go:266+` and
`handler.go`, and `9289383`'s terminal.go change was a same-line reindent that moves no block keys.
Consequence for the workstation checklist: `gofmt -s -l .` on this branch reports exactly
`providers.go` and `terminal.go` — dev's pre-existing state, deliberately not "fixed" here.

**Mixed-verdict commits `046bba9`, `6add4f7` (TABLE + REWRITE) and `0eb7472`, `79f2452`, `34b34eb`
(TABLE + RENAME) left as is.** This ruling has now been under-inclusive twice, which is itself the
lesson. Revision 1 named `046bba9` as *the* mixed-verdict commit; the 02-audit review found a second,
`6add4f7`. Revision 2's re-verdict then created three more without extending the ruling — round 2 of
the review caught that — because relabelling twenty rows from `TABLE` to `RENAME` retroactively made
the commits that executed them mixed-kind: `0eb7472` (kitty) carries 4 RENAME rows beside the
`parseKittyGraphicsCases` TABLE; `34b34eb` (terminal) carries 10 — the five resize-scrollback, three
recording and two row-conversion rows — beside its tables and the resize SPLIT; `79f2452`
(semantic-prompt) carries the two `TestSemanticPromptMark_ScrollbackNavigation` rows beside its
tables. No re-commit is possible or needed; the disclosure is the remedy. Note for the sibling audits
(`cli`, `core`, `release`): a late re-verdict rewrites the *kind* of a commit that is already made, so
the ruling must be re-read after any re-verdict, not only after the refactor. `6add4f7` ("table-driven consolidations in snapshot
tests") also carries out the `TestSnapshot_UnderlineColor` REWRITE row — `- t.Logf("UnderlineColor
= %q", got)` becomes `+ t.Errorf("UnderlineColor = %q, want %q", ...)` — so it is TABLE + REWRITE
exactly as `046bba9` is. Both are left unsplit for the same reason, recorded once below, and both
are named here. `046bba9` consolidates the ten
`TestParseSixel_*` functions and three end-to-end sixel tests into tables (TABLE) and also rewrites
`TestSixelScrollingAtBottom`'s assertions (REWRITE). The two portions are textually disjoint (the
rewrite is a separate function body, not folded into `TestSixelEndToEnd`), so strictly it should
have been two commits, TABLE then REWRITE. It was left unsplit because splitting a mid-chain commit
under the follow-on FIX commit (which touches the same file) risks rebase churn for a cosmetic
ordering gain, and the deviation does not undermine the order rule's purpose: no `DELETE`/`MERGE`
evidence depended on sequence here (there are no DELETEs at all), and each rewrite's target still
exists under its own name with its verdict, evidence and target recorded below. The same holds for
`6add4f7`. Recorded as a documented deviation from "one verdict kind per commit", in both commits.

**SPLIT step added.** The TABLE/ADD consolidations had left eleven test functions over the hard
50-code-line function limit (`architecture.md` §3; table literals are code the compiler acts on).
`2d8fa5a` moves oversized case slices to package-level vars beside their test (variation stays
data, no harness is copied) and splits `TestColorsResolution` (138 code lines) by concern into
`TestDefaultPalette`, `TestResolveDefaultColor` and `TestResolveDefaultColorNamed`. No assertion,
case or coverage change; the only renamed tests are the colors ones, in the name map.

**Recorded-but-unexecuted verdicts executed after the SPLIT step (order deviation).** The prior
workers' verdict tables and name map described an end state that was never committed: forty-odd
TABLE rows, one MERGE (`TestMiddlewareMergeSetUserVar`) and one REWRITE (`TestImageManager_Prune`
— its "rewrite" in `eeed24e` had left a can't-fail body) had no matching code change. The
finishing session executed them all: `cc969b8`, `fa669f3`, `7d36b3b`, `0eb7472`, `79f2452`,
`d6b1161` (TABLE), `6deec26` (MERGE), `04326c2` (REWRITE), `34b34eb` (TABLE + SPLIT — the
consolidations pushed `terminal_test.go` to 1082 code lines, over the 1000 hard limit, so the
resize concern moved to `resize_test.go`; mixed kinds in one commit because the split is a direct
consequence of the same edit), `4c004f3` (SPLIT: `checkImageData` helper, function limit). These
land after the recorded SPLIT commit, out of the fixed ADD→…→SPLIT→DELETE order. The deviation is
safe for the same reason as `046bba9`'s: there are no DELETEs, and the one MERGE's evidence does
not depend on commit sequence. `6deec26` restores the case `7bc970d` dropped, and the
revision-1 claim about *why* was wrong. It said `7bc970d` had "uncovered `Middleware.Merge`'s
`DesktopNotification` field-copy branch". It had not: `middleware.go:433.38,435.3` was never
uncovered, because `TestDesktopNotificationMiddleware` reaches it through `WithMiddleware` →
`terminal.go:287 t.middleware.Merge(mw)` at `a2b682e`, `7bc970d`, `2d8fa5a` and `6deec26` alike.
The 02-audit review disproved the claim with a mutation, and mutation M1 below reproduces it at
this head: removing the restored case loses **zero** middleware blocks. What `7bc970d` actually
dropped was the *assertion* that a merged handler is invoked and forwards to the provider — a loss
no coverage gate can see, which is precisely why the guideline requires a break-the-code mutation
for every MERGE rather than a coverage comparison. `6deec26` restored that assertion; the case now
exists as a subtest of the `TestMiddlewareMerge` host.

**Verify phase caught two test-code defects (fixed in `8c77e78`).** (1) `desktopNotificationCases`
embedded `*testNotificationProvider` instances in the package-level table, so `notifyCount`
accumulated across `-count=3` repetitions (`expected 1 notifications, got 2`, then 3) — a
consolidation defect from `7bc970d`, invisible to single runs, caught by the shuffle and repeat
runs; providers are now constructed inside each subtest. (2) The prune rewrite's third case
asserted an unreachable state: `Store` prunes immediately after adding, before any `Place` can
reference the new image, so "all-referenced over budget" cannot arise through the API; the case
now documents the real contract — an over-budget image is evicted by its own `Store` call. No
production bug found; no audit row stopped on a defect.

**Six commits in the range do not compile — documented deviation, not repaired.** `GOOS=linux go
vet ./...` over an archived tree of every commit in `a2b682e..c412993` fails on exactly six
consecutive commits — `eeed24e`, `f8550ba`, `6add4f7`, `046bba9`, `7bc970d`, `1789aea` — each with
the same error:

    vet: ./image_test.go:44:25: invalid operation: m.UsedMemory() != c.wantMemory (mismatched types int64 and int)

`eeed24e` introduced it (an `int` literal field against an `int64` accessor in the image table) and
`8faf70b` repaired it; every other commit in the range vets clean. **The MERGE commit `7bc970d`
sits inside that window**, so the guideline's "every `DELETE` and `MERGE` mutation is re-run
immediately before its commit, against the suite as refactored so far" was not merely skipped
there — it was not performable, because the suite does not build at that commit. `git bisect` over
the window is broken for the same reason.

The window is **not repaired here**. Repairing it means folding `8faf70b`'s type fixes back into
`eeed24e` (or splitting them across the commits that introduced them), which rewrites six pushed
commits and requires a force-push of the project branch. History is append-only for this session,
so the deviation is documented instead and the decision to rewrite is left to the operator.

What is done instead: **all three MERGE mutations are taken at this head, in both forms** (six
runs), and recorded under the current test names in the Gates block below. What that proves and what it does not:

- it **proves** that at the head of this range each merge preserves the behaviour its row claims —
  the folded case is the unique catcher of the mutation, or (for `TestMiddlewareMergeDesktopNotification`)
  it is not, and the row's evidence says so in full;
- it does **not** prove the merge preserved behaviour *at `7bc970d`*, where the evidence was owed.
  Nine commits land between `7bc970d` and this head, one of them (`8c77e78`) a fix to the very
  table `7bc970d` created. Head evidence is the strongest evidence obtainable without rewriting
  pushed history; it is not the evidence the guideline asks for, and it is not presented as such.

**Coverage artifact layout deviates from the spec (recorded, not changed).** The feature spec asked
for `reviews/2026-09-14-clawee-go-headless-term-coverage/<phase>-covered.txt`. The layout on disk is
`<phase>/covered.txt` directories — the runner's `--artifacts <dir>` writes `test.json`, `cover.out`
and `covered.txt` into one directory per run, so a per-phase directory is what the runner produces —
plus a top-level `baseline-covered.txt` (and `baseline-covered-pre-excision.txt`) which are the
baseline sets the coverage gate is quoted against. Harmless; recorded here because it was an
undocumented spec deviation.

**Revision 2: what was folded and what was re-verdicted (the `TABLE`-row finding).** The 02-audit
review found that around forty-four rows recorded `TABLE` had been executed by moving a test body
into a named `func <lowercase>Case(t *testing.T)` and registering it with `t.Run` — a rename and a
regroup, not "variation is data" (guideline §2.2) — and that the suite had grown rather than shrunk.
Every one of those rows was re-examined, one at a time, and resolved one of two ways:

- **folded** into a real data table where the cases genuinely differ only in input and expectation
  (commit `f8679b4`) — **16 rows**: the eight alternate-screen resize scenarios (R01/PR1) plus the
  three parent funcs they collapse, three of the five resize-bounds cases (T31), the ANSI recording
  capture case (T30) and the empty-terminal snapshot case (SN01).
- **re-verdicted `RENAME`** where the cases differ in *behaviour* rather than data: a different API,
  a different assertion, a different contract. They stay separate named subtests under their host,
  and their rows now carry that verdict with the reason — **20 rows**: four kitty end-to-end cases,
  three recording cases, five resize-scrollback cases, the grow-cols bounds case, both row-conversion
  cases, the wide-character cursor case, two semantic-prompt scrollback cases, the alternate-screen
  event-pairing case, and the snapshot cursor case. Forcing these into a table is the mistake the
  same review found in `bufferGrowCases` — three assertions collapsed into one boolean — and it is
  not repeated to improve a number.

Two clusters were compacted *inside* their subtests without changing the row's verdict: the
`AbsoluteRowToViewport` and `ViewportRowToAbsolute` assertion runs became in/want tables within the
subtest that owns them (nine hand-written if/Errorf blocks became two tables). `f8679b4`'s commit
message lists those as folded rows ("T34 x2") and over-counts the recording and snapshot folds
("T30 x2", "SN01 x2"); the counts above, taken from this table, are the record.

Fixture sprawl found on the way is extracted rather than tabled (`1c024cb`, rows X01-X03), and the
four tables carrying `func`-typed fields are dealt with in `b2dd063` (rows below). The resulting
test-code line delta is stated plainly in the `After` section; it is still above baseline and the
reasons are given there rather than smoothed over.

**Revision 2: commit messages cite report rows.** The guideline requires each commit message to cite
the report rows it carries out. Of the revision-1 commits only `d6b1161` does; the rest cite test
names. Those messages cannot be corrected without rewriting pushed history (see the compile-window
ruling), so they stand. All three revision-2 test-code commits cite row ids.

## Baseline

| package | files | test code lines | tests | cases | skips | covered blocks | runtime | failing |
|---|---|---|---|---|---|---|---|---|
| `headlessterm` | 13 | 4 045 | 220 | 23 | 0 | 952 | 0.173 s | 0 |
| `internal/generate_width_table` | 1 | 97 | 6 | 0 | 0 | included in module set | 0.002 s | 0 |
| module | 14 | 4 142 | 226 | 23 | 0 | 952 | 0.175 s | 0 |

Shuffled run: seed `1789505332128309039` (root), `1789505332135199183` (internal) · result PASS · covered 952 blocks (known non-deterministic loss of 1 block around `image.go:342`) · repeated run: count 3 · result PASS · covered 952 blocks.

Covered set: `reviews/2026-09-14-clawee-go-headless-term-coverage/baseline-covered.txt` —
re-taken at `a2b682e` after the excision ruling (952 blocks; the pre-excision 953-block set is
kept as `baseline-covered-pre-excision.txt` for provenance — its extra entries are the shifted
`providers.go` block keys plus the known non-deterministic `image.go:342` block).

## `internal/generate_width_table`

### Inventory

| Row | Kind | Code | Primary test | Also covered by |
|---|---|---|---|---|
| G01 | function | `parseProperty` parses Unicode property files into wide spans | `TestParseProperty` | — |
| G02 | function | `parseProperty` rejects malformed lines | `TestParsePropertyRejectsMalformedLines` | — |
| G03 | function | `checkVersion` validates UCD/emoji version header | `TestCheckVersion` | — |
| G04 | function | `mergeSpans` sorts and merges overlapping spans | `TestMergeSpans` | — |
| G05 | function | `subtractSpans` removes ranges from spans | `TestSubtractSpans` | — |
| G06 | function | `render` emits valid Go source for a width table | `TestRenderCompiles` | — |
| G07 | I/O | `loadTables`, `fetch`, `run`, `writeRangeTable` (file/network I/O, exercised at generation time) | — | — |

### Verdicts

| Test | Rows covered | Also primary for | Verdict | Evidence | Target |
|---|---|---|---|---|---|
| `TestParseProperty` | G01 | — | KEEP | Direct parser contract | — |
| `TestParsePropertyRejectsMalformedLines` | G02 | — | KEEP | Direct malformed-line rejection | — |
| `TestCheckVersion` | G03 | — | KEEP | Direct version validation | — |
| `TestMergeSpans` | G04 | — | KEEP | Direct merge contract | — |
| `TestSubtractSpans` | G05 | — | KEEP | Direct subtraction contract | — |
| `TestRenderCompiles` | G06 | — | KEEP | Direct render contract | — |

## `wasm/` inventory note

The `wasm/` directory is a separate `js/wasm` module (`wasm/main.go`, `wasm/handlers.go`, `wasm/README.md`, `wasm/example/`). It has no test files today and is not part of `ci/run-tests.sh`. It is inventoried here for completeness; no verdicts are assigned.

## `headlessterm`

### Inventory

| Row | Kind | Code | Primary test | Also covered by |
|---|---|---|---|---|
| T01 | method | `Terminal.New` / default size | `TestNewTerminal` | — |
| T02 | option | `WithSize` | `TestTerminalWithSize` | — |
| T03 | method | `Terminal.Write` / `WriteString` | `TestTerminalWrite` | — |
| T04 | method | Cursor position after writes | `TestTerminalCursorPosition` | `TestTerminalNarrowPictographCursorWrite`, `TestTerminalWideCharacterCursorWrite`, `TestTerminalWideCharacter` |
| T05 | behavior | Newline / CR+LF handling | `TestTerminalNewline` | — |
| T06 | handler | `ClearScreen` (CSI 2J) | `TestTerminalClearScreen` | `TestMiddlewareClearScreen` |
| T07 | integration | Scrollback pushes | `TestTerminalScrollback` | `TestCustomScrollbackProvider` |
| T08 | method | Selection API | `TestTerminalSelection` | — |
| T09 | method | `Search` | `TestTerminalSearch` | — |
| T10 | method | `String` output | `TestTerminalString` | — |
| T11 | method | Terminal dirty tracking | `TestTerminalDirtyTracking` | — |
| T12 | behavior | Wide-character / pictograph cursor handling | `TestTerminalWideCharacter` | `TestTerminalNarrowPictographCursorWrite`, `TestTerminalWideCharacterCursorWrite` |
| T13 | method | `Terminal.Resize` | `TestTerminalResize` | many resize tests |
| T14 | handler | Title handling (OSC 0/1/2) | `TestTerminalTitle` | `TestMiddlewareSetTitle` |
| T15 | handler | SGR color attribute | `TestTerminalColors` | — |
| T16 | handler | SGR bold attribute | `TestTerminalBold` | — |
| T17 | behavior | Alternate screen switching | `TestTerminalAlternateScreen` | — |
| T18 | integration | Custom scrollback provider wiring | `TestTerminalScrollback` | `TestCustomScrollbackProvider` |
| T19 | middleware | `Middleware.Input` | `TestMiddlewareInput` | `TestMiddlewareSkipsCall` |
| T20 | middleware | `Middleware.Bell` | `TestMiddlewareBell` | — |
| T21 | middleware | `Middleware.SetTitle` | `TestMiddlewareSetTitle` | — |
| T22 | middleware | `Middleware.ClearScreen` | `TestMiddlewareClearScreen` | — |
| T23 | provider | Clipboard provider wiring | `TestClipboardProvider` | — |
| T24 | method | `PTYWriter` / response output | `TestResponseWriter` | `TestWriteResponseRaceCondition` |
| T25 | struct | `Middleware.Merge` | `TestMiddlewareMerge` | `TestMiddlewareMergeDesktopNotification`, `TestMiddlewareMergeSetUserVar` |
| T26 | behavior | Wrapped-line tracking | `TestTerminalWrappedLineTracking` | `TestTerminalWrappedLineClearedOnNewline` |
| T27 | option | `WithAutoResize` vertical growth | `TestTerminalAutoResizeY` | — |
| T28 | option | `WithAutoResize` horizontal growth | `TestTerminalAutoResizeX` | — |
| T29 | option | `WithAutoResize` no scrollback | `TestTerminalAutoResizeNoScrollback` | — |
| T30 | provider | Recording capture / clear / replay | `TestTerminalRecording` | `TestTerminalRecordingWithANSI`, `TestTerminalRecordingClear`, `TestTerminalRecordingReplay`, `TestTerminalRecordingSetProvider` |
| T31 | behavior | Resize invalid dimensions / cursor bounds | `TestResizeInvalidDimensions` | `TestResizeCursorBounds`, `TestCursorBoundsAfterGrowCols`, `TestCursorBoundsAfterWrap`, `TestInputWithInvalidCursorPosition` |
| T32 | behavior | Resize shrink/grow with scrollback | `TestResizeGrowPullsFromScrollback` | `TestResizeShrinkWithCursorInBounds`, `TestResizeShrinkWithCursorOutOfBounds`, `TestResizeShrinkScrollbackContent`, `TestResizeGrowNoScrollbackUnchanged`, `TestResizeCursorPositionAfterShrink` |
| T33 | behavior | Resize on alternate screen | `TestResizeAlternateScreenNoScrollback` | resize_alternate_screen tests |
| T34 | method | Viewport / absolute row conversion | `TestViewportRowToAbsolute` | `TestAbsoluteRowToViewport`, `TestRowConversionRoundTrip` |
| T35 | method | `SetActiveCharset` bounds | `TestActiveCharsetBoundsValidation` | — |
| T36 | concurrency | Concurrent response writes | `TestWriteResponseRaceCondition` | — |
| T37 | handler | `DeviceStatus` report | `TestResponseWriter` | — |
| B01 | method | `NewBuffer` dimensions | `TestNewBuffer` | — |
| B02 | method | `Buffer.Cell` access / OOB | `TestBufferCell` | `TestBufferCellOutOfBounds` |
| B03 | method | `Buffer.ClearRow` | `TestBufferClearRow` | — |
| B04 | method | `Buffer.ScrollUp` / `ScrollDown` | `TestBufferScrollUp` | `TestBufferScrollDown` |
| B05 | integration | Buffer scrollback | `TestBufferScrollback` | — |
| B06 | method | `Buffer.LineContent` | `TestBufferLineContent` | — |
| B07 | method | Buffer tab stops | `TestBufferTabStops` | — |
| B08 | method | `Buffer.Resize` | `TestBufferResize` | — |
| B09 | method | Buffer dirty tracking | `TestBufferDirtyTracking` | — |
| B10 | method | `Buffer.InsertBlanks` / `DeleteChars` | `TestBufferInsertBlanks` | `TestBufferDeleteChars` |
| B11 | method | Buffer wrapped-line tracking | `TestBufferWrappedLineTracking` | `TestBufferWrappedLineTrackingWithScroll` |
| B12 | method | `Buffer.GrowRows` / `GrowCols` | `TestBufferGrowRows` | `TestBufferGrowCols` |
| C01 | method | `NewCell` defaults | `TestNewCell` | — |
| C02 | method | `Cell.Reset` | `TestCellReset` | — |
| C03 | method | Cell flag helpers | `TestCellFlags` | — |
| C04 | method | Cell dirty helpers | `TestCellDirty` | — |
| C05 | method | Cell wide/spacer helpers | `TestCellWide` | — |
| C06 | method | `Cell.Copy` | `TestCellCopy` | — |
| I01 | method | `ImageManager.Store` / deduplication | `TestImageManager_Store` | `TestImageManager_Deduplication` |
| I02 | method | `ImageManager.StoreWithID` | `TestImageManager_StoreWithID` | — |
| I03 | method | `ImageManager.Place` | `TestImageManager_Place` | — |
| I04 | method | `ImageManager` delete / clear / prune | `TestImageManager_DeleteImage` | `TestImageManager_Clear`, `TestImageManager_Prune` |
| I05 | method | `ImageManager.Placements` | `TestImageManager_Placements` | — |
| I06 | method | Placement deletion by position/row/range/above/below | `TestImageManager_DeletePlacementsByPosition` | `TestImageManager_DeletePlacementsInRow`, `InRowRange`, `Below`, `Above` |
| I07 | method | `Cell.HasImage` / image reset | `TestCellImage` | — |
| I08 | integration | Terminal image clearing on CSI clear/reset/alternate screen | `TestClearScreenClearsImages` | `TestClearScreenBelowClearsImages`, `TestResetStateClearsImagesAndCache`, `TestAlternateScreenClearsImages` |
| K01 | function | `ParseKittyGraphics` field parsing | `TestParseKittyGraphics_Basic` | `TestParseKittyGraphics_Query`, `Delete`, `Chunked`, `WithZIndex`, `Placement`, `DoNotMoveCursor` |
| K02 | method | `KittyCommand.DecodeImageData` | `TestKittyCommand_DecodeRGBA` | `TestKittyCommand_DecodeRGB` |
| K03 | function | `FormatKittyResponse` | `TestFormatKittyResponse` | — |
| K04 | integration | Terminal Kitty display / cell assignment / UV / chunked / delete | `TestKittyImageDisplay` | `TestKittyImageCellAssignment`, `UVCoordinates`, `ChunkedTransfer`, `ImageDelete` |
| S01 | function | `ParseSixel` (dimensions, repeat, colors, transparency) | `TestParseSixel_SimplePixel` | `TestParseSixel_MultipleColumns`, `NewLine`, `CarriageReturn`, `Repeat`, `ColorRGB`, `ColorHLS`, `Transparent`, `Empty`, `ComplexImage` |
| S02 | integration | Terminal Sixel display / cell assignment / cursor / scrolling | `TestSixelImageDisplay` | `TestSixelImageCellAssignment`, `CursorMovement`, `ScrollingAtBottom` |
| N01 | provider | Notification provider wiring / defaults | `TestDefaultNotificationProvider` | `TestNoopNotification`, `TestWithNotificationOption`, `TestSetNotificationProvider` |
| N02 | handler | `DesktopNotification` payloads / nil / query | `TestDesktopNotificationHandler` | `TestDesktopNotificationWithNilProvider`, `QueryResponse`, `PayloadFields`, `EmptyPayload` |
| N03 | middleware | `DesktopNotification` middleware | `TestDesktopNotificationMiddleware` | `TestDesktopNotificationMiddlewareBlocks` |
| R01 | integration | Resize on alternate screen preserves primary | `TestResizeOnAlternateScreenKeepsPrimary` | `TestResizeGrowBeforeLeavingAlternateScreenKeepsPrimary`, `WithPrimaryCursorAboveShrink`, `PairsLikePrimaryResize`, `ThenPrimaryOutput`, `HeightIndependentScroll` |
| R02 | integration | Resize column clamp on alternate screen | `TestResizeColumnsOnAlternateScreenClampsSavedCursor` | — |
| P01 | handler | Semantic prompt mark types | `TestSemanticPromptMark_PromptStart` | `TestSemanticPromptMark_CommandStart`, `CommandExecuted`, `CommandFinished`, `CommandFinishedWithExitCode` subtests |
| P02 | method | Prompt row navigation | `TestSemanticPromptMark_NextPromptRow` | `TestSemanticPromptMark_PrevPromptRow`, `FilterByType`, `ClearMarks`, `GetMarkAt` |
| P03 | middleware | Semantic prompt handler / middleware / ST terminator | `TestSemanticPromptMark_Handler` | `TestSemanticPromptMark_ST_Terminator`, `Middleware` |
| P04 | method | `GetLastCommandOutput` | `TestGetLastCommandOutput_Basic` | `MultiLine`, `NoOutput`, `NoMarks`, `OnlyExecutedNoFinished`, `MultipleCommands`, `WithExitCode`, `TrailingEmptyLines` |
| P05 | integration | Prompt marks with scrollback | `TestSemanticPromptMark_NextPromptRowWithScrollback` | `PrevPromptRowWithScrollback`, `GetMarkAtWithScrollback` |
| SN01 | method | Snapshot text / cursor / empty | `TestSnapshot_Text` | `TestSnapshot_Cursor`, `EmptyTerminal` |
| SN02 | method | Snapshot styled segments | `TestSnapshot_Styled` | `TestSnapshot_StyledSegments` |
| SN03 | method | Snapshot full cells / attributes | `TestSnapshot_Full` | `TestSnapshot_Attributes` |
| SN04 | method | Snapshot underline / blink styles | `TestSnapshot_UnderlineStyles` | `TestSnapshot_BlinkStyles`, `TestSnapshot_UnderlineColor` |
| SN05 | method | Snapshot hyperlink / wide char | `TestSnapshot_Hyperlink` | `TestSnapshot_WideChar` |
| SN06 | method | Snapshot images | `TestSnapshot_Images` | `TestSnapshot_NoImages` |
| SN07 | function | `colorToHex` / `cursorStyleToString` | `TestColorToHex` | `TestCursorStyleToString` |
| SN08 | method | `GetImageData` | `TestGetImageData` | `TestGetImageData_NotFound` |
| U01 | method | User variable set/get/clear/copy | `TestSetUserVar` | `TestGetUserVarNotSet`, `TestGetUserVars`, `TestGetUserVarsReturnsACopy`, `TestClearUserVars`, `TestUserVarOverwrite`, `TestUserVarEmptyValue` |
| U02 | handler | OSC 1337 SetUserVar parsing | `TestOSC1337SetUserVar` | `WithST`, `InvalidBase64`, `EmptyValue`, `SpecialCharacters`, `TestUserVarsWithPTYWriter` |
| U03 | middleware | `SetUserVar` middleware | `TestUserVarMiddleware` | `TestUserVarMiddlewareBlocks`, `TestMiddlewareMergeSetUserVar` |
| W1 | function | `runeWidth` / `isWideRune` / `StringWidth` | `TestRuneWidth` | `TestIsWideRune`, `TestStringWidth` |
| W2 | constant | Width-table Unicode version | `TestWidthTableMatchesGenerateDirective` | — |
| W3 | function | Width-function agreement over all runes | `TestWidthFunctionsAgree` | — |
| WD1 | handler | Working directory parse / path / middleware | `TestWorkingDirectory_Basic` | `STTerminator`, `Multiple`, `NotSet`, `Path_Basic`, `Path_WithHostname`, `Path_EmptyHostname`, `Path_NotSet`, `TestWorkingDirectory_Middleware` |
| PR1 | function | Primary screen resize helpers (`primary_resize.go`) | `TestResizeOnAlternateScreenKeepsPrimary` | `TestResizeGrowBeforeLeavingAlternateScreenKeepsPrimary`, `TestResizeShrinkWithCursorOutOfBounds`, `TestResizeGrowPullsFromScrollback`, `TestResizeAlternateScreenNoScrollback` |
| CL1 | function | `colors.go` resolution (`DefaultPalette`, `ResolveDefaultColor`, `resolveNamedColor`) | — | — |

### Verdicts

| Test | Rows covered | Also primary for | Verdict | Evidence (final) | Target |
|---|---|---|---|---|---|
| `TestNewBuffer` | B01 | — | KEEP | Direct constructor contract | — |
| `TestBufferCell` | B02 | `TestBufferCellOutOfBounds` | KEEP | Direct cell access contract | — |
| `TestBufferCellOutOfBounds` | B02 | — | TABLE | Same row as `TestBufferCell`; only input/expected varies | `TestBufferCell` |
| `TestBufferClearRow` | B03 | — | KEEP | Direct clear contract | — |
| `TestBufferScrollUp` | B04 | `TestBufferScrollDown` | KEEP | Direct scroll contract | — |
| `TestBufferScrollDown` | B04 | — | TABLE | Near-duplicate of `ScrollUp` with direction as data | `TestBufferScrollUp` |
| `TestBufferScrollback` | B05 | — | KEEP | Direct scrollback integration | — |
| `TestBufferLineContent` | B06 | — | KEEP | Direct line-content contract | — |
| `TestBufferTabStops` | B07 | — | KEEP | Direct tab-stop contract | — |
| `TestBufferResize` | B08 | — | KEEP | Direct resize contract | — |
| `TestBufferDirtyTracking` | B09 | — | KEEP | Direct dirty tracking contract | — |
| `TestBufferInsertBlanks` | B10 | `TestBufferDeleteChars` | KEEP | Direct line-edit contract | — |
| `TestBufferDeleteChars` | B10 | — | TABLE | Near-duplicate of `InsertBlanks`; operation as data | `TestBufferInsertBlanks` |
| `TestBufferWrappedLineTracking` | B11 | `TestBufferWrappedLineTrackingWithScroll` | KEEP | Direct wrapped-flag contract | — |
| `TestBufferWrappedLineTrackingWithScroll` | B11 | — | TABLE | Same row with scroll as extra dimension | `TestBufferWrappedLineTracking` |
| `TestBufferGrowRows` | B12 | `TestBufferGrowCols` | KEEP | Direct grow contract | — |
| `TestBufferGrowCols` | B12 | — | TABLE | Near-duplicate of `GrowRows`; axis as data | `TestBufferGrowRows` |
| `TestNewCell` | C01 | — | KEEP | Direct constructor contract | — |
| `TestCellReset` | C02 | — | KEEP | Direct reset contract | — |
| `TestCellFlags` | C03 | — | KEEP | Direct flag helper contract | — |
| `TestCellDirty` | C04 | — | KEEP | Direct dirty helper contract | — |
| `TestCellWide` | C05 | — | KEEP | Direct wide/spacer contract | — |
| `TestCellCopy` | C06 | — | KEEP | Direct copy/independence contract | — |
| `TestImageManager_Store` | I01 | `TestImageManager_Deduplication` | KEEP | Direct store contract | — |
| `TestImageManager_Deduplication` | I01 | — | TABLE | Same row; deduplication is a table case of `Store` | `TestImageManager_Store` |
| `TestImageManager_StoreWithID` | I02 | — | KEEP | Direct explicit-ID contract | — |
| `TestImageManager_Place` | I03 | — | KEEP | Direct placement contract | — |
| `TestImageManager_DeleteImage` | I04 | `TestImageManager_Clear`, `TestImageManager_Prune` | KEEP | Direct deletion contract | — |
| `TestImageManager_Clear` | I04 | — | TABLE | Same lifecycle row as `DeleteImage` | `TestImageManager_DeleteImage` |
| `TestImageManager_Prune` | I04 | — | REWRITE | Sets max memory but never asserts eviction; does not test the contract | `TestImageManager_Lifecycle` |
| `TestImageManager_Placements` | I05 | — | KEEP | Direct query contract | — |
| `TestImageManager_DeletePlacementsByPosition` | I06 | `DeletePlacementsInRow`, `InRowRange`, `Below`, `Above` | KEEP | Direct placement-deletion contract | — |
| `TestImageManager_DeletePlacementsInRow` | I06 | — | TABLE | Same deletion row; dimension as data | `TestImageManager_DeletePlacementsByPosition` |
| `TestImageManager_DeletePlacementsInRowRange` | I06 | — | TABLE | Same deletion row; dimension as data | `TestImageManager_DeletePlacementsByPosition` |
| `TestImageManager_DeletePlacementsBelow` | I06 | — | TABLE | Same deletion row; dimension as data | `TestImageManager_DeletePlacementsByPosition` |
| `TestImageManager_DeletePlacementsAbove` | I06 | — | TABLE | Same deletion row; dimension as data | `TestImageManager_DeletePlacementsByPosition` |
| `TestCellImage` | I07 | — | KEEP | Direct cell-image contract | — |
| `TestClearScreenClearsImages` | I08 | `BelowClearsImages`, `ResetStateClearsImagesAndCache`, `AlternateScreenClearsImages` | KEEP | Direct terminal-driven image clearing | — |
| `TestClearScreenBelowClearsImages` | I08 | — | TABLE | Same row; clear mode as data | `TestClearScreenClearsImages` |
| `TestResetStateClearsImagesAndCache` | I08 | — | TABLE | Same row; reset path as data | `TestClearScreenClearsImages` |
| `TestAlternateScreenClearsImages` | I08 | — | TABLE | Same row; alternate-screen path as data | `TestClearScreenClearsImages` |
| `TestParseKittyGraphics_Basic` | K01 | `Query`, `Delete`, `Chunked`, `WithZIndex`, `Placement`, `DoNotMoveCursor` | KEEP | Direct parser contract | — |
| `TestParseKittyGraphics_Query` | K01 | — | TABLE | Same parser row; action/field as data | `TestParseKittyGraphics_Basic` |
| `TestParseKittyGraphics_Delete` | K01 | — | TABLE | Same parser row; action/field as data | `TestParseKittyGraphics_Basic` |
| `TestParseKittyGraphics_Chunked` | K01 | — | TABLE | Same parser row; more flag as data | `TestParseKittyGraphics_Basic` |
| `TestParseKittyGraphics_WithZIndex` | K01 | — | TABLE | Same parser row; z-index as data | `TestParseKittyGraphics_Basic` |
| `TestParseKittyGraphics_Placement` | K01 | — | TABLE | Same parser row; placement fields as data | `TestParseKittyGraphics_Basic` |
| `TestParseKittyGraphics_DoNotMoveCursor` | K01 | — | TABLE | Same parser row; cursor flag as data | `TestParseKittyGraphics_Basic` |
| `TestKittyCommand_DecodeRGBA` | K02 | `TestKittyCommand_DecodeRGB` | KEEP | Direct decode contract | — |
| `TestKittyCommand_DecodeRGB` | K02 | — | TABLE | Same decode row; format as data | `TestKittyCommand_DecodeRGBA` |
| `TestFormatKittyResponse` | K03 | — | KEEP | Direct response formatting contract | — |
| `TestKittyImageDisplay` | K04 | `CellAssignment`, `UVCoordinates`, `ChunkedTransfer`, `ImageDelete` | KEEP | End-to-end Kitty transmit/display | — |
| `TestKittyImageCellAssignment` | K04 | — | RENAME | Named subtest of `TestKittyImageEndToEnd`; asserts placeholder chars and image refs per cell, not a data variation of the display case | `TestKittyImageEndToEnd/cell assignment` |
| `TestKittyImageUVCoordinates` | K04 | — | RENAME | Named subtest; asserts UV rectangles (its own table inside), a different contract from the display case | `TestKittyImageEndToEnd/uv coordinates` |
| `TestKittyChunkedTransfer` | K04 | — | RENAME | Named subtest; asserts the staging of an incomplete transfer, a behaviour no other case has | `TestKittyImageEndToEnd/chunked transfer` |
| `TestKittyImageDelete` | K04 | — | RENAME | Named subtest; asserts the d=a / d=I delete split, a different contract | `TestKittyImageEndToEnd/image delete` |
| `TestNoopNotification` | N01 | — | MERGE | **M2 at head (both forms).** (b) dropping the folded `default is NoopNotification` subtest loses `providers.go:316.69,316.82` (`NoopNotification.Notify`) — the folded case is its unique home. (a) `providers.go:316` `return ""` → `return "mutated"` reddens exactly `TestNotificationProviderWiring/default_is_NoopNotification`, and nothing else. Gates block, M2 | `TestNotificationProviderWiring/default is NoopNotification` |
| `TestWithNotificationOption` | N01 | — | TABLE | Same wiring row; option path as data | `TestDefaultNotificationProvider` |
| `TestDefaultNotificationProvider` | N01 | `TestNoopNotification`, `TestWithNotificationOption`, `TestSetNotificationProvider` | KEEP | Direct default-provider contract | — |
| `TestSetNotificationProvider` | N01 | — | TABLE | Same wiring row; runtime set as data | `TestDefaultNotificationProvider` |
| `TestDesktopNotificationHandler` | N02 | `WithNilProvider`, `PayloadFields`, `EmptyPayload` | KEEP | Direct handler contract | — |
| `TestDesktopNotificationWithNilProvider` | N02 | — | TABLE | Same handler row; nil payload as data | `TestDesktopNotificationHandler` |
| `TestDesktopNotificationQueryResponse` | N02 | — | KEEP | Distinct query-response path | — |
| `TestDesktopNotificationMiddleware` | N03 | `TestDesktopNotificationMiddlewareBlocks` | KEEP | Direct middleware contract | — |
| `TestDesktopNotificationMiddlewareBlocks` | N03 | — | TABLE | Same middleware row; blocking as data | `TestDesktopNotificationMiddleware` |
| `TestNotificationPayloadFields` | N02 | — | TABLE | Same handler row; full field list as data | `TestDesktopNotificationHandler` |
| `TestMiddlewareMergeDesktopNotification` | T25 | — | MERGE | **M1 at head (both forms).** (b) dropping `middlewareMergeDesktopNotificationCase` loses **no** middleware block (only the known `image.go:342` flake) — the case is not a unique coverage home. (a) neutering `middleware.go:434` `m.DesktopNotification = other.DesktopNotification` reddens `TestMiddlewareMerge/desktop_notification` **and** `TestDesktopNotificationMiddleware/{intercepts_and_modifies,blocks}`: the merged case catches the break, but it is not the only catcher. Gates block, M1 | `TestMiddlewareMerge/desktop notification` |
| `TestNotificationProviderThreadSafety` | N03 | — | KEEP | Concurrency smoke test | — |
| `TestNotificationEmptyPayload` | N02 | — | TABLE | Same handler row; empty payload as data | `TestDesktopNotificationHandler` |
| `TestResizeOnAlternateScreenKeepsPrimary` | R01, PR1 | — | KEEP | Complex integration contract for alternate-screen resize | — |
| `TestResizeGrowBeforeLeavingAlternateScreenKeepsPrimary` | R01, PR1 | — | TABLE | Same integration row; ordering as data | `TestResizeOnAlternateScreenKeepsPrimary` |
| `TestResizeOnAlternateScreenWithPrimaryCursorAboveShrink/growWhileAlternate=false` | R01, PR1 | — | TABLE | Same integration row; cursor-above-shrink as data | `TestResizeOnAlternateScreenKeepsPrimary` |
| `TestResizeOnAlternateScreenWithPrimaryCursorAboveShrink/growWhileAlternate=true` | R01, PR1 | — | TABLE | Same integration row; cursor-above-shrink as data | `TestResizeOnAlternateScreenKeepsPrimary` |
| `TestResizeOnAlternateScreenPairsLikePrimaryResize` | R01, PR1 | — | RENAME | Named subtest; compares the Push/Pop sequences of TWO terminals, which no data field can express | `TestResizeOnAlternateScreenKeepsPrimary/pairs like primary resize` |
| `TestResizeOnAlternateScreenThenPrimaryOutputKeepsPrimary/lines=19` | R01, PR1 | — | TABLE | Same integration row; output volume as data | `TestResizeOnAlternateScreenKeepsPrimary` |
| `TestResizeOnAlternateScreenThenPrimaryOutputKeepsPrimary/lines=20` | R01, PR1 | — | TABLE | Same integration row; output volume as data | `TestResizeOnAlternateScreenKeepsPrimary` |
| `TestResizeOnAlternateScreenThenPrimaryOutputKeepsPrimary/lines=30` | R01, PR1 | — | TABLE | Same integration row; output volume as data | `TestResizeOnAlternateScreenKeepsPrimary` |
| `TestResizeColumnsOnAlternateScreenClampsSavedCursor` | R02 | — | KEEP | Distinct column-clamp contract | — |
| `TestResizeOnAlternateScreenThenHeightIndependentScrollKeepsPrimary/SU 3` | R01, PR1 | — | TABLE | Same integration row; scroll type as data | `TestResizeOnAlternateScreenKeepsPrimary` |
| `TestResizeOnAlternateScreenThenHeightIndependentScrollKeepsPrimary/region 1;10 scroll` | R01, PR1 | — | TABLE | Same integration row; scroll type as data | `TestResizeOnAlternateScreenKeepsPrimary` |
| `TestSemanticPromptMark_PromptStart` | P01 | — | KEEP | Direct mark-type contract | — |
| `TestSemanticPromptMark_CommandStart` | P01 | — | TABLE | Same mark-type row; type as data | `TestSemanticPromptMark_PromptStart` |
| `TestSemanticPromptMark_CommandExecuted` | P01 | — | TABLE | Same mark-type row; type as data | `TestSemanticPromptMark_PromptStart` |
| `TestSemanticPromptMark_CommandFinished` | P01 | — | TABLE | Same mark-type row; type as data | `TestSemanticPromptMark_PromptStart` |
| `TestSemanticPromptMark_CommandFinishedWithExitCode/exit code 0` | P01 | — | TABLE | Same mark-type row; exit code as data | `TestSemanticPromptMark_PromptStart` |
| `TestSemanticPromptMark_CommandFinishedWithExitCode/exit code 1` | P01 | — | TABLE | Same mark-type row; exit code as data | `TestSemanticPromptMark_PromptStart` |
| `TestSemanticPromptMark_CommandFinishedWithExitCode/exit code 127` | P01 | — | TABLE | Same mark-type row; exit code as data | `TestSemanticPromptMark_PromptStart` |
| `TestSemanticPromptMark_FullSequence` | P01 | — | KEEP | Integration: full prompt cycle | — |
| `TestSemanticPromptMark_RowTracking` | P01, P02 | — | KEEP | Distinct row-tracking contract | — |
| `TestSemanticPromptMark_NextPromptRow` | P02 | `PrevPromptRow`, `FilterByType`, `ClearMarks`, `GetMarkAt` | KEEP | Direct next-prompt contract | — |
| `TestSemanticPromptMark_PrevPromptRow` | P02 | — | KEEP | Direct prev-prompt contract | — |
| `TestSemanticPromptMark_FilterByType` | P02 | — | KEEP | Distinct filter-by-type contract | — |
| `TestSemanticPromptMark_ClearMarks` | P02 | — | KEEP | Direct clear contract | — |
| `TestSemanticPromptMark_GetMarkAt` | P02 | — | KEEP | Direct get-at contract | — |
| `TestSemanticPromptMark_Handler` | P03 | `ST_Terminator`, `Middleware` | KEEP | Direct handler contract | — |
| `TestSemanticPromptMark_ST_Terminator` | P03 | — | TABLE | Same row; terminator as data | `TestSemanticPromptMark_Handler` |
| `TestSemanticPromptMark_Middleware` | P03 | — | KEEP | Distinct middleware path | — |
| `TestGetLastCommandOutput_Basic` | P04 | — | KEEP | Direct output-extraction contract | — |
| `TestGetLastCommandOutput_MultiLine` | P04 | — | TABLE | Same row; output shape as data | `TestGetLastCommandOutput_Basic` |
| `TestGetLastCommandOutput_NoOutput` | P04 | — | TABLE | Same row; empty-output case as data | `TestGetLastCommandOutput_Basic` |
| `TestGetLastCommandOutput_NoMarks` | P04 | — | TABLE | Same row; no-marks case as data | `TestGetLastCommandOutput_Basic` |
| `TestGetLastCommandOutput_OnlyExecutedNoFinished` | P04 | — | TABLE | Same row; missing-finish case as data | `TestGetLastCommandOutput_Basic` |
| `TestGetLastCommandOutput_MultipleCommands` | P04 | — | TABLE | Same row; last-command case as data | `TestGetLastCommandOutput_Basic` |
| `TestGetLastCommandOutput_WithExitCode` | P04 | — | TABLE | Same row; exit-code case as data | `TestGetLastCommandOutput_Basic` |
| `TestGetLastCommandOutput_TrailingEmptyLines` | P04 | — | TABLE | Same row; trimming case as data | `TestGetLastCommandOutput_Basic` |
| `TestSemanticPromptMark_NextPromptRowWithScrollback` | P05 | `PrevPromptRowWithScrollback`, `GetMarkAtWithScrollback` | KEEP | Direct scrollback navigation contract | — |
| `TestSemanticPromptMark_PrevPromptRowWithScrollback` | P05 | — | RENAME | Named subtest; `PrevPromptRow` over absolute rows, a different function | `TestSemanticPromptMark_ScrollbackNavigation/prev prompt row` |
| `TestSemanticPromptMark_GetMarkAtWithScrollback` | P05 | — | RENAME | Named subtest; `GetPromptMarkAt` over a scrollback row, a different function | `TestSemanticPromptMark_ScrollbackNavigation/get mark at` |
| `TestParseSixel_SimplePixel` | S01 | `MultipleColumns`, `NewLine`, `CarriageReturn`, `Repeat`, `ColorRGB`, `ColorHLS`, `Transparent`, `Empty`, `ComplexImage` | KEEP | Direct parser contract | — |
| `TestParseSixel_MultipleColumns` | S01 | — | TABLE | Same parser row; width as data | `TestParseSixel_SimplePixel` |
| `TestParseSixel_NewLine` | S01 | — | TABLE | Same parser row; newline as data | `TestParseSixel_SimplePixel` |
| `TestParseSixel_CarriageReturn` | S01 | — | TABLE | Same parser row; CR as data | `TestParseSixel_SimplePixel` |
| `TestParseSixel_Repeat` | S01 | — | TABLE | Same parser row; repeat as data | `TestParseSixel_SimplePixel` |
| `TestParseSixel_ColorRGB` | S01 | — | TABLE | Same parser row; RGB color as data | `TestParseSixel_SimplePixel` |
| `TestParseSixel_ColorHLS` | S01 | — | TABLE | Same parser row; HLS color as data | `TestParseSixel_SimplePixel` |
| `TestParseSixel_Transparent` | S01 | — | TABLE | Same parser row; transparency as data | `TestParseSixel_SimplePixel` |
| `TestParseSixel_Empty` | S01 | — | TABLE | Same parser row; empty input as data | `TestParseSixel_SimplePixel` |
| `TestParseSixel_ComplexImage` | S01 | — | TABLE | Same parser row; multi-color/row as data | `TestParseSixel_SimplePixel` |
| `TestSixelImageDisplay` | S02 | `CellAssignment`, `CursorMovement`, `ScrollingAtBottom` | KEEP | End-to-end Sixel display | — |
| `TestSixelImageCellAssignment` | S02 | — | TABLE | Same end-to-end row; cell assignment detail | `TestSixelImageDisplay` |
| `TestSixelCursorMovement` | S02 | — | TABLE | Same end-to-end row; cursor path as data | `TestSixelImageDisplay` |
| `TestSixelScrollingAtBottom` | S02 | — | REWRITE | Weak assertions; does not verify scrolled content | `TestSixelEndToEnd` |
| `TestSnapshot_Text` | SN01 | `TestSnapshot_Cursor`, `TestSnapshot_EmptyTerminal` | KEEP | Direct text snapshot contract | — |
| `TestSnapshot_Cursor` | SN01 | — | RENAME | Named subtest of `TestSnapshot_Text`; asserts the cursor block, not a line-text variation, so it is not a row of `snapshotTextCases` | `TestSnapshot_Text/cursor` |
| `TestSnapshot_Styled` | SN02 | `TestSnapshot_StyledSegments` | KEEP | Direct styled-segment contract | — |
| `TestSnapshot_Full` | SN03 | `TestSnapshot_Attributes` | KEEP | Direct full-cell contract | — |
| `TestSnapshot_Attributes` | SN03 | — | TABLE | Same full-cell row; bold attribute as data | `TestSnapshot_Full` |
| `TestSnapshot_UnderlineStyles` | SN04 | `TestSnapshot_BlinkStyles`, `TestSnapshot_UnderlineColor` | KEEP | Direct style snapshot contract | — |
| `TestSnapshot_BlinkStyles` | SN04 | — | KEEP | Direct blink style contract | — |
| `TestSnapshot_UnderlineColor` | SN04 | — | REWRITE | No assertion; only logs current behavior | `TestSnapshot_Styles` |
| `TestSnapshot_Hyperlink` | SN05 | — | KEEP | Direct hyperlink snapshot contract | — |
| `TestSnapshot_WideChar` | SN05 | — | KEEP | Direct wide-char snapshot contract | — |
| `TestColorToHex` | SN07 | `TestCursorStyleToString` | KEEP | Direct helper contract | — |
| `TestCursorStyleToString` | SN07 | — | KEEP | Direct helper contract | — |
| `TestSnapshot_EmptyTerminal` | SN01 | — | TABLE | Same text/cursor row; empty terminal as data | `TestSnapshot_Text` |
| `TestSnapshot_StyledSegments` | SN02 | — | TABLE | Same styled row; segment merging as data | `TestSnapshot_Styled` |
| `TestSnapshot_Images` | SN06 | `TestSnapshot_NoImages` | KEEP | Direct image snapshot contract | — |
| `TestSnapshot_NoImages` | SN06 | — | TABLE | Same image row; absent images as data | `TestSnapshot_Images` |
| `TestGetImageData` | SN08 | `TestGetImageData_NotFound` | KEEP | Direct image-data contract | — |
| `TestGetImageData_NotFound` | SN08 | — | KEEP | Direct missing-image contract | — |
| `TestNewTerminal` | T01 | — | KEEP | Direct default-size contract | — |
| `TestTerminalWithSize` | T02 | — | KEEP | Direct `WithSize` contract | — |
| `TestTerminalWrite` | T03 | — | KEEP | Direct write contract | — |
| `TestTerminalCursorPosition` | T04 | — | KEEP | Direct cursor-position contract | — |
| `TestTerminalNewline` | T05 | — | KEEP | Direct newline contract | — |
| `TestTerminalClearScreen` | T06 | — | KEEP | Direct clear-screen contract | — |
| `TestTerminalScrollback` | T07, T18 | — | KEEP | Direct scrollback integration | — |
| `TestTerminalSelection` | T08 | — | KEEP | Direct selection contract | — |
| `TestTerminalSearch` | T09 | — | KEEP | Direct search contract | — |
| `TestTerminalString` | T10 | — | KEEP | Direct string-output contract | — |
| `TestTerminalDirtyTracking` | T11 | — | KEEP | Direct terminal dirty contract | — |
| `TestTerminalNarrowPictographCursorWrite` | T12 | — | KEEP | Regression test for narrow pictograph spacing | — |
| `TestTerminalWideCharacterCursorWrite` | T12 | — | RENAME | Named subtest; the control for `TestTerminalNarrowPictographCursorWrite`, asserting spacer plus cursor-positioned write | `TestTerminalWideCharacter/cursor write after wide char` |
| `TestTerminalWideCharacter` | T12 | — | KEEP | Direct wide-character contract | — |
| `TestTerminalResize` | T13 | — | KEEP | Direct resize contract | — |
| `TestTerminalTitle` | T14 | — | KEEP | Direct title contract | — |
| `TestTerminalColors` | T15 | — | KEEP | Direct color attribute contract | — |
| `TestTerminalBold` | T16 | — | KEEP | Direct bold attribute contract | — |
| `TestTerminalAlternateScreen` | T17 | — | KEEP | Direct alternate-screen contract | — |
| `TestCustomScrollbackProvider` | T07, T18 | — | TABLE | Same scrollback row; custom provider as data | `TestTerminalScrollback` |
| `TestMiddlewareInput` | T19 | — | KEEP | Direct input middleware contract | — |
| `TestMiddlewareBell` | T20 | — | KEEP | Direct bell middleware contract | — |
| `TestMiddlewareSetTitle` | T21 | — | KEEP | Direct title middleware contract | — |
| `TestMiddlewareClearScreen` | T22 | — | KEEP | Direct clear-screen middleware contract | — |
| `TestClipboardProvider` | T23 | — | REWRITE | Mutation evidence shows deleting it loses the only coverage of `WithClipboard`/`ClipboardProvider`; rewrite to assert production wiring instead of testing the helper | `TestClipboardProvider` |
| `TestResponseWriter` | T24, T37 | — | KEEP | Direct response-writer contract | — |
| `TestMiddlewareSkipsCall` | T19 | — | TABLE | Same input middleware row; blocking as data | `TestMiddlewareInput` |
| `TestMiddlewareMerge` | T25 | — | KEEP | Direct `Merge` contract | — |
| `TestTerminalWrappedLineTracking` | T26 | — | KEEP | Direct wrapped-line contract | — |
| `TestTerminalWrappedLineClearedOnNewline` | T26 | — | TABLE | Same row; newline clears wrap as data | `TestTerminalWrappedLineTracking` |
| `TestTerminalAutoResizeY` | T27 | — | KEEP | Direct vertical auto-resize contract | — |
| `TestTerminalAutoResizeX` | T28 | — | KEEP | Direct horizontal auto-resize contract | — |
| `TestTerminalAutoResizeNoScrollback` | T29 | — | KEEP | Direct auto-resize scrollback contract | — |
| `TestTerminalRecording` | T30 | — | KEEP | Direct recording contract | — |
| `TestTerminalRecordingWithANSI` | T30 | — | TABLE | Same recording row; ANSI as data | `TestTerminalRecording` |
| `TestTerminalRecordingClear` | T30 | — | RENAME | Named subtest; exercises `ClearRecording`, a different API from the capture cases | `TestTerminalRecording/clear` |
| `TestTerminalRecordingReplay` | T30 | — | RENAME | Named subtest; replays into a second terminal and compares the two, not a capture assertion | `TestTerminalRecording/replay` |
| `TestTerminalRecordingSetProvider` | T30 | — | RENAME | Named subtest; exercises `SetRecordingProvider` and the Noop default | `TestTerminalRecording/set provider` |
| `TestActiveCharsetBoundsValidation` | T35 | — | KEEP | Mutation evidence shows it is the only coverage for `SetActiveCharset`/`setActiveCharsetInternal`; keep and improve later if desired | — |
| `TestResizeInvalidDimensions` | T31 | — | KEEP | Direct resize validation contract | — |
| `TestResizeCursorBounds` | T31 | — | TABLE | Same bounds row; cursor clamp as data | `TestResizeInvalidDimensions` |
| `TestWriteResponseRaceCondition` | T36 | — | KEEP | Concurrency smoke test | — |
| `TestCursorBoundsAfterGrowCols` | T31 | — | RENAME | Named subtest; asserts the line GREW as well as the cursor staying in bounds, so it is not a row of `cursorInBoundsCases` | `TestTerminalResizeBounds/bounds after grow cols` |
| `TestCursorBoundsAfterWrap` | T31 | — | TABLE | Same bounds row; wrap path as data | `TestResizeInvalidDimensions` |
| `TestInputWithInvalidCursorPosition` | T31 | — | TABLE | Same bounds row; fill path as data | `TestResizeInvalidDimensions` |
| `TestResizeShrinkWithCursorInBounds` | T32 | — | RENAME | Named subtest; asserts no scrollback growth, content preserved and cursor unmoved | `TestTerminalResizeScrollback/shrink with cursor in bounds` |
| `TestResizeShrinkWithCursorOutOfBounds` | T32, PR1 | — | RENAME | Named subtest; asserts scrollback growth, cursor clamp and content near the cursor | `TestTerminalResizeScrollback/shrink with cursor out of bounds` |
| `TestResizeShrinkScrollbackContent` | T32 | — | RENAME | Named subtest; asserts what reached the scrollback, not what stayed on screen | `TestTerminalResizeScrollback/shrink scrollback content` |
| `TestResizeGrowPullsFromScrollback` | T32, PR1 | — | KEEP | Direct grow-pull contract | — |
| `TestResizeGrowNoScrollbackUnchanged` | T32 | — | RENAME | Named subtest; asserts the scrollback was NOT consumed | `TestTerminalResizeScrollback/grow no scrollback unchanged` |
| `TestResizeAlternateScreenNoScrollback` | T33, PR1 | — | KEEP | Direct alternate-screen resize contract | — |
| `TestResizeCursorPositionAfterShrink` | T32 | — | RENAME | Named subtest; asserts the cursor line stayed visible | `TestTerminalResizeScrollback/cursor position after shrink` |
| `TestViewportRowToAbsolute` | T34 | — | KEEP | Direct conversion contract | — |
| `TestAbsoluteRowToViewport` | T34 | — | RENAME | Named subtest; a different function from `ViewportRowToAbsolute`, with its own in/want table inside | `TestRowCoordinateConversion/absolute to viewport` |
| `TestRowConversionRoundTrip` | T34 | — | RENAME | Named subtest; a property over both functions, not a case of either | `TestRowCoordinateConversion/round trip` |
| `TestSetUserVar` | U01 | — | KEEP | Direct user-variable contract | — |
| `TestGetUserVarNotSet` | U01 | — | TABLE | Same user-var row; unset case as data | `TestSetUserVar` |
| `TestGetUserVars` | U01 | — | TABLE | Same user-var row; all-vars case as data | `TestSetUserVar` |
| `TestGetUserVarsReturnsACopy` | U01 | — | TABLE | Same user-var row; copy case as data | `TestSetUserVar` |
| `TestClearUserVars` | U01 | — | TABLE | Same user-var row; clear case as data | `TestSetUserVar` |
| `TestUserVarOverwrite` | U01 | — | TABLE | Same user-var row; overwrite case as data | `TestSetUserVar` |
| `TestUserVarEmptyValue` | U01 | — | TABLE | Same user-var row; empty-value case as data | `TestSetUserVar` |
| `TestUserVarMiddleware` | U03 | — | KEEP | Direct middleware contract | — |
| `TestUserVarMiddlewareBlocks` | U03 | — | TABLE | Same middleware row; blocking as data | `TestUserVarMiddleware` |
| `TestUserVarThreadSafety` | U01, U03 | — | KEEP | Concurrency smoke test | — |
| `TestOSC1337SetUserVar` | U02 | — | KEEP | Direct OSC 1337 parsing contract | — |
| `TestOSC1337SetUserVarWithST` | U02 | — | TABLE | Same OSC row; ST terminator as data | `TestOSC1337SetUserVar` |
| `TestOSC1337InvalidBase64` | U02 | — | TABLE | Same OSC row; invalid base64 as data | `TestOSC1337SetUserVar` |
| `TestOSC1337EmptyValue` | U02 | — | TABLE | Same OSC row; empty value as data | `TestOSC1337SetUserVar` |
| `TestOSC1337SpecialCharacters` | U02 | — | TABLE | Same OSC row; special chars as data | `TestOSC1337SetUserVar` |
| `TestUserVarsWithPTYWriter` | U02 | — | TABLE | Same OSC row; no-response check as data | `TestOSC1337SetUserVar` |
| `TestMiddlewareMergeSetUserVar` | T25 | — | MERGE | **M3 at head (both forms) — taken for the first time by anyone.** (b) dropping `middlewareMergeSetUserVarCase` loses **no** middleware block (only the `image.go:342` flake). (a) neutering `middleware.go:437` `m.SetUserVar = other.SetUserVar` reddens `TestMiddlewareMerge/set_user_var` **and** `TestUserVarMiddleware/{intercepts,blocks}`: the case catches the break, but it is not the only catcher. Gates block, M3 | `TestMiddlewareMerge/set user var` |
| `TestRuneWidth` | W1 | — | KEEP | Direct width function contract | — |
| `TestIsWideRune` | W1 | — | TABLE | Same width row; `isWide` wrapper as data | `TestRuneWidth` |
| `TestStringWidth` | W1 | — | TABLE | Same width row; string sum as data | `TestRuneWidth` |
| `TestWidthFunctionsAgree` | W3 | — | KEEP | Whole-contract agreement over all runes | — |
| `TestWidthTableMatchesGenerateDirective` | W2 | — | KEEP | Direct version-gate contract | — |
| `TestWorkingDirectory_Basic` | WD1 | — | KEEP | Direct OSC 7 parse contract | — |
| `TestWorkingDirectory_STTerminator` | WD1 | — | TABLE | Same row; ST terminator as data | `TestWorkingDirectory_Basic` |
| `TestWorkingDirectory_Multiple` | WD1 | — | TABLE | Same row; update case as data | `TestWorkingDirectory_Basic` |
| `TestWorkingDirectory_NotSet` | WD1 | — | TABLE | Same row; unset case as data | `TestWorkingDirectory_Basic` |
| `TestWorkingDirectoryPath_Basic` | WD1 | — | TABLE | Same row; path extraction as data | `TestWorkingDirectoryPath/basic` |
| `TestWorkingDirectoryPath_WithHostname` | WD1 | — | TABLE | Same row; hostname case as data | `TestWorkingDirectoryPath/with hostname` |
| `TestWorkingDirectoryPath_EmptyHostname` | WD1 | — | TABLE | Same row; empty-hostname case as data | `TestWorkingDirectoryPath/empty hostname` |
| `TestWorkingDirectoryPath_NotSet` | WD1 | — | TABLE | Same row; path-not-set case as data | `TestWorkingDirectoryPath/not set` |
| `TestWorkingDirectory_Middleware` | WD1 | — | KEEP | Distinct middleware path | — |
| `TestWorkingDirectoryPath` | WD1 | — | KEEP | New host created by `5346c22` for the four `WorkingDirectoryPath_*` rows: `Path()` is a different accessor from `WorkingDirectory()`, so they did not fold into `TestWorkingDirectory`. Added in revision 2; the revision-1 name map sent them to the wrong host | — |
| `TestColorToHex/nil` | SN07 | — | KEEP | Already a table case at baseline; direct `colorToHex` contract | — |
| `TestColorToHex/black` | SN07 | — | KEEP | Already a table case at baseline | — |
| `TestColorToHex/white` | SN07 | — | KEEP | Already a table case at baseline | — |
| `TestColorToHex/red` | SN07 | — | KEEP | Already a table case at baseline | — |
| `TestColorToHex/indexed` | SN07 | — | KEEP | Already a table case at baseline; the only `IndexedColor` path | — |
| `TestSnapshot_UnderlineStyles/single` | SN04 | — | TABLE | Same style row; SGR sequence and expected field value as data | `TestSnapshot_Styles/underline single` |
| `TestSnapshot_UnderlineStyles/single_4:1` | SN04 | — | TABLE | Same style row; sequence as data | `TestSnapshot_Styles/underline single_4:1` |
| `TestSnapshot_UnderlineStyles/double` | SN04 | — | TABLE | Same style row; sequence as data | `TestSnapshot_Styles/underline double` |
| `TestSnapshot_UnderlineStyles/curly` | SN04 | — | TABLE | Same style row; sequence as data | `TestSnapshot_Styles/underline curly` |
| `TestSnapshot_UnderlineStyles/dotted` | SN04 | — | TABLE | Same style row; sequence as data | `TestSnapshot_Styles/underline dotted` |
| `TestSnapshot_UnderlineStyles/dashed` | SN04 | — | TABLE | Same style row; sequence as data | `TestSnapshot_Styles/underline dashed` |
| `TestSnapshot_BlinkStyles/slow` | SN04 | — | TABLE | Same style row; blink field as data | `TestSnapshot_Styles/blink slow` |
| `TestSnapshot_BlinkStyles/fast` | SN04 | — | TABLE | Same style row; blink field as data | `TestSnapshot_Styles/blink fast` |
| `TestResizeOnAlternateScreenWithPrimaryCursorAboveShrink` | R01, PR1 | — | TABLE | Parent of two baseline subtests, both folded; the func itself is gone | `alternateResizeCases` |
| `TestResizeOnAlternateScreenThenPrimaryOutputKeepsPrimary` | R01, PR1 | — | TABLE | Parent of three baseline subtests, all folded; the func itself is gone | `alternateResizeCases` |
| `TestResizeOnAlternateScreenThenHeightIndependentScrollKeepsPrimary` | R01, PR1 | — | TABLE | Parent of two baseline subtests, both folded; the func itself is gone | `alternateResizeCases` |
| `TestSemanticPromptMark_CommandFinishedWithExitCode` | P01 | — | TABLE | Parent of three baseline subtests, all folded into the mark-type table | `TestSemanticPromptMark_Types` |
| `(ADD)` | CL1 | — | ADD | `colors.go` resolution functions have no direct test coverage | `TestColorsResolution`, split by concern into `TestDefaultPalette` / `TestResolveDefaultColor` / `TestResolveDefaultColorNamed` by the SPLIT commit (size limit) |
| `terminal_test.go` | — | — | SPLIT | Over the 1 000 code-line hard limit after the consolidations; the resize concern moved to `resize_test.go` (`34b34eb`). At this head the two files are 641 and 388 code lines | `resize_test.go` |

### Revision-2 rows (fixtures and table shape)

These rows are not baseline tests; they are the fixture and table-shape work the 02-audit review's
HIGH 4 (the suite grew) and LOW 3 (closures in tables) findings required. Each has a verdict, an
evidence line and a target like any other row.

| Row | Subject | Verdict | Evidence (final) | Target |
|---|---|---|---|---|
| X01 | twelve copies of the `testScrollback` + `SetMaxLines(100)` + `New(WithSize(r, 80), WithScrollback(storage))` block | EXTRACT | Repeated setup across `resize_test.go` (11) and `terminal_test.go` (1); guideline §2.5 | `newScrollbackTerm(rows, opts...)` in `helpers_test.go` |
| X02 | four copies of the "write Line0..LineN, last without a newline" loop | EXTRACT | Repeated setup in `resize_test.go` | `writeNumberedLines(term, prefix, n)` |
| X03 | four copies of the make/fill/base64 RGBA payload block | EXTRACT | Repeated setup in `kitty_test.go` | `kittyRGBAPayload(w, h, fill)` |
| C1 | `bufferGrowCases` (`setup`/`grow`/`checkDim`/`checkPres` func fields) | REWRITE | Three distinct assertions with three distinct messages had been collapsed into one boolean `checkPres` and one message — the "one reason to fail" loss the 02-audit review found; the case data is cells, not behaviour | seeded/expected `bufferCellWant` cells, one message per cell |
| C2 | `userVarsCases` (`test func(t, *Terminal)` field) | REWRITE | A slice of subtests at package scope, not variation as data; six of the seven cases are Set calls and expected reads. "get all variables" and "clear all variables" asserted a length and one key where the contract is the whole map | `sets`/`clear`/`wantVars`/`wantAll` data; `maps.Equal` on the whole map; "get all returns a copy" written out, since it mutates the returned map |
| C3 | `deletePlacementsCases` (`setup`/`delete` func fields) | REWRITE | The closures hold placements and a selector, both of which are data | `places`/`del`/`args` data with the selector switched in the runner |
| C4 | `parseKittyGraphicsCases` (`check func(t, *KittyCommand)` field) | REWRITE | The closure held the ASSERTION, and each case asserted only the two or three fields it happened to look at; the parser's contract is the whole command it returns (guideline §2.4) | a whole `want KittyCommand` compared with `reflect.DeepEqual`, plus the payload length |
| C5 | `terminalImageClearingCases`, `sixelEndToEndCases`, `snapshotImagesCases` (`setup`/`act` func fields) | KEEP | Judged and kept: their closures build the SCENARIO (terminal writes, image stores) while every assertion lives once in the shared runner, so no assertion is collapsed and "one reason to fail" holds. Turning a sequence of terminal operations into data would need an interpreter for them — more test code, not less | — |

Deliberate non-fold: the twenty rows re-verdicted `RENAME` above. Each was checked case by case;
in every one the cases differ in the API called or the contract asserted, not in input and expected
output. §2.2 folds variation, not behaviour, and the LOW 3 finding above is what forcing the second
kind into a table costs.

## Refactor progress

Commits landed (branch base `a2b682e`; hashes below are the post-rebase ones — the pre-excision
hashes were `c0fbc9c`, `bfb4727`, `ea4b09d`, `a62d18a`, `9cf1707`, `88a3b15`, `a040066`,
`73a55b6`, `37adadd`, `9089dce`, `7f676b1` in the same order, with `9289383` excised per the
ruling above):

| Verdict kind | Commit | Summary |
|---|---|---|
| ADD | `e414a49` | `TestColorsResolution` |
| EXTRACT | `9b314a1` | shared `testScrollback` fixture |
| TABLE | `5346c22` | working-directory and user-var tests |
| TABLE | `c325569` | buffer tests |
| TABLE | `eeed24e` | image tests |
| TABLE | `f8550ba` | width-function tests |
| TABLE | `6add4f7` | snapshot tests (also rewrote `TestSnapshot_UnderlineColor`) |
| TABLE | `046bba9` | sixel tests (also rewrote `TestSixelScrollingAtBottom` — mixed verdict kind, ruling above) |
| MERGE | `7bc970d` | notification-provider tests |
| REWRITE | `1789aea` | `TestClipboardProvider` (changed from DELETE after mutation evidence) |
| FIX | `8faf70b` | type mismatches and table expectations in buffer, image, sixel, snapshot tests |
| SPLIT | `2d8fa5a` | oversized table-driven tests under the 50-line function limit (ruling above) |
| TABLE | `cc969b8` | `TestBufferCellOutOfBounds` folded into `TestBufferCell` |
| TABLE | `fa669f3` | `TestSnapshot_Cursor` folded into `TestSnapshot_Text` |
| TABLE | `7d36b3b` | `TestUserVarMiddlewareBlocks` folded into `TestUserVarMiddleware` |
| MERGE | `6deec26` | `TestMiddlewareMergeSetUserVar` into `TestMiddlewareMerge`; restores the desktop-notification merge case dropped by `7bc970d` |
| REWRITE | `04326c2` | `TestImageManager_Prune` asserts the eviction contract (real rewrite; `eeed24e`'s body could not fail) |
| TABLE + RENAME | `0eb7472` | kitty tests; also carries 4 rows re-verdicted `RENAME` in revision 2 (mixed kinds, ruling above) |
| TABLE + RENAME | `79f2452` | semantic-prompt tests; also carries the 2 `ScrollbackNavigation` `RENAME` rows (mixed kinds, ruling above) |
| TABLE + RENAME + SPLIT | `34b34eb` | terminal tests; resize concern split into `resize_test.go`; also carries 10 `RENAME` rows — five resize-scrollback, three recording, two row-conversion (mixed kinds, ruling above) |
| TABLE | `d6b1161` | alternate-screen resize scenarios into `TestResizeOnAlternateScreenKeepsPrimary` |
| SPLIT | `4c004f3` | `checkImageData` helper under the function limit |
| FIX | `8c77e78` | repeat-unsafe notification table and unreachable prune case, caught by verify (ruling above) |

Revision-2 commits (2026-09-17), in the guideline's order among themselves:

| Verdict kind | Commit | Summary |
|---|---|---|
| EXTRACT | `1c024cb` | rows X01-X03: shared scrollback-terminal, numbered-lines and kitty-payload fixtures |
| TABLE | `f8679b4` | the **sixteen** rows that were subtest extraction folded into real data tables (R01/PR1 x8, T31 x3, T34, T30, SN01, and the three parents they collapse). The other twenty of the ~44 flagged rows were re-verdicted `RENAME`, not folded — `32e1182` exists to retract this row's earlier "twenty folded" over-count |
| REWRITE | `b2dd063` | rows C1-C4: func-typed table fields replaced by data and whole-contract assertions |

Outstanding: none. Every verdict row is executed. DELETE: none — `TestClipboardProvider` was
changed to REWRITE based on mutation evidence (committed as `1789aea`);
`TestActiveCharsetBoundsValidation` changed to KEEP based on mutation evidence.

## Blocker (resolved)

The earlier block — the `burrowee-ci` Clawee CI lock at `/tmp/ci-lock/clawee` held by `mjc` for
`2026-09-15-clawee-connection-stability` — was cleared by waiting: the finishing session found the
lock free, resumed, and re-checked it before each run (locks held by another session are reported,
never broken).

The second block — `burrowee-ci` unreachable from 2026-09-17 (ICMP and TCP/22 answered, the ssh
banner never completed), which stopped the whole verify phase — was cleared by the machine being
restarted at 23:19 local. It was confirmed back before any run: `ps -eo etime` on PID 1 read
`16:05` (the VM's own age; `uptime` there reports the shared host kernel and reads `up 4 days`),
`/tmp/ci-env.sh` was present, `ci-lock status clawee: free`, no `go test` process of any brand was
running, and the load average was falling through 0.53 / 2.36 / 16.61. No run in this session ever
had to wait for another suite.

## Gates

All four verify runs and all six mutation runs were taken at head `32e1182` on `burrowee-ci`
through `ci/run-tests.sh`, on 2026-09-17 (2026-09-18 UTC), after the machine came back. Evidence
under `reviews/2026-09-14-clawee-go-headless-term-coverage/`; the `after-*` directories now hold
this head's runs, replacing the `8c77e78` ones they carried before (those stay reachable in git
history).

**Machine note, recorded because the brief for this session assumed otherwise.** `burrowee-ci`
migrated the Clawee lock to the flock tool `ci-lock run clawee` (`/tmp/ci-lock/clawee.lock` +
`clawee.who`). The two schemes **interoperate by design** — `ci-lock` waits on a live old
directory lock and leaves a `by=ci-lock` marker at `/tmp/ci-lock/clawee` so an old runner waits
for it in turn (`/usr/local/bin/ci-lock`, "WHY the old directory locks are still read"). Taking
`ci-lock run clawee` by hand *around* this runner therefore does not protect it, it blocks it:
the attempt exited `3` with `by=ci-lock user=jc pid=68596`. The hand-held lock was released and
every run below took the runner's own lock, which is real mutual exclusion against `ci-lock`
runs. Nothing about `ci/run-tests.sh` was changed (that is feature 01's surface).

Workstation checks at head `32e1182` (workstation-only; the suite is never run here):

    $ go version
    go version go1.26.6 darwin/arm64
    $ gofmt -s -l .
    providers.go
    terminal.go
    $ goimports -l *_test.go
    (empty)
    $ GOOS=linux GOWORK=off go vet ./...   → rc 0

`gofmt` reports exactly `providers.go` and `terminal.go` — dev's pre-existing unformatted state
(excision ruling); every test file this branch touches is formatted. `vet` is the test-file
compile gate; `go build` does not compile `*_test.go`.

**No production-code change over the whole range**

    $ git diff --stat a2b682e..32e1182 -- ':!*_test.go' ':!**/testdata/**' ':!reviews/**'
    (empty)

**Coverage — PASS.** After ⊇ baseline, as a set.

    $ LC_ALL=C comm -23 reviews/2026-09-14-clawee-go-headless-term-coverage/baseline-covered.txt \
        reviews/2026-09-14-clawee-go-headless-term-coverage/after-evidence/covered.txt
    (empty)                                    # 0 lines; baseline 952, after 971
    $ LC_ALL=C comm -13 … | sed 's/:.*//' | sort | uniq -c
      15 github.com/clawee-git/go-headless-term/colors.go
       4 github.com/clawee-git/go-headless-term/image.go

The same gate against the other two evidence runs is empty too: `after-shuffle` 971 blocks,
`after-repeat` 970 — the one block `after-repeat` lacks is `image.go:342.53,344.5`, the known
non-deterministic block, which is not in the baseline set and so costs the gate nothing.
`after-evidence/covered.txt` and `after-repeat/covered.txt` are byte-identical to the `8c77e78`
runs they replace: the revision-2 commits `1c024cb`, `f8679b4` and `b2dd063` moved no coverage.

**Deletion evidence — PASS for all three MERGE rows, taken at head, both forms each.** Form (b)
removes the folded case and compares covered sets against the control (`after-evidence`); form (a)
breaks the production line the row claims to protect and reads the failing names. Form (a) mutates
production files **in a scratch copy only** (`git archive HEAD | tar -x`); the worktree was never
touched and every file was compared back afterwards. No mutation is committed.

*M1 — `TestMiddlewareMergeDesktopNotification` → `TestMiddlewareMerge/desktop notification`.*

    (b) drop middlewareMergeDesktopNotificationCase + its t.Run     rc 0, 190 cases, 970 blocks
    $ LC_ALL=C comm -23 after-evidence/covered.txt mutate-merge-desktop-drop-case/covered.txt
    github.com/clawee-git/go-headless-term/image.go:342.53,344.5    ← the known flake, only

    (a) middleware.go:434 `m.DesktopNotification = other.DesktopNotification` neutered   rc 1
    === fail … TestDesktopNotificationMiddleware/intercepts_and_modifies
    === fail … TestDesktopNotificationMiddleware/blocks
    === fail … TestDesktopNotificationMiddleware
    === fail … TestMiddlewareMerge/desktop_notification
    === fail … TestMiddlewareMerge

**Zero middleware blocks lost** — reproducing the 02-audit review's row 1 at this head and
confirming the corrected ruling: `middleware.go:433.38,435.3` is reached by
`TestDesktopNotificationMiddleware` through `WithMiddleware` → `terminal.go:287 Merge`, so the
folded case never was its coverage home. Form (a) shows the case does **fail** when the merge is
broken, so the behavioural assertion the MERGE preserved is real — but it is not the unique
catcher, and the row is recorded that way rather than claimed stronger.

*M2 — `TestNoopNotification` → `TestNotificationProviderWiring/default is NoopNotification`.*

    (b) drop the "default is NoopNotification" subtest                rc 0, 190 cases, 969 blocks
    $ LC_ALL=C comm -23 after-evidence/covered.txt mutate-noop-notify-drop-case/covered.txt
    github.com/clawee-git/go-headless-term/image.go:342.53,344.5     ← the flake
    github.com/clawee-git/go-headless-term/providers.go:316.69,316.82 ← NoopNotification.Notify

    (a) providers.go:316 `return ""` → `return "mutated"`                               rc 1
    === fail … TestNotificationProviderWiring/default_is_NoopNotification
    === fail … TestNotificationProviderWiring

**Row confirmed both ways, and it is the strongest of the three**: the folded case is the unique
coverage home of `providers.go:316.69,316.82` *and* the unique catcher of the mutation.

*M3 — `TestMiddlewareMergeSetUserVar` → `TestMiddlewareMerge/set user var`. Taken here for the
first time: neither the audit nor the 02-audit review ever ran this row.*

    (b) drop middlewareMergeSetUserVarCase + its t.Run              rc 0, 190 cases, 970 blocks
    $ LC_ALL=C comm -23 after-evidence/covered.txt mutate-merge-setuservar-drop-case/covered.txt
    github.com/clawee-git/go-headless-term/image.go:342.53,344.5    ← the known flake, only

    (a) middleware.go:437 `m.SetUserVar = other.SetUserVar` neutered                    rc 1
    === fail … TestMiddlewareMerge/set_user_var
    === fail … TestMiddlewareMerge
    === fail … TestUserVarMiddleware/intercepts
    === fail … TestUserVarMiddleware/blocks
    === fail … TestUserVarMiddleware

**Same shape as M1, and now measured rather than assumed.** `middleware.go:436.30,438.3` is
reached by `TestUserVarMiddleware` through `WithMiddleware` → `Merge`, so dropping the folded case
loses no block; breaking the field copy reddens the folded case **and** `TestUserVarMiddleware`.
The MERGE is behaviourally sound and preserved a real assertion; it is not the unique catcher.

**What the three rows jointly show.** Every middleware-merge row in this suite is reachable two
ways, because `WithMiddleware` itself calls `Merge`. That is a property of the production API, not
a defect in the folds, and it is why coverage alone could never have judged M1 or M3 — exactly the
case §5's mutation requirement exists for. No row's claim is wrong; two of the three are weaker
than "unique catcher" and are now written that way. No row goes back to `ADD` or `REWRITE`.

Earlier mutation evidence, still valid (taken before the REWRITE/KEEP verdict changes, unaffected
by the excision — the ruling above):

- removing `TestClipboardProvider` loses the only coverage of `WithClipboard`/`ClipboardProvider`
  (`terminal.go:266.48,267.27`, `267.27,269.3`, `659.58,663.2`); verdict changed to REWRITE;
- removing `TestActiveCharsetBoundsValidation` loses the only coverage of
  `SetActiveCharset`/`setActiveCharsetInternal` (`handler.go:1289-1303`, four blocks, plus the
  known non-deterministic `image.go:342` block); verdict changed to KEEP.

**Failure-name set, diffed both ways — PASS.** Parsed from the `-json` streams' `"Action":"fail"`
events (`test.json` in each evidence directory):

    baseline-evidence-retake: fail=0 skip=0
    after-evidence:           fail=0 skip=0
    after-shuffle:            fail=0 skip=0
    after-repeat:             fail=0 skip=0
    failure-name diff baseline vs after-evidence: added=[] vanished=[]
    failure-name diff baseline vs after-shuffle:  added=[] vanished=[]
    failure-name diff baseline vs after-repeat:   added=[] vanished=[]

Nothing added, nothing silently vanished; both sets are empty at both ends.

**Skips and counts — PASS.** `"Action":"skip"` events = 0 in every stream, baseline and after
alike; no new skips. Counts from the runner's own per-package summary, evidence run at head:

    === github.com/clawee-git/go-headless-term: tests 112 (pass 112, fail 0, skip 0) · cases 191 (pass 191, fail 0, skip 0)
    === github.com/clawee-git/go-headless-term/internal/generate_width_table: tests 6 (pass 6, fail 0, skip 0) · cases 0 (pass 0, fail 0, skip 0)
    === module: tests 118 (pass 118, fail 0, skip 0) · cases 191 (pass 191, fail 0, skip 0)

226 → 118 tests and 23 → 191 cases. Every drop is a fold accounted for in the name map. The case
count is **191, not the 194 measured at `8c77e78`**: `f8679b4` flattened three grouping subtests
(`TestResizeOnAlternateScreenKeepsPrimary/{primary cursor above shrink, primary output before
grow, height independent scroll}`) into rows of one table, so three case events disappear with no
successor name — recorded in the name map.

**Size — PASS.** Hard limits 1000 code lines per file and 50 per function (comments and blanks
excluded, table literals counted). Largest test file `terminal_test.go` **641**; next
`semantic_prompt_test.go` 435, `resize_test.go` 388. Largest test function `TestSnapshot_Styles`
**49**; nothing at or over 50. Measured over all 16 root `*_test.go` plus
`internal/generate_width_table/main_test.go`.

## Name map

| Before | After |
|---|---|
| `TestBufferCellOutOfBounds` | `TestBufferCell` table case |
| `TestBufferScrollDown` | `TestBufferScroll` table case |
| `TestBufferDeleteChars` | `TestBufferLineEdits` table case |
| `TestBufferWrappedLineTrackingWithScroll` | `TestBufferWrappedLineTracking` table case |
| `TestBufferGrowCols` | `TestBufferGrow` table case |
| `TestImageManager_Deduplication` | `TestImageManager_Store` table case |
| `TestImageManager_Clear` | `TestImageManager_Lifecycle` |
| `TestImageManager_Prune` | `TestImageManager_Prune` (REWRITE in place — same name, real eviction assertions) |
| `TestImageManager_DeletePlacementsInRow`, `InRowRange`, `Below`, `Above` | `TestImageManager_DeletePlacements` |
| `TestClearScreenBelowClearsImages`, `TestResetStateClearsImagesAndCache`, `TestAlternateScreenClearsImages` | `TestTerminalImageClearing` |
| `TestParseKittyGraphics_Query`, `Delete`, `Chunked`, `WithZIndex`, `Placement`, `DoNotMoveCursor` | `TestParseKittyGraphics` |
| `TestKittyCommand_DecodeRGB` | `TestKittyCommand_DecodeImageData` |
| `TestKittyImageCellAssignment`, `UVCoordinates`, `ChunkedTransfer`, `ImageDelete` | `TestKittyImageEndToEnd` |
| `TestNoopNotification`, `TestWithNotificationOption`, `TestSetNotificationProvider` | `TestNotificationProviderWiring` |
| `TestDesktopNotificationWithNilProvider`, `PayloadFields`, `EmptyPayload` | `TestDesktopNotification_Cases` |
| `TestMiddlewareMergeDesktopNotification`, `TestMiddlewareMergeSetUserVar` | `TestMiddlewareMerge` |
| `TestResizeGrowBeforeLeavingAlternateScreenKeepsPrimary`, `WithPrimaryCursorAboveShrink` subtests, `PairsLikePrimaryResize`, `ThenPrimaryOutput` subtests, `HeightIndependentScroll` subtests | `TestResizeOnAlternateScreenKeepsPrimary` table cases |
| `TestSemanticPromptMark_CommandStart`, `CommandExecuted`, `CommandFinished`, `CommandFinishedWithExitCode` subtests | `TestSemanticPromptMark_Types` |
| `TestGetLastCommandOutput_MultiLine`, `NoOutput`, `NoMarks`, `OnlyExecutedNoFinished`, `MultipleCommands`, `WithExitCode`, `TrailingEmptyLines` | `TestGetLastCommandOutput_Cases` |
| `TestSemanticPromptMark_PrevPromptRowWithScrollback`, `GetMarkAtWithScrollback` | `TestSemanticPromptMark_ScrollbackNavigation` |
| `TestParseSixel_MultipleColumns`, `NewLine`, `CarriageReturn`, `Repeat`, `ColorRGB`, `ColorHLS`, `Transparent`, `Empty`, `ComplexImage` | `TestParseSixel` |
| `TestSixelImageCellAssignment`, `CursorMovement` | `TestSixelEndToEnd` |
| `TestSixelScrollingAtBottom` | `TestSixelScrollingAtBottom` (REWRITE in place — same name, new assertions; not folded into `TestSixelEndToEnd`) |
| `TestSnapshot_Cursor`, `EmptyTerminal` | `TestSnapshot_Text` table cases |
| `TestSnapshot_Attributes` | `TestSnapshot_Full` table case |
| `TestSnapshot_BlinkStyles`, `UnderlineColor` | `TestSnapshot_Styles` |
| `TestSnapshot_StyledSegments` | `TestSnapshot_Styled` table case |
| `TestSnapshot_NoImages` | `TestSnapshot_Images` table case |
| `TestTerminalWideCharacterCursorWrite` | `TestTerminalWideCharacter` table case |
| `TestTerminalWrappedLineClearedOnNewline` | `TestTerminalWrappedLineTracking` table case |
| `TestTerminalRecordingWithANSI`, `Clear`, `Replay`, `SetProvider` | `TestTerminalRecording` table cases |
| `TestResizeCursorBounds`, `TestCursorBoundsAfterGrowCols`, `TestCursorBoundsAfterWrap`, `TestInputWithInvalidCursorPosition` | `TestTerminalResizeBounds` |
| `TestResizeShrinkWithCursorInBounds`, `OutOfBounds`, `ScrollbackContent`, `GrowNoScrollbackUnchanged`, `CursorPositionAfterShrink` | `TestTerminalResizeScrollback` |
| `TestAbsoluteRowToViewport`, `TestRowConversionRoundTrip` | `TestRowCoordinateConversion` |
| `TestGetUserVarNotSet`, `TestGetUserVars`, `ReturnsACopy`, `ClearUserVars`, `UserVarOverwrite`, `EmptyValue` | `TestUserVars` |
| `TestOSC1337SetUserVarWithST`, `InvalidBase64`, `EmptyValue`, `SpecialCharacters`, `TestUserVarsWithPTYWriter` | `TestOSC1337SetUserVar` |
| `TestUserVarMiddlewareBlocks` | `TestUserVarMiddleware` table case |
| `TestIsWideRune`, `TestStringWidth` | `TestWidthFunctions` |
| `TestWorkingDirectory_Basic`, `_STTerminator`, `_Multiple`, `_NotSet` | `TestWorkingDirectory` table cases |
| `TestWorkingDirectoryPath_Basic`, `_WithHostname`, `_EmptyHostname`, `_NotSet` | `TestWorkingDirectoryPath` table cases (a new host, not `TestWorkingDirectory`) |
| `TestColorsResolution` subtests (SPLIT, size limit) | `TestDefaultPalette/standard colors`, `/color cube`, `/grayscale`; `TestResolveDefaultColor/nil`, `/RGBA passthrough`, `/NRGBA fallback`, `/indexed`, `/indexed out of range`; `TestResolveDefaultColorNamed/named`, `/named dim`, `/named out of range` |

Revision-2 renames (commit `f8679b4`, the folds that finished): the three intermediate grouping
subtests disappear, because their nested cases become rows of one flat table.

| Before | After |
|---|---|
| `TestResizeOnAlternateScreenKeepsPrimary/primary cursor above shrink/growWhileAlternate=false` | `…/primary cursor above shrink, grow after leaving` |
| `…/primary cursor above shrink/growWhileAlternate=true` | `…/primary cursor above shrink, grow while alternate` |
| `…/primary output before grow/lines=19` (and `=20`, `=30`) | `…/primary output before grow, lines=19` (and `, lines=20`, `, lines=30`) |
| `…/height independent scroll/SU 3` | `…/height independent scroll, SU 3` |
| `…/height independent scroll/region 1;10 scroll` | `…/height independent scroll, region 1;10 scroll` |
| `TestTerminalResizeBounds/invalid cursor position` | `TestTerminalResizeBounds/bounds after overfilling the grid` |

Gone with no successor name: the three grouping levels
`TestResizeOnAlternateScreenKeepsPrimary/{primary cursor above shrink, primary output before grow,
height independent scroll}` — three case events, accounted for in the `After` counts.

Host renames that carry no row of their own above: `TestParseKittyGraphics_Basic` →
`TestParseKittyGraphics`, `TestKittyCommand_DecodeRGBA` → `TestKittyCommand_DecodeImageData`,
`TestKittyImageDisplay` → `TestKittyImageEndToEnd`, `TestSemanticPromptMark_PromptStart` →
`TestSemanticPromptMark_Types`, `TestGetLastCommandOutput_Basic` → `TestGetLastCommandOutput_Cases`,
`TestSemanticPromptMark_NextPromptRowWithScrollback` → `TestSemanticPromptMark_ScrollbackNavigation`,
`TestResizeInvalidDimensions` → `TestTerminalResizeBounds`, `TestResizeGrowPullsFromScrollback` →
`TestTerminalResizeScrollback`, `TestViewportRowToAbsolute` → `TestRowCoordinateConversion`.

## After

Verify runs at head `32e1182` on `burrowee-ci`, all four green (evidence under
`reviews/2026-09-14-clawee-go-headless-term-coverage/after-*`):

- plain `ci/run-tests.sh` — **the runtime of record**: `ok github.com/clawee-git/go-headless-term
  0.138s` · `ok .../internal/generate_width_table 0.002s` (rc 0)
- evidence (`--artifacts after-evidence`): rc 0, `covered 971 blocks`
- `--shuffle --repeat 3 --artifacts after-shuffle`: rc 0, seeds `1789713719586658617` (root) /
  `1789713719623347600` (internal), 354 tests / 573 cases, `covered 971 blocks`
- `--repeat 3 --artifacts after-repeat`: rc 0, 354 tests / 573 cases, `covered 970 blocks` (the
  known `image.go:342` flake)

| package | files | test code lines | tests | cases | skips | covered blocks | runtime | failing |
|---|---|---|---|---|---|---|---|---|
| `headlessterm` | 16 | 4 273 | 112 | 191 | 0 | 971 (⊇ baseline 952) | 0.138 s | 0 |
| `internal/generate_width_table` | 1 | 97 | 6 | 0 | 0 | included in module set | 0.002 s | 0 |
| module | 17 | 4 370 | 118 | 191 | 0 | 971 (⊇ baseline 952) | 0.140 s | 0 |

Against the baseline table: files 14 → 17, test code lines **4 142 → 4 370 (+228, +5.5 %)**,
tests 226 → 118, cases 23 → 191, skips 0 → 0, covered blocks 952 → 971, runtime 0.175 s →
0.140 s, failing 0 → 0. Code lines counted as `architecture.md` §3 counts them: comments, block
comments and blank lines stripped, string, rune and raw literals preserved.

**The suite is bigger, and that is the honest result.** The stated goal of this process is the
same coverage from less test code, and the root package grew by **228 code lines (4 045 →
4 273)**. Revision 2 recovered 110 of the 338 lines the 02-audit review measured at `c412993`
(4 383), by finishing sixteen folds into real data tables, extracting three shared fixtures and
replacing four `func`-typed table fields with data — but it did not get back to baseline and does
not claim to. Where the growth is, and what was bought with it:

| | lines | what it bought |
|---|---|---|
| `colors_test.go` (ADD) | +148 | an inventory row that had **no test at all** — 15 of the 19 gained covered blocks are its |
| `helpers_test.go` (EXTRACT), file alone | +48 | shared fixtures that removed copy-paste setup from four files |
| everything else, net | +32 | 23 → 191 real subtests, whole-contract assertions replacing `t.Logf`, and a rewritten `TestImageManager_Prune` that can now fail |

**The honest compaction figure is +80, not +32** (corrected after round 2 of the 02-audit review).
The two rows above are a per-file decomposition and nothing more: +32 is obtained by charging the
EXTRACT's cost to a separate line while keeping its savings inside "everything else", which flatters
the result. An EXTRACT *is* one of the guideline's verdict kinds and is §2.5 compaction work, and
measured across both EXTRACT commits it is net **−39** for the suite — `9b314a1` 4 189→4 182
(helpers +33, other files −40) and `1c024cb` 4 383→4 351 (helpers +15, other files −47): **87 lines
moved out of the 14 pre-existing files**. So the defensible statements are **+80 excluding only the
ADD** (4 273 − 148 = 4 125 against baseline 4 045), or **+119** if the extract's effect is excluded
along with its cost. Against 168 more counted cases and 19 more covered blocks, +80 is the number
that argument has to stand on; the compaction the guideline asks for did not
happen here, and the reason is in the TABLE ruling above: roughly half the rows recorded `TABLE`
described cases that differ in *behaviour*, and forcing those into tables is the defect the review
found in `bufferGrowCases`, not a saving.

Deliberately lost coverage: none — the `comm -23` gate above is empty.
New skips: none — 0 at baseline, 0 in all three after runs.
Known flake, named and not chased: `image.go:342.53,344.5` covers non-deterministically
(971/970); it is absent from the baseline set, so it never moves a gate.
