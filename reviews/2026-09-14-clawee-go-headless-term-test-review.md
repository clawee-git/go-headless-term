# Test-suite review — go-headless-term

Module: `github.com/clawee-git/go-headless-term` · baseline commit `a2b682e` · after commit `8c77e78` (test-code head; the report and evidence commit follows it) · suite command: `ci/run-tests.sh [options] [pkg...]` · target: `burrowee-ci` · runtime = plain uninstrumented run · covered = set mode profile

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

**Mixed-verdict commit `046bba9` (TABLE + REWRITE) left as is.** That commit consolidates the ten
`TestParseSixel_*` functions and three end-to-end sixel tests into tables (TABLE) and also rewrites
`TestSixelScrollingAtBottom`'s assertions (REWRITE). The two portions are textually disjoint (the
rewrite is a separate function body, not folded into `TestSixelEndToEnd`), so strictly it should
have been two commits, TABLE then REWRITE. It was left unsplit because splitting a mid-chain commit
under the follow-on FIX commit (which touches the same file) risks rebase churn for a cosmetic
ordering gain, and the deviation does not undermine the order rule's purpose: no `DELETE`/`MERGE`
evidence depended on sequence here (there are no DELETEs at all), and the rewrite's target still
exists under its own name with its verdict, evidence and target recorded below. Recorded as a
documented deviation from "one verdict kind per commit".

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
not depend on commit sequence. `6deec26` also repairs a coverage drop from `7bc970d`: that MERGE
had removed `TestMiddlewareMergeDesktopNotification` without a replacement, uncovering
`Middleware.Merge`'s `DesktopNotification` field-copy branch; the case now exists as a subtest of
the `TestMiddlewareMerge` host.

**Verify phase caught two test-code defects (fixed in `8c77e78`).** (1) `desktopNotificationCases`
embedded `*testNotificationProvider` instances in the package-level table, so `notifyCount`
accumulated across `-count=3` repetitions (`expected 1 notifications, got 2`, then 3) — a
consolidation defect from `7bc970d`, invisible to single runs, caught by the shuffle and repeat
runs; providers are now constructed inside each subtest. (2) The prune rewrite's third case
asserted an unreachable state: `Store` prunes immediately after adding, before any `Place` can
reference the new image, so "all-referenced over budget" cannot arise through the API; the case
now documents the real contract — an over-budget image is evicted by its own `Store` call. No
production bug found; no audit row stopped on a defect.

## Baseline

