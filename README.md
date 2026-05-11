# Backup Service

A robust backup service written in Go that supports multiple storage backends (S3, Rsync, Google Drive) and provides scheduled backups.

## Features

- Multiple storage backends support:
  - Amazon S3 and S3-compatible providers like Cloudflare R2 ([setup guide](docs/s3-guide.md))
  - Rsync ([setup guide](docs/rsync-guide.md))
  - Google Drive ([setup guide](docs/google-drive-guide.md))
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
    type: folder
    source_path: ./data/mysql
    storage:
      - google_drive
    scheduler:
      enabled: true
      cron_expr: "0 2 * * *"
      max_backups: 7
    ignore:
      files:
        - "*.tmp"
        - "*.log"
      folders:
        - "temp"
        - "cache"

storage:
  google_drive:
    enabled: true
    kind: google_drive
    credentials_file: /app/config/service-account.json
    folder_id: your-google-drive-folder-id
```

### Google Drive setup

1. Enable the Google Drive API in your Google Cloud project.
2. Create a service account and download its JSON key file.
3. Open the destination folder in Google Drive and share it with the service account email from the JSON file.
4. Set `credentials_file` to the JSON key path and `folder_id` to the destination folder ID.
5. Add `google_drive` to the backup's `storage` list.

Run the service with your config:

```bash
go run . --config config.yaml
```

If you back up MySQL/Postgres data folders with `type: folder` while the database is running, read the [raw database folder backup guide](docs/raw-db-folder-backup-guide.md) first. Running raw backups can be inconsistent unless you use database dumps or filesystem snapshots.

For S3-compatible storage such as Cloudflare R2, use `object_key_prefix` to store archives under a folder-like path inside the bucket. If an R2 bucket-scoped token fails startup validation with `HeadBucket` and `403 Forbidden`, set `skip_bucket_validation: true` and let upload permissions be checked during `PutObject`.

### Remote retention

Remote retention is configured per backup job for S3-compatible and Google Drive storage. It runs after a successful upload, lists existing remote archives for the same backup name and location, sorts them by the timestamp in the generated archive filename, and deletes older matching archives.

```yaml
backups:
  - name: mysql_data
    type: folder
    source_path: ./data/mysql
    storage: [r2]
    object_key_prefix: mysql
    remote_retention:
      enabled: true
      max_per_day: 3
      period_days: 3
      max_per_period: 1
      max_per_month: 1
      max_per_year: 1
```

Retention rules:

- Latest backup day: keep newest `max_per_day` archives from the newest day found remotely.
- Older backups in the same month as the latest backup: if `period_days` is set, group them into `period_days` windows and keep newest `max_per_period` archives per window.
- Older months in the latest backup year: keep newest `max_per_month` archives for each month.
- Older years: keep newest `max_per_year` archives for each year.
- A zero or omitted max value disables deletion for that tier.

With the example above, the latest backup day keeps 3 archives, older days in the same month keep 1 archive per 3-day window, each older month keeps 1 archive, and each older year keeps 1 archive.

Provider and backup setup guides:

- [Raw database folder backup guide](docs/raw-db-folder-backup-guide.md)
- [S3 guide](docs/s3-guide.md)
- [Rsync guide](docs/rsync-guide.md)
- [Google Drive guide](docs/google-drive-guide.md)

## Running with Docker

Use the published image:

```bash
docker pull vuongtlt13/backup
```

Create a local config file from `config.yaml.example`, then mount it into the container at `/app/config/config.yaml`:

```bash
docker run -d \
  --name backupdb \
  --restart unless-stopped \
  -v $(pwd)/config.yaml:/app/config/config.yaml:ro \
  -v $(pwd)/backups:/app/backups \
  -v $(pwd)/data:/app/data:ro \
  vuongtlt13/backup
