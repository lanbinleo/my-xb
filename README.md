# MyXB - Xiaobao Grade Viewer

A command-line tool for viewing grades and calculating GPA from the school's Xiaobao system.

## Features

- Login to Xiaobao with automatic cookie management
- View semester information
- List subjects and courses
- Calculate GPA (coming soon)

## Installation

```bash
go build -o myxb ./cmd/myxb
```

## Usage

### Interactive Mode

Run the program without arguments to enter interactive mode:

```bash
./myxb
```

You will be prompted to enter your username, password, and captcha (if required).

### Command-Line Mode

Provide credentials as command-line arguments:

```bash
./myxb -user YOUR_USERNAME -pass YOUR_PASSWORD
```

### Options

- `-user` - Username for login
- `-pass` - Password for login

## Project Structure

```
myxb/
├── cmd/myxb/          # Main application entry point
├── internal/
│   ├── api/           # API client methods
│   ├── auth/          # Authentication logic
│   ├── client/        # HTTP client with cookie management
│   └── models/        # Data structures
└── pkg/gpa/           # GPA calculation (coming soon)
```

## Authentication

The program uses double MD5 hashing for password security:

1. First hash: MD5(password)
2. Second hash: MD5(hash1 + timestamp)

All requests after login automatically include session cookies.

## API Documentation

See `API_DOCUMENTATION.md` for detailed API endpoint information.

See `GPA_CALCULATION.md` for GPA calculation methodology.

## Development

### Prerequisites

- Go 1.20 or higher

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