| package | files | test code lines | tests | cases | skips | covered blocks | runtime | failing |
|---|---|---|---|---|---|---|---|---|
| `headlessterm` | 13 | 4 045 | 220 | 23 | 0 | 952 | 0.173 s | 0 |
| `internal/generate_width_table` | 1 | 92 | 6 | 0 | 0 | included in module set | 0.002 s | 0 |
| module | 14 | 4 137 | 226 | 23 | 0 | 952 | 0.175 s | 0 |

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
| `TestKittyImageCellAssignment` | K04 | — | TABLE | Same end-to-end row; cell assignment detail | `TestKittyImageDisplay` |
| `TestKittyImageUVCoordinates` | K04 | — | TABLE | Same end-to-end row; asserts UV detail | `TestKittyImageDisplay` |
| `TestKittyChunkedTransfer` | K04 | — | TABLE | Same end-to-end row; chunked path as data | `TestKittyImageDisplay` |
| `TestKittyImageDelete` | K04 | — | TABLE | Same end-to-end row; delete action as data | `TestKittyImageDisplay` |
| `TestNoopNotification` | N01 | — | MERGE | Same wiring/default row as `TestDefaultNotificationProvider` | `TestDefaultNotificationProvider` |
| `TestWithNotificationOption` | N01 | — | TABLE | Same wiring row; option path as data | `TestDefaultNotificationProvider` |
| `TestDefaultNotificationProvider` | N01 | `TestNoopNotification`, `TestWithNotificationOption`, `TestSetNotificationProvider` | KEEP | Direct default-provider contract | — |
| `TestSetNotificationProvider` | N01 | — | TABLE | Same wiring row; runtime set as data | `TestDefaultNotificationProvider` |
| `TestDesktopNotificationHandler` | N02 | `WithNilProvider`, `PayloadFields`, `EmptyPayload` | KEEP | Direct handler contract | — |
| `TestDesktopNotificationWithNilProvider` | N02 | — | TABLE | Same handler row; nil payload as data | `TestDesktopNotificationHandler` |
| `TestDesktopNotificationQueryResponse` | N02 | — | KEEP | Distinct query-response path | — |
| `TestDesktopNotificationMiddleware` | N03 | `TestDesktopNotificationMiddlewareBlocks` | KEEP | Direct middleware contract | — |
| `TestDesktopNotificationMiddlewareBlocks` | N03 | — | TABLE | Same middleware row; blocking as data | `TestDesktopNotificationMiddleware` |
| `TestNotificationPayloadFields` | N02 | — | TABLE | Same handler row; full field list as data | `TestDesktopNotificationHandler` |
| `TestMiddlewareMergeDesktopNotification` | T25 | — | MERGE | Same `Middleware.Merge` row; handler as data | `TestMiddlewareMerge` |
| `TestNotificationProviderThreadSafety` | N03 | — | KEEP | Concurrency smoke test | — |
| `TestNotificationEmptyPayload` | N02 | — | TABLE | Same handler row; empty payload as data | `TestDesktopNotificationHandler` |
| `TestResizeOnAlternateScreenKeepsPrimary` | R01, PR1 | — | KEEP | Complex integration contract for alternate-screen resize | — |
| `TestResizeGrowBeforeLeavingAlternateScreenKeepsPrimary` | R01, PR1 | — | TABLE | Same integration row; ordering as data | `TestResizeOnAlternateScreenKeepsPrimary` |
| `TestResizeOnAlternateScreenWithPrimaryCursorAboveShrink/growWhileAlternate=false` | R01, PR1 | — | TABLE | Same integration row; cursor-above-shrink as data | `TestResizeOnAlternateScreenKeepsPrimary` |
| `TestResizeOnAlternateScreenWithPrimaryCursorAboveShrink/growWhileAlternate=true` | R01, PR1 | — | TABLE | Same integration row; cursor-above-shrink as data | `TestResizeOnAlternateScreenKeepsPrimary` |
| `TestResizeOnAlternateScreenPairsLikePrimaryResize` | R01, PR1 | — | TABLE | Same integration row; event-sequence comparison as data | `TestResizeOnAlternateScreenKeepsPrimary` |
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
| `TestSemanticPromptMark_PrevPromptRowWithScrollback` | P05 | — | TABLE | Same scrollback row; direction as data | `TestSemanticPromptMark_NextPromptRowWithScrollback` |
| `TestSemanticPromptMark_GetMarkAtWithScrollback` | P05 | — | TABLE | Same scrollback row; get-at as data | `TestSemanticPromptMark_NextPromptRowWithScrollback` |
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
| `TestSnapshot_Cursor` | SN01 | — | KEEP | Direct cursor snapshot contract | — |
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
| `TestTerminalWideCharacterCursorWrite` | T12 | — | TABLE | Same row as `TestTerminalWideCharacter`; cursor-positioned wide char as data | `TestTerminalWideCharacter` |
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
| `TestTerminalRecordingClear` | T30 | — | TABLE | Same recording row; clear as data | `TestTerminalRecording` |
| `TestTerminalRecordingReplay` | T30 | — | TABLE | Same recording row; replay as data | `TestTerminalRecording` |
| `TestTerminalRecordingSetProvider` | T30 | — | TABLE | Same recording row; provider set as data | `TestTerminalRecording` |
| `TestActiveCharsetBoundsValidation` | T35 | — | KEEP | Mutation evidence shows it is the only coverage for `SetActiveCharset`/`setActiveCharsetInternal`; keep and improve later if desired | — |
| `TestResizeInvalidDimensions` | T31 | — | KEEP | Direct resize validation contract | — |
| `TestResizeCursorBounds` | T31 | — | TABLE | Same bounds row; cursor clamp as data | `TestResizeInvalidDimensions` |
| `TestWriteResponseRaceCondition` | T36 | — | KEEP | Concurrency smoke test | — |
| `TestCursorBoundsAfterGrowCols` | T31 | — | TABLE | Same bounds row; `GrowCols` path as data | `TestResizeInvalidDimensions` |
| `TestCursorBoundsAfterWrap` | T31 | — | TABLE | Same bounds row; wrap path as data | `TestResizeInvalidDimensions` |
| `TestInputWithInvalidCursorPosition` | T31 | — | TABLE | Same bounds row; fill path as data | `TestResizeInvalidDimensions` |
| `TestResizeShrinkWithCursorInBounds` | T32 | — | TABLE | Same scrollback-resize row; cursor-in-bounds as data | `TestResizeGrowPullsFromScrollback` |
| `TestResizeShrinkWithCursorOutOfBounds` | T32, PR1 | — | TABLE | Same scrollback-resize row; cursor-out-of-bounds as data | `TestResizeGrowPullsFromScrollback` |
| `TestResizeShrinkScrollbackContent` | T32 | — | TABLE | Same scrollback-resize row; content verification as data | `TestResizeGrowPullsFromScrollback` |
| `TestResizeGrowPullsFromScrollback` | T32, PR1 | — | KEEP | Direct grow-pull contract | — |
| `TestResizeGrowNoScrollbackUnchanged` | T32 | — | TABLE | Same scrollback-resize row; no-scrollback as data | `TestResizeGrowPullsFromScrollback` |
| `TestResizeAlternateScreenNoScrollback` | T33, PR1 | — | KEEP | Direct alternate-screen resize contract | — |
| `TestResizeCursorPositionAfterShrink` | T32 | — | TABLE | Same scrollback-resize row; saved-cursor as data | `TestResizeGrowPullsFromScrollback` |
| `TestViewportRowToAbsolute` | T34 | — | KEEP | Direct conversion contract | — |
| `TestAbsoluteRowToViewport` | T34 | — | TABLE | Same conversion row; reverse direction as data | `TestViewportRowToAbsolute` |
| `TestRowConversionRoundTrip` | T34 | — | TABLE | Same conversion row; round-trip as data | `TestViewportRowToAbsolute` |
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
| `TestMiddlewareMergeSetUserVar` | T25 | — | MERGE | Same `Middleware.Merge` row; handler as data | `TestMiddlewareMerge` |
| `TestRuneWidth` | W1 | — | KEEP | Direct width function contract | — |
| `TestIsWideRune` | W1 | — | TABLE | Same width row; `isWide` wrapper as data | `TestRuneWidth` |
| `TestStringWidth` | W1 | — | TABLE | Same width row; string sum as data | `TestRuneWidth` |
| `TestWidthFunctionsAgree` | W3 | — | KEEP | Whole-contract agreement over all runes | — |
| `TestWidthTableMatchesGenerateDirective` | W2 | — | KEEP | Direct version-gate contract | — |
| `TestWorkingDirectory_Basic` | WD1 | — | KEEP | Direct OSC 7 parse contract | — |
| `TestWorkingDirectory_STTerminator` | WD1 | — | TABLE | Same row; ST terminator as data | `TestWorkingDirectory_Basic` |
| `TestWorkingDirectory_Multiple` | WD1 | — | TABLE | Same row; update case as data | `TestWorkingDirectory_Basic` |
| `TestWorkingDirectory_NotSet` | WD1 | — | TABLE | Same row; unset case as data | `TestWorkingDirectory_Basic` |
| `TestWorkingDirectoryPath_Basic` | WD1 | — | TABLE | Same row; path extraction as data | `TestWorkingDirectory_Basic` |
| `TestWorkingDirectoryPath_WithHostname` | WD1 | — | TABLE | Same row; hostname case as data | `TestWorkingDirectory_Basic` |
| `TestWorkingDirectoryPath_EmptyHostname` | WD1 | — | TABLE | Same row; empty-hostname case as data | `TestWorkingDirectory_Basic` |
| `TestWorkingDirectoryPath_NotSet` | WD1 | — | TABLE | Same row; path-not-set case as data | `TestWorkingDirectory_Basic` |
| `TestWorkingDirectory_Middleware` | WD1 | — | KEEP | Distinct middleware path | — |
| `(ADD)` | CL1 | — | ADD | `colors.go` resolution functions have no direct test coverage | `TestColorsResolution`, split by concern into `TestDefaultPalette` / `TestResolveDefaultColor` / `TestResolveDefaultColorNamed` by the SPLIT commit (size limit) |
| `terminal_test.go` | — | — | KEEP | Extracted shared `testScrollback` fixture; file now 986 code lines, under the 1 000-line limit, so SPLIT is no longer required | — |

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
| TABLE | `0eb7472` | kitty tests |
| TABLE | `79f2452` | semantic-prompt tests |
| TABLE + SPLIT | `34b34eb` | terminal tests; resize concern split into `resize_test.go` (mixed kinds, ruling above) |
| TABLE | `d6b1161` | alternate-screen resize scenarios into `TestResizeOnAlternateScreenKeepsPrimary` |
| SPLIT | `4c004f3` | `checkImageData` helper under the function limit |
| FIX | `8c77e78` | repeat-unsafe notification table and unreachable prune case, caught by verify (ruling above) |

