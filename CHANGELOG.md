# Changelog

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
