# go_drive_backup

Docker image to backup dicrectories with `rclone` to Google Drive 

## Build

```sh
docker build -t ghcr.io/reeywhaar/go_drive_backup .
```

## Run

### Authorize app

```sh
docker run --rm -v ./credentials:/app/credentials -i ghcr.io/reeywhaar/go_drive_backup auth
```

### Check auth

```sh
docker run --rm -v ./credentials:/app/credentials ghcr.io/reeywhaar/go_drive_backup check-auth
```

### Backup

```sh
docker run --rm -v ./credentials:/app/credentials -v ./private/test_backup:/app/private/test_backup:ro --env BACKUP_TARGETS="private/test_backup/:backup/test_backup/" ghcr.io/reeywhaar/go_drive_backup backup
```