Outstanding: none. Every verdict row is executed. DELETE: none — `TestClipboardProvider` was
changed to REWRITE based on mutation evidence (committed as `1789aea`);
`TestActiveCharsetBoundsValidation` changed to KEEP based on mutation evidence.

## Blocker (resolved)

The earlier block — the `burrowee-ci` Clawee CI lock at `/tmp/ci-lock/clawee` held by `mjc` for
`2026-09-15-clawee-connection-stability` — was cleared by waiting: the finishing session found the
lock free, resumed, and re-checked it before each run (locks held by another session are reported,
never broken).

## Gates

Workstation checks (final head `8c77e78`):
- `gofmt -s -l .` reports exactly `providers.go` and `terminal.go` — dev's pre-existing unformatted
  state (see the excision ruling); every file this branch touches is formatted.
- `/Users/hjc/bin/goimports -w` clean on all touched test files.
- `GOOS=linux go build ./...` passes, and `GOOS=linux go vet ./...` passes — vet is the test-file
  compile gate; `go build` does not compile `*_test.go` (an unused import survived a build-only
  check during the split and was caught by vet).

No production-code change:

    $ git diff --stat a2b682e..8c77e78 -- ':!*_test.go' ':!**/testdata/**' ':!reviews/**'
    (empty)

Mutation evidence (taken before the REWRITE/KEEP verdict changes):
- Removing `TestClipboardProvider` loses the only coverage of `WithClipboard`/`ClipboardProvider`
  (`terminal.go:266.48,267.27`, `267.27,269.3`, `659.58,663.2`); verdict changed to REWRITE.