```

The container runs:

```bash
/app/backupdb --config /app/config/config.yaml
```

### Docker Compose

Create `docker-compose.yaml`:

```yaml
services:
  backupdb:
    image: vuongtlt13/backup:latest
    container_name: backupdb
    restart: unless-stopped
    environment:
      TZ: UTC
    volumes:
      - ./config.yaml:/app/config/config.yaml:ro
      - ./backups:/app/backups
      - ./data:/app/data:ro
      # Google Drive service account or OAuth files, if needed:
      # - ./service-account.json:/app/config/service-account.json:ro
      # - ./google-drive-oauth:/app/config/google-drive-oauth
      # SSH key for rsync or SSH database dumps, if needed:
      # - ~/.ssh/id_rsa:/app/config/id_rsa:ro
```

The same content is also available in `docker-compose.example.yaml`.

Run it:

```bash
docker compose up -d
```

View logs:

```bash
docker compose logs -f backupdb
```

Stop it:

```bash
docker compose down
```

### Mount paths

Use container paths in `config.yaml`, not host paths. For example, if you mount host `./data` to container `/app/data`, configure folder backups like this:

```yaml
backups:
  - name: mysql_data
    type: folder
    source_path: /app/data/mysql
```

Common mounts:

- `/app/config/config.yaml`: your backup configuration.
- `/app/backups`: local archive output and retention directory.
- `/app/data`: source folders to back up when using `type: folder`.
- `/app/config/*.json`: Google Drive service account, OAuth client secret, or token files.
- SSH keys can be mounted read-only if using rsync or SSH database dumps.

### Google Drive credentials example

```bash
docker run -d \
  --name backupdb \
  --restart unless-stopped \
  -v $(pwd)/config.yaml:/app/config/config.yaml:ro \
  -v $(pwd)/backups:/app/backups \
  -v $(pwd)/data:/app/data:ro \
  -v $(pwd)/service-account.json:/app/config/service-account.json:ro \
  vuongtlt13/backup
```

Then set the credential path in `config.yaml`:

```yaml
storage:
  google_drive:
    enabled: true
    kind: google_drive
    auth_mode: service_account
    credentials_file: /app/config/service-account.json
    folder_id: your-google-drive-folder-id
```

### OAuth token initialization in Docker

For Google Drive OAuth user mode, run the token init command interactively once:

```bash
docker run --rm -it \
  -v $(pwd)/config.yaml:/app/config/config.yaml:ro \
  -v $(pwd)/google-drive-oauth:/app/config/google-drive-oauth \
  vuongtlt13/backup \
  --config /app/config/config.yaml \
  --gdrive-auth-init google_drive_oauth
```

Mount the same token directory in the long-running container:

```bash
-v $(pwd)/google-drive-oauth:/app/config/google-drive-oauth
```

### Cloudflare R2 or S3 example

No extra credential files are needed for S3/R2 if credentials are stored in `config.yaml`:

```yaml
storage:
  r2:
    enabled: true
    kind: s3
    bucket: your-r2-bucket-name
    region: auto
    access_key_id: your-r2-access-key-id
    secret_access_key: your-r2-secret-access-key
    endpoint: https://your-cloudflare-account-id.r2.cloudflarestorage.com
    force_path_style: true
    object_key_prefix: backups
    skip_bucket_validation: true
```

### Rsync or SSH database dump example

Mount your SSH key read-only and reference the container path in `config.yaml`:

```bash
docker run -d \
  --name backupdb \
  --restart unless-stopped \
  -v $(pwd)/config.yaml:/app/config/config.yaml:ro \
  -v $(pwd)/backups:/app/backups \
  -v $HOME/.ssh/id_rsa:/app/config/id_rsa:ro \
  vuongtlt13/backup
```

```yaml
ssh:
  host: your-server.com
  port: 22
  user: root
  key_file: /app/config/id_rsa
```

### Logs and stopping

```bash
docker logs -f backupdb
docker stop backupdb
docker rm backupdb
```

### Build locally instead

```bash
docker build -t backupdb .
docker run -d --name backupdb -v $(pwd)/config.yaml:/app/config/config.yaml:ro backupdb
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