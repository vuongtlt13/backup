# Google Drive guide

Use this guide to configure Google Drive storage and verify that a real backup archive is uploaded.

There are two Google Drive auth modes:

- `service_account`: best for Google Workspace Shared Drive. Service accounts cannot upload into normal personal **My Drive** because they do not have personal Drive storage quota.
- `oauth_user`: best for personal **My Drive**. You log in once in a browser and the app stores a refresh token.

## 1. Enable Google Drive API

1. Open your [Google Cloud project](https://console.cloud.google.com/).
2. Enable the Google Drive API.

## 2. Option A: service account for Shared Drive

Use this mode when you have Google Workspace Shared Drive.

1. Open **IAM & Admin** > **Service Accounts**.
2. Click **Create service account**.
3. Enter a name such as `backup-uploader`.
4. Skip **Grant this service account access to project**. This backup tool does not need project IAM roles to upload into a shared Drive folder.
5. Click **Done**.
6. Open the service account.
7. Go to **Keys** > **Add key** > **Create new key**.
8. Select **JSON** and download the key file.
9. Add the service account email to a Shared Drive as **Contributor** or higher.
10. Create or choose a folder inside that Shared Drive.
11. Copy the folder ID from the browser URL.

Example config:

```yaml
storage:
  google_drive:
    enabled: true
    kind: google_drive
    auth_mode: service_account
    credentials_file: /absolute/path/to/service-account.json
    folder_id: your-google-drive-folder-id
```

## 3. Option B: OAuth user login for personal My Drive

Use this mode when you want to upload into your own personal Google Drive.

1. Open your Google Cloud project.
2. Open **APIs & Services** > **OAuth consent screen**.
3. Configure the consent screen for your app.
4. Open **APIs & Services** > **Credentials**.
5. Click **Create credentials** > **OAuth client ID**.
6. Select **Desktop app**.
7. Download the OAuth client JSON file.
8. Choose where the app should save the token file, for example `./google-drive-token.json`.

Example config:

```yaml
storage:
  google_drive:
    enabled: true
    kind: google_drive
    auth_mode: oauth_user
    client_secret_file: /absolute/path/to/oauth-client-secret.json
    token_file: /absolute/path/to/google-drive-token.json
    folder_id: your-google-drive-folder-id
```

Create the token once:

```bash
go run . --config config.google-drive-smoke.yaml -gdrive-auth-init google_drive
```

Open the printed URL in your browser, allow access, paste the authorization code, and the app saves `token_file`. After that, run backups normally:

```bash
go run . --config config.google-drive-smoke.yaml
```

Do not commit the OAuth client secret JSON or token JSON.

## 4. Prepare the Drive folder

Create or choose the destination folder and copy the folder ID from the browser URL.

For a URL like this:

```text
https://drive.google.com/drive/folders/1AbCdEfGhIjKlMnOpQrStUvWxYz
```

The folder ID is:

```text
1AbCdEfGhIjKlMnOpQrStUvWxYz
```

## 5. Create a local test source

From the project root, create a temporary source folder:

```bash
mkdir -p data/smoke/source
printf 'google drive smoke test\n' > data/smoke/source/test.txt
```

## 6. Create a smoke test config

Create `config.google-drive-smoke.yaml` in the project root.

For service account mode:

```yaml
backups:
  - name: google_drive_smoke
    type: folder
    source_path: ./data/smoke/source
    storage: [google_drive]
    scheduler:
      enabled: false
      cron_expr: ""
      max_backups: 3
    ignore:
      files: []
      folders: []

storage:
  google_drive:
    enabled: true
    kind: google_drive
    auth_mode: service_account
    credentials_file: /absolute/path/to/service-account.json
    folder_id: your-google-drive-folder-id
```

For OAuth user mode:

```yaml
backups:
  - name: google_drive_smoke
    type: folder
    source_path: ./data/smoke/source
    storage: [google_drive]
    scheduler:
      enabled: false
      cron_expr: ""
      max_backups: 3
    ignore:
      files: []
      folders: []

storage:
  google_drive:
    enabled: true
    kind: google_drive
    auth_mode: oauth_user
    client_secret_file: /absolute/path/to/oauth-client-secret.json
    token_file: /absolute/path/to/google-drive-token.json
    folder_id: your-google-drive-folder-id
```

## 7. Run the smoke test

If you use OAuth user mode, create the token first:

```bash
go run . --config config.google-drive-smoke.yaml -gdrive-auth-init google_drive
```

Then run the backup:

```bash
go run . --config config.google-drive-smoke.yaml
```

The app starts with an initial backup run and then waits for a shutdown signal. Stop it after the first upload succeeds:

```text
Ctrl+C
```

## 8. Verify the result

Check the local archive:

```bash
ls -lh backups/google_drive_smoke/*.tar.gz
```

Then open the Google Drive folder and confirm one file with a name like this exists:

```text
google_drive_smoke_YYYYMMDDHHMMSS_NNNNNNNNN.tar.gz
```

## 9. Remote retention

Google Drive supports the same `remote_retention` settings as S3-compatible storage. Configure it on the backup job:

```yaml
backups:
  - name: google_drive_smoke
    type: folder
    source_path: ./data/smoke/source
    storage: [google_drive]
    remote_retention:
      enabled: true
      max_per_day: 3
      max_per_month: 1
      max_per_year: 1
```

Retention only applies inside the configured `folder_id`. The app lists non-trashed files in that folder, filters names matching the current backup archive pattern, and deletes old matching files by Google Drive file ID.

Retention tiers:

- Current month: keep newest `max_per_day` files for each day.
- Previous months in the current year: keep newest `max_per_month` files for each month.
- Previous years: keep newest `max_per_year` files for each year.

A zero or omitted max value disables deletion for that tier.

The selected Google Drive auth mode must be able to list files in the folder and delete matching backup files. For service-account mode, this depends on the service account's permissions in the Shared Drive or folder.

## Docker notes

Mount the OAuth client secret and token file into the container. The token file must be on a persistent volume so it survives container restarts.

## Troubleshooting

### `failed to validate Google Drive credentials`

Check that:

- the Drive API is enabled,
- credential paths point to the correct JSON files,
- the destination folder is accessible to the selected auth mode,
- `folder_id` is the folder ID, not the full URL.

### `Service Accounts do not have storage quota`

The folder is accessible, but it is probably in a personal **My Drive**. Use `auth_mode: oauth_user`, or use a Google Workspace Shared Drive with service-account mode.

### `failed to load Google Drive OAuth token`

Run the one-time auth command:

```bash
go run . --config config.google-drive-smoke.yaml -gdrive-auth-init google_drive
```

### Upload succeeds but you cannot find the file

Make sure you are looking in the folder configured by `folder_id`. The app uploads directly into that folder.

### The app uploads more than once

The app runs one initial backup, then scheduled backups. For smoke tests, keep `scheduler.enabled: false` and stop the app after the first successful upload.
