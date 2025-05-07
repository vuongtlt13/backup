# BackupDB

A flexible backup solution that can backup any folder to multiple storage destinations (S3, Google Drive, Rsync).

## Features

- Backup any folder to multiple storage destinations
- Support for multiple storage types:
  - Amazon S3
  - Google Drive
  - Rsync
- Compressed backups with timestamps
- Configurable backup destinations per folder
- Simple YAML configuration

## Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/backupdb.git
cd backupdb
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

### Local Development Setup

1. Start the test databases using Docker Compose:
```bash
docker-compose up -d
```

This will start:
- MySQL 8.0 on port 3306
- PostgreSQL 16 on port 5432

2. Create a `config.yaml` file for local development:

```yaml
backups:
  # Database backups
  - name: "mysql_data"
    source_path: "./data/mysql"
    storages: ["s3", "rsync"]

  - name: "postgres_data"
    source_path: "./data/postgres"
    storages: ["google_drive", "s3"]

storage:
  s3:
    enabled: true
    bucket: "your-backup-bucket"
    region: "us-west-2"
    access_key: "your-access-key"
    secret_key: "your-secret-key"
    path: "backups/"

  google_drive:
    enabled: true
    credentials_file: "credentials.json"
    folder_id: "your-folder-id"

  rsync:
    enabled: true
    target_server: "backup-server"
    target_path: "/backup/data/"
    user: "backup-user"
    port: 22
```

3. Verify the databases are running:
```bash
# Check MySQL
docker exec -it mysql_db mysql -uroot -pyour_password -e "SHOW DATABASES;"

# Check PostgreSQL
docker exec -it postgres_db psql -U postgres -c "\l"
```

4. Run the backup:
```bash
./backupdb
```

5. Stop the databases when done:
```bash
docker-compose down
```

To remove all data including volumes:
```bash
docker-compose down -v
```

### Production Configuration

For production environments, create a `config.yaml` file with your backup configurations:

```yaml
backups:
  # Database backups
  - name: "mysql_data"
    source_path: "/var/lib/mysql"
    storages: ["s3", "rsync"]

  - name: "postgres_data"
    source_path: "/var/lib/postgresql/data"
    storages: ["google_drive", "s3"]

storage:
  s3:
    enabled: true
    bucket: "your-backup-bucket"
    region: "us-west-2"
    access_key: "your-access-key"
    secret_key: "your-secret-key"
    path: "backups/"

  google_drive:
    enabled: true
    credentials_file: "credentials.json"
    folder_id: "your-folder-id"

  rsync:
    enabled: true
    target_server: "backup-server"
    target_path: "/backup/data/"
    user: "backup-user"
    port: 22
```

### Database Backup Examples

#### MySQL Backup
To backup a MySQL database:
1. Stop the MySQL service to ensure data consistency:
```bash
sudo systemctl stop mysql
```

2. Add the MySQL data directory to your config:
```yaml
backups:
  - name: "mysql_data"
    source_path: "/var/lib/mysql"
    storages: ["s3", "rsync"]
```

3. Run the backup:
```bash
./backupdb
```

4. Restart MySQL:
```bash
sudo systemctl start mysql
```

#### PostgreSQL Backup
To backup a PostgreSQL database:
1. Stop the PostgreSQL service:
```bash
sudo systemctl stop postgresql
```

2. Add the PostgreSQL data directory to your config:
```yaml
backups:
  - name: "postgres_data"
    source_path: "/var/lib/postgresql/data"
    storages: ["google_drive", "s3"]
```

3. Run the backup:
```bash
./backupdb
```

4. Restart PostgreSQL:
```bash
sudo systemctl start postgresql
```

### Storage Configuration

#### S3 Configuration
1. Create an S3 bucket
2. Create an IAM user with S3 access
3. Configure the S3 section in `config.yaml`:
```yaml
storage:
  s3:
    enabled: true
    bucket: "your-backup-bucket"
    region: "us-west-2"
    access_key: "your-access-key"
    secret_key: "your-secret-key"
    path: "backups/"
```

#### Google Drive Configuration
1. Create a Google Cloud project
2. Enable the Google Drive API
3. Create a service account and download credentials
4. Configure the Google Drive section in `config.yaml`:
```yaml
storage:
  google_drive:
    enabled: true
    credentials_file: "credentials.json"
    folder_id: "your-folder-id"
```

#### Rsync Configuration
1. Set up SSH keys for passwordless authentication
2. Configure the Rsync section in `config.yaml`:
```yaml
storage:
  rsync:
    enabled: true
    target_server: "backup-server"
    target_path: "/backup/data/"
    user: "backup-user"
    port: 22
```

## Usage

Run the backup:
```bash
./backupdb
```

The backup process will:
1. Create a temporary directory for each backup
2. Copy the source folder
3. Compress it with a timestamp
4. Send it to the specified storage services
5. Clean up temporary files

## Backup File Structure

Backups are stored with the following naming convention:
```
backups/
  ├── mysql_data-20240314150405.tar.gz
  ├── postgres_data-20240314150405.tar.gz
  ├── application_logs-20240314150405.tar.gz
  └── documents-20240314150405.tar.gz
```

## Security Considerations

1. Database Backups:
   - Always stop the database service before backup
   - Ensure proper permissions on data directories
   - Use secure storage destinations

2. Storage Security:
   - Use IAM roles and policies for S3
   - Secure your Google Drive credentials
   - Use SSH keys for Rsync
   - Encrypt sensitive data in transit and at rest

3. Configuration:
   - Keep your `config.yaml` secure
   - Don't commit credentials to version control
   - Use environment variables for sensitive data

## Development

### Local Development Setup

1. Install Go 1.21 or later
2. Install dependencies:
```bash
go mod download
```

3. Run tests:
```bash
go test ./...
```

4. Build:
```bash
go build -o backupdb
```

### Project Structure

```
.
├── backup/
│   └── backup.go      # Backup service implementation
├── config/
│   └── config.go      # Configuration handling
├── storage/
│   └── storage.go     # Storage service implementation
├── config.yaml        # Configuration file
├── go.mod            # Go module file
└── README.md         # This file
```

## License

MIT License 