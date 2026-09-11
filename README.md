# my-xb

A command-line tool for viewing grades and calculating GPA from the [Tsinglan School Xiaobao](https://tsinglanstudent.schoolis.cn/) system.

This project is a refactor project of [tls-xb](https://github.com/hey2022/tls-xb), written in golang. Claude Code is using in this project, AI generated code appears.

## Features

- Login to Xiaobao with automatic credential storage
- Calculate weighted and unweighted GPA
- View detailed subject scores and grades
- View the weekly timetable grid, the current class, and the next class (`myxb week`, `myxb now`, `myxb next`)
- Support readable formatted output modes and JSON export
- Support non-interactive semester selection, including multiple semesters / full school years
- Support clean output mode for scripting and automation
- Support exporting output to Desktop or a custom path
- Compare calculated GPA with official GPA
- Support for AP, A Level, and AS weighted courses
- Automatic elective and fractional-credit course detection

## Installation

```bash
go build -o myxb ./cmd/myxb
```

## Usage

### First Time Setup

Login and save your credentials:

```bash
./myxb login
```

You will be prompted to enter your username, password, and captcha (if required).
Credentials are securely stored in your home directory (~/.myxb/config.json).

### Calculate GPA

After logging in, simply run:

```bash
./myxb
```

This will:
- Authenticate using saved credentials
- Fetch your current semester grades
- Calculate weighted and unweighted GPA
- Display detailed subject information
- Compare with official GPA (if published)

Useful flags:

- `-t, --tasks` - include detailed task rows, with task category and estimated subject-weight contribution when safely inferable
- `-f, --formatted` - choose output format only; bare `-f` defaults to `table`
- `-c, --clean` - suppress banner, prompts, progress, and other human-oriented output; without `-s`, defaults to the current semester
- `-s, --semester` - select semester(s) without interactive prompts
- `-e, --export` - export output to Desktop by default, or to a directory / file path

Examples:

```bash
./myxb -t
./myxb -t --refresh-cache
./myxb -f
./myxb -f plain
./myxb -f markdown
./myxb -f json
./myxb -f -t -c
./myxb -f table -t -e
./myxb -f markdown -e ~/Desktop
./myxb -s current
./myxb -s 2025-1
./myxb -s 2025-2026
./myxb -s 2024-2025,2025-2026
```

Formatted output modes:

- `table` - default formatted mode; readable aligned text blocks
- `plain` - concise sectioned text
- `markdown` / `md` - markdown headings and bullets
- `json` - structured JSON output

Semester selector formats:

- `current` - current semester
- `all` - all available semesters
- `0`, `1`, ... - semester index from the interactive list
- `2025-1` - semester 1 in school year starting 2025
- `2025-2026` - all semesters in that school year
- `2025-2026-2` - semester 2 in that school year

Semester selection behavior:

- without `-s`, the CLI prompts you to choose a semester interactively
- with `-c` but without `-s`, the CLI skips prompts and uses `current`
- for automation or scripting, prefer passing both `-c` and `-s`
- `-f` only changes the report format; it does not suppress the banner or prompts

Export path behavior:

- an existing directory exports into that directory with an auto-generated filename
- an existing file path overwrites that file
- a new directory path should end with `/` or `\\`
- any other non-existent path is treated as a file path

### Commands

- `myxb` - Calculate GPA (default command)
- `myxb login` - Login and save credentials
- `myxb week` - Show this week's timetable grid (shortcut for `myxb schedule week`)
- `myxb now` - Show the class currently in session
- `myxb next` - Show the next class for today
- `myxb day friday` - Show the timetable for a weekday in the current week
- `myxb day 2026-04-03` - Show the timetable for a specific date
- `myxb attendance` - Show your attendance rate for the current school year
- `myxb schedule profile highschool` - Save the high-school bell schedule profile
- `myxb help` - Show help message

`myxb schedule` (aliases: `myxb s`, `myxb cal`, `myxb calendar`) groups the same views: `now`, `next`, `day`, `week`, and `profile`.

## Project Structure

```
myxb/
├── cmd/myxb/          # Main application
│   ├── main.go        # Entry point and command routing
│   ├── login.go       # Login handlers
│   ├── gpa.go         # GPA calculation command
│   └── colors.go      # Terminal color utilities
├── internal/
│   ├── api/           # API client methods
│   ├── auth/          # Password hashing
│   ├── client/        # HTTP client with cookie management
│   ├── config/        # Credential storage
│   ├── schedule/      # Timetable fetching, caching, and live schedule views
│   └── models/        # Data structures
└── pkg/gpa/           # GPA calculation logic
    ├── calculator.go  # Core GPA calculator
    └── score_mapping.go # Score to GPA mapping
```

## How It Works

### Authentication

1. Passwords are hashed using double MD5:
   - First hash: MD5(password)
   - Second hash: MD5(hash1 + timestamp)
2. First hash is stored locally for automatic login
3. Session cookies are managed automatically

### GPA Calculation

1. Fetches all subjects for current semester
2. Retrieves evaluation projects and scores for each subject
3. Adjusts proportions to account for incomplete assignments
4. Calculates subject total score (0-100)
5. Converts score to GPA using weighted/non-weighted mapping
6. Applies course credit weights (regular: 1.0, electives such as Spanish: 0.5, plus configured fractional-credit courses)
7. Computes weighted average GPA

If a subject's score details cannot be fetched (for example when a course's scores have not been synced to the mobile API), that subject falls back to its official semester score with a warning instead of aborting the whole report. Subjects without an official score are skipped and excluded from the GPA calculation.

See `GPA_CALCULATION.md` for detailed methodology.

### Weighted Courses

AP & A Level courses, and hard courses (linear algebra, modern physics and optics, multivariable calculus) use weighted scale (max 4.8).

All other courses use non-weighted scale (max 4.3).

## Configuration

Credentials are stored in:
- Windows: `%USERPROFILE%\.myxb\config.json`
- Linux/Mac: `~/.myxb/config.json`

Task detail metadata used by `--tasks` is cached locally in `~/.myxb/task_detail_cache.json`.
The cache stores task category metadata, not raw passwords, and avoids refetching every task detail on each run.
Use `--refresh-cache` with `--tasks` to rebuild it.

The config file contains:
```json
{
  "username": "your_username",
  "password_hash": "MD5_HASH_OF_PASSWORD",
  "schedule_profile": "highschool"
}
```

To reset credentials, delete this file or run `myxb login` again.

### Schedule / Timetable

The schedule commands read Xiaobao's `/api/Schedule/ListScheduleByParent` endpoint and cache the current school week locally. When the cache is still fresh, timetable queries skip the login round trip entirely and return instantly; `--refresh` forces a fresh fetch.

Useful commands:

```bash
./myxb week                 # this week's grid (today's column highlighted)
./myxb now                  # class in session right now
./myxb next                 # next class today
./myxb day friday           # a weekday in the current week
./myxb day 2026-04-03       # a specific date
./myxb week next            # next week's grid
./myxb day 下周五            # next Friday
./myxb schedule -d 周四      # same views via the schedule group
./myxb week --refresh
./myxb schedule profile highschool
```

Notes:

- `myxb schedule` (aliases `s`, `cal`, `calendar`) defaults to the week grid; `myxb day` shows one day's full table with `NOW` / `NEXT` markers and untruncated course names
- Week-grid cells show the course (and room); long names are truncated to keep the grid readable — use `myxb day <selector>` for full names
- Selectors accept `YYYY-MM-DD`, English weekdays, Chinese weekdays, `today`, `tomorrow`, `yesterday`, `next <weekday>`, `下周一`-style next-week names, and `next week` / `下周`
- Without a saved profile, timetable commands default to `standard` and print a hint; set one permanently with `myxb schedule profile standard` or `myxb schedule profile highschool`
- `myxb schedule profile highschool` adjusts periods 1-8 to the high-school bell schedule:
  - `P1` `08:00-08:40`
  - `P2` `08:40-09:20`
  - `P3` `10:10-10:50`
  - `P4` `11:00-11:40`
  - `P5` `11:50-12:30`
  - `B6` `13:25-14:05`
  - `B7` `14:15-14:55`
  - `B8` `15:05-15:45`
- `standard` keeps the raw begin/end times returned by Xiaobao
- `--refresh` bypasses the local cache when you want a fresh fetch

### Attendance

`myxb attendance` (alias `myxb att`) shows your attendance rate for the current school year:

```bash
./myxb attendance
```

Notes:

- The time range spans the whole school year (every semester of the current semester's year), matching the web client's attendance page
- The summary (total sessions, per-state counts, overall rate) comes from `/api/Attendance/GetAttendanceStatistic`; the per-subject table comes from `/api/Attendance/GetSubjectAttendanceStatistic` and is sorted by absent count descending
- Per-subject rate divides normal attendance by total sessions; attendance data is always fetched fresh with no local cache

### Building

```bash
go build -o myxb ./cmd/myxb
```

### Running

```bash
./myxb
```

## License

MIT License
