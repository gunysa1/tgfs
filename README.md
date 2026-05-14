# tgfs

A FUSE filesystem backed by Telegram channels. Store large video libraries (movies, TV shows) in Telegram and mount them read-only for Plex or other media servers.

Files are split into ~45 MB chunks to stay within the Telegram Bot API's 50 MB per-upload limit. Metadata (paths, chunk locations) lives in a local SQLite database. A daemon handles the mount; a CLI manages everything else.

Uploads run in parallel: configurable file-level and chunk-level workers, with 429 rate-limit retries honoring Telegram's `RetryAfter`.

## Architecture

```
Plex (Docker)
    │  reads files via FUSE
    ▼
/mnt/tgfs  ←── tgfsd (FUSE daemon)
                    │  fetches chunks on read
                    ▼
             Telegram Bot API
                    │  stores chunks as messages
                    ▼
          Telegram channel(s)
```

## Requirements

- Linux with FUSE support (`apt install fuse`)
- Go 1.22+
- A Telegram bot token (create one via [@BotFather](https://t.me/BotFather))
- One or more Telegram channels with the bot added as admin

## Installation

```bash
git clone https://github.com/gunysa1/tgfs
cd tgfs
sudo make install
```

This builds both binaries and installs them to `/usr/local/bin`, copies the systemd unit, and reloads systemd.

If `go` is not in your `PATH` (e.g. installed to a custom location), pass it explicitly:

```bash
sudo make install GO=/usr/local/go/bin/go
```

## Configuration

Copy the example config and fill in your bot token:

```bash
sudo mkdir -p /etc/tgfs
sudo cp config.yaml.example /etc/tgfs/config.yaml
sudo $EDITOR /etc/tgfs/config.yaml
```

```yaml
telegram:
  bot_token: "YOUR_BOT_TOKEN_HERE"

mount:
  path: /mnt/tgfs

db:
  path: /var/lib/tgfs/tgfs.db

cache:
  max_size_gb: 2      # in-memory read cache

chunk:
  size_mb: 45         # Bot API caps uploads at 50 MB per request

migrate:
  file_workers: 4     # files uploaded in parallel
  chunk_workers: 4    # chunks per file uploaded in parallel
```

With the defaults that's up to 16 concurrent uploads. Raise them if you don't hit 429s; lower them if you do. The daemon retries 429s automatically using `RetryAfter`.

## Usage

### Start the daemon

```bash
sudo systemctl start tgfs
sudo systemctl enable tgfs   # start on boot
```

### Manage channels

```bash
# Add a Telegram channel (get the ID from the channel URL or a bot like @userinfobot).
# Note: channel IDs start with '-', which cobra parses as a flag — use '--' to terminate flags:
sudo tgfs channel add -- -1001234567890 movies

# List configured channels
tgfs channel ls

# Remove a channel
tgfs channel rm movies
```

### Migrate an existing library

```bash
# Preview what would be uploaded (no writes)
tgfs migrate /path/to/media --dry-run

# Run the migration
tgfs migrate /path/to/media
```

Migration is idempotent — already-uploaded files are skipped. Per-file upload errors are logged and the run continues with the remaining files.

### Manage files

```bash
# List files at a virtual path
tgfs ls /Movies

# Upload a single file
tgfs upload /local/path/file.mkv /Movies/file.mkv

# Delete a file (removes Telegram messages and DB record)
tgfs delete /Movies/file.mkv

# Move/rename a file
tgfs mv /Movies/old-name.mkv /Movies/new-name.mkv
```

### Other commands

```bash
tgfs status          # show mount path and cache usage
tgfs cache status    # show cache stats in bytes
tgfs cache clear     # evict all cached chunks
tgfs stop            # gracefully unmount and stop the daemon
```

## Plex setup

Point your Plex library at `/mnt/tgfs` (or a subdirectory like `/mnt/tgfs/Movies`). The filesystem is read-only; all writes go through the CLI.

If Plex runs in Docker, bind-mount the FUSE mount into the container:

```yaml
# docker-compose.yml
volumes:
  - /mnt/tgfs:/data/tgfs:ro
```

## Development

```bash
make build    # build binaries to ./bin/
make test     # run all tests
make clean    # remove build artifacts
```

## License

MIT
