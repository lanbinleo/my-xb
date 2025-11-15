# my-xb

A command-line tool for viewing grades and calculating GPA from the [Tsinglan School Xiaobao](https://tsinglanstudent.schoolis.cn/) system.

This project is a refactor project of [tls-xb](https://github.com/hey2022/tls-xb), written in golang. Claude Code is using in this project, AI generated code appears.

## Features

- Login to Xiaobao with automatic credential storage
- Calculate weighted and unweighted GPA
- View detailed subject scores and grades
- Compare calculated GPA with official GPA
- Support for AP, A Level, and AS weighted courses
- Automatic elective course detection

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

### Commands

- `myxb` - Calculate GPA (default command)
- `myxb login` - Login and save credentials
- `myxb help` - Show help message

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
6. Applies course weights (electives: 0.5, regular: 1.0)
7. Computes weighted average GPA

See `GPA_CALCULATION.md` for detailed methodology.

### Weighted Courses

AP & A Level courses, and hard courses (linear algebra, modern physics and optics, multivariable calculus) use weighted scale (max 4.8).

All other courses use non-weighted scale (max 4.3).

## Configuration

Credentials are stored in:
- Windows: `%USERPROFILE%\.myxb\config.json`
- Linux/Mac: `~/.myxb/config.json`

The config file contains:
```json
{
  "username": "your_username",
  "password_hash": "MD5_HASH_OF_PASSWORD"
}
```

To reset credentials, delete this file or run `myxb login` again.

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