- Removing `TestActiveCharsetBoundsValidation` loses the only coverage of
  `SetActiveCharset`/`setActiveCharsetInternal` (`handler.go:1289-1303`, four blocks, plus the
  known non-deterministic `image.go:342` block); verdict changed to KEEP.

Covered-set gate, failure-name diff, skip counts and per-package counts: quoted below after the
verify runs.

Size limits (hard: 1000 code lines per file, 50 per function; comments and blanks excluded,
table literals counted): largest touched file `terminal_test.go` at 645 code lines; no function
over 50 code lines in any test file (checked over `*_test.go` in both packages).
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
| `TestWorkingDirectory_STTerminator`, `Multiple`, `NotSet`, `Path_Basic`, `WithHostname`, `EmptyHostname`, `Path_NotSet` | `TestWorkingDirectory` |
| `TestColorsResolution` subtests (SPLIT, size limit) | `TestDefaultPalette/standard colors`, `/color cube`, `/grayscale`; `TestResolveDefaultColor/nil`, `/RGBA passthrough`, `/NRGBA fallback`, `/indexed`, `/indexed out of range`; `TestResolveDefaultColorNamed/named`, `/named dim`, `/named out of range` |

Host renames that carry no row of their own above: `TestParseKittyGraphics_Basic` →
`TestParseKittyGraphics`, `TestKittyCommand_DecodeRGBA` → `TestKittyCommand_DecodeImageData`,
`TestKittyImageDisplay` → `TestKittyImageEndToEnd`, `TestSemanticPromptMark_PromptStart` →
`TestSemanticPromptMark_Types`, `TestGetLastCommandOutput_Basic` → `TestGetLastCommandOutput_Cases`,
`TestSemanticPromptMark_NextPromptRowWithScrollback` → `TestSemanticPromptMark_ScrollbackNavigation`,
`TestResizeInvalidDimensions` → `TestTerminalResizeBounds`, `TestResizeGrowPullsFromScrollback` →
`TestTerminalResizeScrollback`, `TestViewportRowToAbsolute` → `TestRowCoordinateConversion`.

