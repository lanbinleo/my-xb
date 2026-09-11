# Changelog

## v1.2.1 - 2026-09-11

### Changes / Bug Fixes

- A subject whose score-detail fetch fails (for example when Xiaobao reports the course's scores have not been synced to the mobile API) no longer aborts the whole GPA report. The subject falls back to its official semester score with a warning; subjects without an official score are excluded from GPA calculation. Batch endpoints (subject list, semester-wide scores) remain fatal.
- Degraded runs still match the official GPA when score details are unavailable, because the official semester score was already preferred for final subject scores.
- Subject-level NaN scores now serialize as `null` in JSON output instead of failing JSON encoding.

### Chores

- Added regression tests for the official-score-only fallback and NaN-safe JSON output.
- Documented the degradation behavior in README and GPA_CALCULATION.md.

## v1.2.0 - 2026-09-08

### Features

- Added `myxb attendance` (alias `att`): overall attendance rate, per-state counts, and a per-subject attendance table for the current school year, sorted by absent count descending.
- Added the week timetable grid: `myxb week` (also `myxb schedule week`) renders Monday-Sunday by period with today's column highlighted, free blocks, and rooms — no extra API calls.
- Made the week grid the default view for `myxb schedule` (aliases: `s`, `cal`, `calendar`).
- Added top-level shortcuts `myxb now`, `myxb next`, `myxb day`, and `myxb week` so common timetable queries no longer need the `schedule` group name.
- Added `yesterday`/`昨天`, `next week`/`下周`, bare `next`, and `next <weekday>`/`下周五`-style day selectors.

### Changes / Bug Fixes

- Timetable commands now skip the login round trip when the week cache is fresh, making cached queries instant and quiet; login only happens on cache miss or `--refresh`.
- Removed the hard profile requirement: timetable commands default to the `standard` profile (with a hint) instead of erroring out on first use.
- Malformed schedule items from other days no longer break a day's view; items are filtered by date before time parsing.
- The schedule cache file now prunes expired weeks on save instead of growing without bound.
- Replaced `os.Exit` in the schedule auth path with returned errors, merged the duplicated now/next handlers, and load the config once per run.

### Chores

- Added schedule service, cache-pruning, week-view, lazy-login, and rendering regression tests.
- Updated README and CLAUDE.md architecture notes for the timetable commands.

## v1.1.0 - 2026-05-04

### Features

- Added task category metadata to `-t/--tasks` output when the Xiaobao task detail endpoint provides evaluation project data.
- Added safe estimated subject-weight display for task rows when category scores match the average of scored tasks.
- Added a local task detail metadata cache at `~/.myxb/task_detail_cache.json` plus `--refresh-cache` to rebuild it.

### Changes / Bug Fixes

- Grouped detailed task rows under their evaluation categories instead of showing one flat task list.
- Preserved uncategorized or unmapped tasks in a separate section so task detail output remains complete.
- Extended formatted, plain, markdown, and table reports with task category and estimated weight columns where available.
- Updated Xiaobao API models and documentation for task detail `evaProjects`, task metadata, and evaluation project IDs.

### Chores

- Added regression coverage for task metadata enrichment and category-aware task rendering.
- Documented task cache behavior and the new `--refresh-cache` flag.

## v1.0.9 - 2026-05-04

### Changes / Bug Fixes

- Fixed GPA reports so critical score API failures no longer silently produce incomplete calculations.
- Fixed unreleased/all-null score projects so they are excluded as unavailable instead of being treated as 0/F.
- Fixed `-c -f json` output so saved-credential login progress no longer pollutes machine-readable JSON.
- Added visible warnings for skipped subjects that have no returned learning tasks.
- Added HTTP request timeouts and non-2xx status handling for clearer network failures.

### Chores

- Updated `github.com/ulikunitz/xz` to `v0.5.15` to resolve the `GO-2025-3922` vulnerability reported by `govulncheck`.
- Added GPA regression coverage for unreleased score handling.

## v1.0.8 - 2026-04-02

### Features

- Added `myxb schedule` with readable day views, current-class highlighting, and next-class lookup.
- Added `myxb schedule now`, `myxb schedule next`, and `myxb schedule day <date|weekday>` commands.
- Added saved schedule profiles so users can choose `highschool` or `standard` timetable interpretation.

### Changes / Bug Fixes

- Added local week-level schedule caching for `/api/Schedule/ListScheduleByParent`.
- Scoped timetable cache entries by account so different logins do not reuse each other's cached weeks.
- Mapped high-school periods 1-8 onto the provided bell schedule while leaving other schedule items on their API times.
- Required an explicit `schedule profile` selection before timetable commands can run.
- Preserved saved schedule profile preferences when logging out or logging in again.

### Chores

- Added schedule service, rendering, and cache coverage tests.
- Ignored local HAR captures so request archives are not committed accidentally.
- Updated README and API documentation for the new timetable flow.

## v1.0.7 - 2026-04-02

### Features

- Improved GPA accuracy by supporting configurable fractional course weights from `pkg/gpa/course_classification.json`.
- Added built-in fractional-credit entries for `C-Humanities`, `Spanish I`, `Spanish II`, and `Fine Art I` through `Fine Art IV`.

### Changes / Bug Fixes

- Fixed GPA mismatches caused by treating half-credit electives as full-credit courses.
- Wired previously unused `half_weighted`, `one_third_weighted`, and `two_third_weighted` course classifications into the calculation pipeline.
- Updated GPA output labels and docs to reflect fractional credit weights more clearly.

### Chores

- Added focused GPA regression tests for fractional-credit calculations.
- Reorganized `cmd/myxb` test files by concern while keeping them colocated with the CLI package, which is the standard Go testing layout.


## v1.0.6 - 2026-04-02

### Features

- Added formatted output modes for CLI reports with `-f/--formatted`
- Added readable `table` format as the default formatted output
- Added optional `plain`, `markdown`/`md`, and `json` output modes
- Added output export support with `-e/--export`
- Added non-interactive semester selection with `-s/--semester`
- Added support for selecting multiple semesters and full school years
- Added `-c/--clean` mode for quieter scripting and automation output

### Changes / Bug Fixes

- Refactored GPA report generation to separate data collection from rendering
- Kept the default human-readable output flow while enabling alternate machine-friendly formats
- Improved CLI argument normalization so bare `-f` and bare `-e` work as expected
- Preserved ASCII-safe subject/task display for problematic non-ASCII terminal rendering cases
- Added tests for format parsing, semester selection, and export path resolution

### Chores

- Updated README with new CLI examples and flag documentation
- Ignored local Go cache directories such as `.gocache/` and `.gomodcache/`
- Prepared release tag `v1.0.6`
