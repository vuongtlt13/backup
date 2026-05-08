# Raw database folder backup guide

Use this guide when you intentionally want `type: folder` backups of database data directories while the database keeps running.

This is a size/convenience tradeoff, not the safest database backup method. Logical dumps or coordinated filesystem snapshots are safer for production restores.

## What this backs up

For the Docker compose setup in this project, database files are stored in project-local folders:

```yaml
./data/mysql:/var/lib/mysql
./data/postgres:/var/lib/postgresql/data
```

A raw folder backup archives those host folders directly:

```yaml
source_path: ./data/mysql
source_path: ./data/postgres
```

## Important consistency warning

If MySQL or Postgres is running while files are copied, files can change during archive creation. The resulting backup may be inconsistent even if upload succeeds.

If you do not want to stop the DB, use these rules:

- Keep WAL/binlog/recovery files because they may be needed for crash recovery.
- Exclude only clearly temporary/runtime files.
- Restore-test backups regularly.
- Prefer daily or hourly schedules, not every minute.
- Keep retention small to control storage usage.

## MySQL raw folder backup while running

Safer running-DB config:

```yaml
backups:
  - name: mysql_data
    type: folder
    source_path: ./data/mysql
    storage: [r2]
    scheduler:
      enabled: true
      cron_expr: "0 2 * * *"
      max_backups: 3
    ignore:
      files:
        - "*.tmp"
        - "*.pid"
        - "*.sock"
        - "mysql.sock"
      folders:
        - "#innodb_temp"
        - "tmp"
```

Do not ignore these for running raw backups:

```text
ibdata*
ib_logfile*
undo*
binlog.*
relay-log.*
```

They can be large, but excluding them can make restores fail or remove point-in-time recovery data.

## Postgres raw folder backup while running

Safer running-DB config:

```yaml
backups:
  - name: postgres_data
    type: folder
    source_path: ./data/postgres
    storage: [r2]
    scheduler:
      enabled: true
      cron_expr: "30 2 * * *"
      max_backups: 3
    ignore:
      files:
        - "*.pid"
        - "*.tmp"
        - "postmaster.pid"
        - "postmaster.opts"
      folders:
        - "pg_stat_tmp"
```

Do not ignore these for running raw backups:

```text
base
global
pg_wal
pg_xact
pg_multixact
pg_subtrans
```

`pg_wal` can be large, but excluding it while Postgres is running can make the copied data directory unrecoverable.

## Reducing backup size safely

Use these first:

1. Reduce retention:

```yaml
max_backups: 3
```

2. Reduce frequency:

```yaml
cron_expr: "0 2 * * *"
```

3. Exclude only temp/runtime files shown above.

4. Clean old DB logs using database-native retention settings instead of excluding required recovery files.

5. Store backups in S3/R2 lifecycle-managed buckets if available.

Avoid these for running raw backups:

- excluding MySQL binlogs unless you do not need them,
- excluding Postgres `pg_wal`,
- backing up every minute,
- assuming a successful archive means the DB can restore.

## Restore testing

At least once, test restoring into a separate directory/container:

1. Download the `.tar.gz` backup.
2. Extract it into a clean data directory.
3. Start a new MySQL/Postgres container pointing at that extracted directory.
4. Check logs and query data.

If restore fails, switch to logical dumps or filesystem snapshots.

## Better alternatives

If the database must keep running and restore reliability matters, prefer:

- `type: mysql` / `type: postgres` logical dumps,
- LVM/Btrfs/ZFS/cloud disk snapshots,
- database-native backup tools.

Raw folder backups while DB is running are best treated as a convenience fallback, not the only production backup.
