# tgfs — Telegram-backed FUSE Storage for Plex

**Date:** 2026-05-13  
**Status:** Approved

---

## Overview

`tgfs` is a Go daemon that mounts a virtual filesystem (via FUSE) backed by Telegram channel storage. Large video files are split into ≤1.9 GB chunks, uploaded to Telegram channels via the Bot API, and reassembled transparently on read. Plex Media Server accesses the mount via a Docker bind mount, replacing an existing Google Drive / rclone / mergerfs stack.

**Constraints:**
- Telegram Bot API only — no phone/MTProto user session
- 2 GB per-message hard limit → solved by chunking at 1.9 GB
- Read-only FUSE mount from Plex's perspective
- Deployment: single Debian host, Plex in Docker

---

## Architecture

```
Debian Host
├── tgfsd (systemd service)
│   ├── FUSE layer        → /mnt/tgfs
│   ├── Telegram Bot client (upload / download)
│   ├── SQLite DB         → /var/lib/tgfs/tgfs.db
│   ├── Chunker           → split on upload, reassemble on read
│   └── IPC server        → /run/tgfs/tgfs.sock
│
├── tgfs (CLI)
│   └── connects to daemon via Unix socket
│
└── Plex Docker
    └── bind mount: /mnt/tgfs → /data/media
```

One binary per role. The daemon owns all state; the CLI is a thin IPC client.

---

## Data Model

```sql
CREATE TABLE channels (
    id          INTEGER PRIMARY KEY,
    telegram_id INTEGER NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE files (
    id          INTEGER PRIMARY KEY,
    path        TEXT NOT NULL UNIQUE,   -- e.g. /Movies/Inception (2010) {imdb-tt1375666}/Inception.mkv
    name        TEXT NOT NULL,
    size        INTEGER NOT NULL,       -- total reassembled size in bytes
    mime_type   TEXT,
    is_dir      INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE chunks (
    id          INTEGER PRIMARY KEY,
    file_id     INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    chunk_index INTEGER NOT NULL,       -- 0-based
    message_id  INTEGER NOT NULL,       -- Telegram message ID
    channel_id  INTEGER NOT NULL REFERENCES channels(id),
    size        INTEGER NOT NULL,
    UNIQUE(file_id, chunk_index)
);
```

Directories are `files` rows with `is_dir=1` and no chunks. Virtual paths are stored directly — directory listing is a `WHERE path LIKE '/Movies/%'` query filtered to depth+1.

---

## Components

### `internal/bot`
Telegram Bot API client. Responsibilities:
- Upload a file chunk (multipart, ≤1.9 GB) to a configured channel, return `message_id`
- Download a chunk by `message_id`, return an `io.Reader`
- Delete a message by `message_id`

### `internal/chunker`
- **Upload:** splits an `io.Reader` into sequential 1.9 GB chunks, passes each to `bot.Upload`
- **Read:** given a byte range `[offset, offset+size)`, determines which chunk indices are needed, fetches them, returns the correct slice

### `internal/db`
SQLite access layer (WAL mode). CRUD for `files`, `chunks`, `channels`. Path-based lookups for FUSE readdir/getattr.

### `internal/fs`
FUSE filesystem using `bazil.org/fuse`. Implements:
- `Readdir` — list virtual directory children from DB
- `Getattr` — file size = sum of chunk sizes, timestamps from DB
- `Read(offset, size)` — delegates to chunker, uses LRU cache

**Chunk cache:** In-memory LRU, default 2 GB, keyed by `(file_id, chunk_index)`. Covers Plex's sequential read pattern; cross-chunk seeks trigger a new download.

### `internal/ipc`
Unix socket (`/run/tgfs/tgfs.sock`) with a simple JSON-RPC protocol. Daemon listens; CLI connects and sends commands.

---

## CLI Commands

```
# Daemon control
tgfs start
tgfs stop
tgfs status

# Library migration (day-one operation)
tgfs migrate <path> [--dry-run]   # walks dir tree, uploads all video files, idempotent

# File management
tgfs upload <local-file> <virtual-path>
tgfs delete <virtual-path>
tgfs ls [virtual-path]
tgfs mv <src> <dst>               # rename in DB only, no re-upload

# Channel management
tgfs channel add <telegram-channel-id> <name>
tgfs channel ls
tgfs channel rm <name>

# Cache
tgfs cache status
tgfs cache clear
```

`tgfs migrate` is idempotent: it skips files already present in the DB by path. Progress bar per file. Preserves exact Plex-compatible directory structure from source.

---

## Configuration (`config.yaml`)

```yaml
telegram:
  bot_token: ""
  channels:
    - id: -1001234567890
      name: movies
    - id: -1009876543210
      name: tv

mount:
  path: /mnt/tgfs

db:
  path: /var/lib/tgfs/tgfs.db

cache:
  max_size_gb: 2

chunk:
  size_mb: 1900
```

---

## Deployment

**systemd unit** (`/etc/systemd/system/tgfs.service`):
```ini
[Unit]
Description=tgfs Telegram FUSE daemon
After=network.target

[Service]
ExecStart=/usr/local/bin/tgfsd --config /etc/tgfs/config.yaml
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

**Plex Docker bind mount** (`docker-compose.yml` excerpt):
```yaml
volumes:
  - /mnt/tgfs:/data/media:ro
```

**Migration workflow:**
1. Deploy `tgfsd`, configure bot token + channels
2. Run `tgfs migrate /home/guny/data/merged`
3. Verify with `tgfs ls /Movies` and `tgfs ls /TV`
4. Update Plex library path to `/data/media`
5. Disable rclone/mergerfs mounts

---

## Known Constraints

- **2 GB file limit per Telegram message** — mitigated by 1.9 GB chunking; files of any size are supported
- **Telegram Bot API rate limits** — uploads throttled; large migrations may take hours depending on library size
- **No direct write from FUSE** — all writes go through CLI; Plex cannot add/edit files directly
- **Cache is in-memory only** — restarting the daemon clears the chunk cache; first-play after restart re-downloads
