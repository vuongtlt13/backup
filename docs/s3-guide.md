# S3 guide

Use this guide to configure S3-compatible storage and verify uploads with AWS S3, Cloudflare R2, or a local MinIO server.

## AWS S3 config

Create or choose an S3 bucket and an IAM access key that can upload objects to that bucket.

Minimum IAM actions for the configured bucket:

- `s3:ListBucket` for startup bucket access validation.
- `s3:PutObject` for uploads.

Optional fields:

- `object_key_prefix`: stores archives under a folder-like prefix inside the bucket, such as `mysql` or `prod/mysql`.
- `skip_bucket_validation`: skips startup bucket validation and checks permissions only when uploading.

Example:

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
    bucket: your-bucket-name
    region: ap-southeast-1
    access_key_id: your-access-key-id
    secret_access_key: your-secret-access-key
    object_key_prefix: backups
```

With `object_key_prefix: backups`, an archive named `s3_smoke_20260508020000.tar.gz` is uploaded as `backups/s3_smoke_20260508020000.tar.gz`.

## Cloudflare R2 config

Cloudflare R2 is S3-compatible. Use the S3 provider with the R2 endpoint and path-style addressing.

Create an R2 bucket and an R2 API token, then configure:

```yaml
backups:
  - name: r2_backup
    type: folder
    source_path: ./data/smoke/source
    storage: [r2]
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
    object_key_prefix: mysql
    skip_bucket_validation: true
```

R2 bucket folders are object key prefixes. With `object_key_prefix: mysql`, the archive appears in the bucket as `mysql/r2_backup_YYYYMMDDHHMMSS_NNNNNNNNN.tar.gz`.

Use `skip_bucket_validation: true` when an R2 bucket-scoped token can upload objects but returns `403 Forbidden` for `HeadBucket` during startup validation. The token still needs permission to upload objects to the bucket.

Run the backup:

```bash
go run . --config config.r2.yaml
```

If `object_key_prefix` is empty, the uploaded object key is only the generated archive filename.

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

Check the bucket configured by `bucket`. If `object_key_prefix` is empty, the object key is the generated archive filename. If `object_key_prefix` is set, check that prefix folder inside the bucket.
