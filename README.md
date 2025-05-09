# Backup Service

A robust backup service written in Go that supports multiple storage backends (S3, Rsync, Google Drive) and provides scheduled backups.

## Features

- Multiple storage backends support:
  - Amazon S3
  - Rsync
  - Google Drive
- Scheduled backups using cron expressions
- Configurable backup retention
- File/folder exclusion patterns
- Docker support
- GitLab CI/CD integration

## Prerequisites

- Go 1.21 or later
- Docker (optional)
- GitLab account (for CI/CD)

## Installation

1. Clone the repository:
```bash
git clone https://gitlab.com/vuongtlt13/backup.git
cd backup
```

2. Install dependencies:
```bash
go mod download
```

3. Build the application:
```bash
go build -o backupdb
```

## Configuration

Create a `config.yaml` file in the project root:

```yaml
backups:
  - name: mysql_data
    source: ./data/mysql
    storage:
      - s3
      - rsync
    scheduler:
      enabled: true
      cron_expr: "0 2 * * *"  # Run at 2 AM daily
      max_backups: 7
    ignore:
      files:
        - "*.tmp"
        - "*.log"
      folders:
        - "temp"
        - "cache"

storage:
  s3:
    enabled: true
    kind: s3
    bucket: your-bucket
    region: your-region
    access_key: your-access-key
    secret_key: your-secret-key

  rsync:
    enabled: true
    kind: rsync
    host: backup-server
    path: /backups
    user: backup-user
```

## Running with Docker

1. Build the Docker image:
```bash
docker build -t backupdb .
```

2. Run the container:
```bash
docker run -d \
  -v /path/to/backups:/app/backups \
  -v /path/to/config.yaml:/app/config/config.yaml \
  -v /path/to/credentials.json:/app/config/credentials.json \
  backupdb
```

## Development

### Pre-commit Hooks

The project uses pre-commit hooks to ensure code quality. These hooks run tests and linting before each commit.

To set up pre-commit hooks:

1. Run the setup script:
```bash
./scripts/setup-pre-commit.sh
```

This will:
- Install pre-commit if not already installed
- Install golangci-lint if not already installed
- Set up the pre-commit hooks

The hooks will:
- Run tests (`go test -v ./...`)
- Run linter (`golangci-lint run`)

### Running Tests

```bash
go test -v ./...
```

### Running Linter

```bash
golangci-lint run
```

## CI/CD

The project uses GitLab CI/CD with the following stages:
- `lint`: Code quality checks
- `test`: Unit tests and coverage
- `dockerize`: Docker image build and push

## License

This project is licensed under the MIT License - see the LICENSE file for details. 