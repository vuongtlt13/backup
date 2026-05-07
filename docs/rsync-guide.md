# Rsync guide

Use this guide to configure rsync storage over SSH.

The rsync provider sends the generated `.tar.gz` archive file to a remote server with `rsync` over SSH.

## Requirements

On the machine running this backup app:

- `rsync`
- `ssh`
- SSH access to the destination server

On the destination server:

- `rsync`
- a destination directory for backups

## 1. Prepare SSH access

Create or choose a remote backup directory:

```bash
ssh backup-user@backup-server 'mkdir -p /backups'
```

Verify SSH login works without an interactive password prompt:

```bash
ssh backup-user@backup-server 'echo ok'
```

If you use an SSH key, load it into your SSH agent or configure it in your SSH config before running the backup app.

## 2. Configure rsync storage

Example:

```yaml
backups:
  - name: rsync_smoke
    type: folder
    source_path: ./data/smoke/source
    storage: [rsync]
    scheduler:
      enabled: false
      cron_expr: ""
      max_backups: 3
    ignore:
      files: []
      folders: []

storage:
  rsync:
    enabled: true
    kind: rsync
    server: backup-server
    username: backup-user
    path: /backups
    port: 22
```

The app runs a command equivalent to:

```bash
rsync -avzr -e "ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -p 22" --delete --progress <archive-file> backup-user@backup-server:/backups
```

## 3. Create a local smoke-test source

From the project root:

```bash
mkdir -p data/smoke/source
printf 'rsync smoke test\n' > data/smoke/source/test.txt
```

## 4. Run the backup

```bash
go run . --config config.rsync-smoke.yaml
```

The app starts with one initial backup and then waits for a shutdown signal. Stop it after the first upload succeeds:

```text
Ctrl+C
```

## 5. Verify the result

On the destination server:

```bash
ssh backup-user@backup-server 'ls -lh /backups'
```

Confirm one file with a name like this exists:

```text
rsync_smoke_YYYYMMDDHHMMSS_NNNNNNNNN.tar.gz
```

## Troubleshooting

### `failed to send file via rsync`

Check that:

- SSH login works without an interactive prompt,
- the configured `server`, `username`, `path`, and `port` are correct,
- the remote directory exists and is writable by the SSH user,
- `rsync` is installed locally and on the remote server.

### Upload succeeds but you cannot find the file

Check the configured remote `path`. The app sends the generated archive file directly to that path.
