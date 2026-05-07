# S3 guide

Use this guide to configure S3 storage and verify uploads with either AWS S3 or a local MinIO server.

## AWS S3 config

Create or choose an S3 bucket and an IAM access key that can upload objects to that bucket.

Minimum IAM actions for the configured bucket:

- `s3:ListAllMyBuckets` for startup credential validation.
- `s3:PutObject` for uploads.

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
```

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

### Upload succeeds but you cannot find the file

Check the bucket configured by `bucket`. The object key is the generated archive filename.
