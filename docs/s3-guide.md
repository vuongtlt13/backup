# S3 guide

Use this guide to configure S3-compatible storage and verify uploads with AWS S3, Cloudflare R2, or a local MinIO server.

## AWS S3 config

Create or choose an S3 bucket and an IAM access key that can upload objects to that bucket.

Minimum IAM actions for the configured bucket:

- `s3:ListBucket` for startup bucket access validation and remote retention.
- `s3:PutObject` for uploads.
- `s3:DeleteObject` for remote retention.

Optional fields:

- Backup-level `object_key_prefix`: stores that backup job's archives under a folder-like prefix inside the bucket, such as `mysql` or `prod/mysql`.
- Storage-level `skip_bucket_validation`: skips startup bucket validation and checks upload permissions only when uploading.

Example:

```yaml
backups:
  - name: s3_smoke
    type: folder
    source_path: ./data/smoke/source
    storage: [s3]
    object_key_prefix: backups
    scheduler:
      enabled: false
      cron_expr: ""
      max_backups: 3
    ignore:
      files: []
      folders: []

storage:
  s3:
    enabled: true
    kind: s3
    bucket: your-bucket-name
    region: ap-southeast-1
    access_key_id: your-access-key-id
    secret_access_key: your-secret-access-key
```

With backup-level `object_key_prefix: backups`, an archive named `s3_smoke_20260508020000.tar.gz` is uploaded as `backups/s3_smoke_20260508020000.tar.gz`.

## Cloudflare R2 config

Cloudflare R2 is S3-compatible. Use the S3 provider with the R2 endpoint and path-style addressing.

Create an R2 bucket and an R2 API token, then configure:

```yaml
backups:
  - name: r2_backup
    type: folder
    source_path: ./data/smoke/source
    storage: [r2]
    object_key_prefix: mysql
    remote_retention:
      enabled: true
      max_per_day: 3
      period_days: 3
      max_per_period: 1
      max_per_month: 1
      max_per_year: 1
    scheduler:
      enabled: false
      cron_expr: ""
      max_backups: 3
    ignore:
      files: []
      folders: []

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
    skip_bucket_validation: true
```

R2 bucket folders are object key prefixes. Put `object_key_prefix` on each backup job so one `r2` storage can be reused by multiple jobs. With `object_key_prefix: mysql`, the archive appears in the bucket as `mysql/r2_backup_YYYYMMDDHHMMSS_NNNNNNNNN.tar.gz`.

Use `skip_bucket_validation: true` when an R2 bucket-scoped token can upload objects but returns `403 Forbidden` for `HeadBucket` during startup validation. The token still needs permission to upload objects to the bucket. If `remote_retention` is enabled, the token also needs list and delete permissions.

Run the backup:

```bash
go run . --config config.r2.yaml
```

If `object_key_prefix` is empty, the uploaded object key is only the generated archive filename.

## Remote retention

Remote retention is configured per backup job and applies to S3-compatible storage providers used by that job:

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

Retention uses the timestamp in generated archive filenames, not S3 upload time. It only deletes objects matching the current backup name and prefix.

Remote retention runs after a successful upload. The app lists objects under the effective `object_key_prefix`, keeps only names matching the current backup archive pattern, sorts them from newest to oldest by filename timestamp, then selects old matching objects for deletion.

Retention tiers:

- Latest backup day: keep newest `max_per_day` files from the newest day found in remote storage.
- Older backups in the same month as the latest backup: if `period_days` is set, group them into `period_days` windows and keep newest `max_per_period` files per window.
- Older months in the latest backup year: keep newest `max_per_month` files for each month.
- Older years: keep newest `max_per_year` files for each year.

Example with `max_per_day: 3`, `period_days: 3`, `max_per_period: 1`, `max_per_month: 1`, and `max_per_year: 1`:

```text
2026-05-08: keep 3 newest backups because this is the latest backup day
2026-05-07 to 2026-05-05: keep 1 newest backup for this 3-day window
2026-05-04 to 2026-05-02: keep 1 newest backup for this 3-day window
2026-04: keep 1 newest backup
2025:    keep 1 newest backup
```

If `period_days` or `max_per_period` is zero or omitted, older backups in the latest month use the monthly tier instead. A zero or omitted max value disables deletion for that tier.

## Local S3 with MinIO

Start MinIO and create the test bucket:

```bash
docker compose -f docker-compose.s3.yaml up -d
```

MinIO endpoints:

- S3 API: `http://localhost:9000`
- Console: `http://localhost:9001`
- Username: `minioadmin`
- Password: `minioadmin`
- Bucket: `backup-smoke`

Create a local smoke-test source folder:

```bash
mkdir -p data/smoke/source
printf 's3 smoke test\n' > data/smoke/source/test.txt
```

Create `config.s3-smoke.yaml`:

```yaml
backups:
  - name: s3_smoke
    type: folder
    source_path: ./data/smoke/source
    storage: [s3]
    scheduler:
      enabled: false
      cron_expr: ""
      max_backups: 3
    ignore:
      files: []
      folders: []

storage:
  s3:
    enabled: true
    kind: s3
    bucket: backup-smoke
    region: us-east-1
    access_key_id: minioadmin
    secret_access_key: minioadmin
    endpoint: http://localhost:9000
    force_path_style: true
```

Run the backup:

```bash
go run . --config config.s3-smoke.yaml
```

The app starts with one initial backup and then waits for a shutdown signal. Stop it after the first upload succeeds:

```text
Ctrl+C
```

## Verify upload

Open the MinIO console:

```text
http://localhost:9001
```

Log in with `minioadmin` / `minioadmin`, open the `backup-smoke` bucket, and confirm one file with a name like this exists:

```text
s3_smoke_YYYYMMDDHHMMSS_NNNNNNNNN.tar.gz
```

You can also use the MinIO client from Docker:

```bash
docker compose -f docker-compose.s3.yaml run --rm minio-init mc ls local/backup-smoke
```

## Cleanup

Stop MinIO:

```bash
docker compose -f docker-compose.s3.yaml down
```

Remove local MinIO data only if you no longer need it:

```bash
rm -rf data/minio
```

## Troubleshooting

### `failed to validate S3 credentials`

Check that:

- MinIO is running for local tests,
- `endpoint` is `http://localhost:9000`,
- `force_path_style` is `true` for MinIO,
- bucket name is `backup-smoke`,
- access key and secret are both `minioadmin` for the local compose setup.

For Cloudflare R2, if upload permissions are correct but startup fails with `HeadBucket` and `403 Forbidden`, set `skip_bucket_validation: true`.

### Upload succeeds but you cannot find the file

Check the bucket configured by `bucket`. If backup-level `object_key_prefix` is empty, the object key is the generated archive filename. If `object_key_prefix` is set, check that prefix folder inside the bucket.