## After

Verify runs at head `8c77e78` on burrowee-ci (evidence under
`reviews/2026-09-14-clawee-go-headless-term-coverage/after-*`), all four green:

- plain `ci/run-tests.sh`: `ok github.com/clawee-git/go-headless-term 0.158s` ·
  `ok .../internal/generate_width_table 0.002s` (rc=0)
- evidence (`--artifacts after-evidence`): rc=0, `covered 971 blocks`
- `--shuffle --repeat 3` (seeds 1789538420429992250 / 1789538420430244373): rc=0
- `--repeat 3`: rc=0

Per-package counts (evidence run, `-count=1`):

    === github.com/clawee-git/go-headless-term: tests 112 (pass 112, fail 0, skip 0) · cases 194 (pass 194, fail 0, skip 0)
    === github.com/clawee-git/go-headless-term/internal/generate_width_table: tests 6 (pass 6, fail 0, skip 0) · cases 0 (pass 0, fail 0, skip 0)
    === module: tests 118 (pass 118, fail 0, skip 0) · cases 194 (pass 194, fail 0, skip 0)

Before → after: 226 tests (220 root + 6 internal) / 23 cases → 118 tests (112 root + 6
internal) / 194 cases. Test count halves because 130+ single-purpose tests folded into
table subtests and named case funcs (the name map above); case count rises from 23 to 194
because each folded variation is now its own counted subtest. Nothing was deleted without a
fold — the two verdicts changed off DELETE (`TestClipboardProvider` → REWRITE,
`TestActiveCharsetBoundsValidation` → KEEP) are the only rows whose behavior did not move.

Failure-name set diff (from `test.json` `"Action":"fail"` events, both directions):
baseline `baseline-evidence-retake` has 0 failed test names; after-evidence, after-shuffle
and after-repeat each have 0 failed test names — the two sets are equal (empty).

Covered-set gate (set-mode blocks, sorted):

    $ LC_ALL=C comm -23 reviews/2026-09-14-clawee-go-headless-term-coverage/baseline-covered.txt \
        reviews/2026-09-14-clawee-go-headless-term-coverage/after-evidence/covered.txt
    (empty — after ⊇ baseline)

Baseline 952 blocks, after 971. The reverse direction (`comm -13`, covered after but not in
baseline) lists 19 gained blocks: 15 in `colors.go` (blink styles, underline colors,
`colorToHex` paths from the folded `TestSnapshot_Styles`/`TestColorToHex` tables) and 4 in
`image.go` (`DeletePlacementsByPosition`/`InRow`, prune paths, and the known
non-deterministic `image.go:342.53,344.5`).

Deliberately lost coverage: none — `comm -23` above is empty.
New skips: 0 (baseline 0, after 0 — counted from `test.json` `"Action":"skip"` events)
