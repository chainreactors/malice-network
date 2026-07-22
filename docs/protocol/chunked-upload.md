# Chunked Upload

## Overview

`UploadChunk` lets browser clients upload a file in sequential chunks without holding the complete file in one request. The server writes accepted chunks to a private `.part` file and creates one normal implant upload task after the declared file size is reached.

An upload is scoped by operator identity, session ID, and `upload_id`. Metadata must remain unchanged across chunks. Replaying the latest accepted chunk is idempotent, but out-of-order chunks are rejected. Staged files are not resumed after a server restart.

## Usage

Send `UploadChunkRequest` messages with the same `upload_id`, file metadata, and `total_size`. Start at offset `0`, then use each response's `next_offset` for the next request. The final response includes `task`; earlier responses do not.

For a zero-byte file, send one request with `total_size: 0`, `offset: 0`, and empty `data`.

## Configuration

The `server.upload` section is optional. Omitting it uses these defaults:

| Field | Default | Meaning |
| --- | ---: | --- |
| `max_chunk_bytes` | 8 MiB | Maximum data carried by one request |
| `max_file_bytes` | 20 GiB | Maximum declared size of one file |
| `max_staging_bytes` | 20 GiB | Maximum total size reserved by active staged uploads |
| `min_free_disk_bytes` | 1 GiB | Disk space kept free in addition to active reservations |
| `max_active_per_session` | 4 | Maximum receiving uploads for one operator and session |
| `staging_ttl_seconds` | 21600 | Six-hour inactivity window before incomplete staging is removed |

Set `min_free_disk_bytes` to `0` only when deployment provides an equivalent external disk-space guard.

## Example

```yaml
server:
  upload:
    max_chunk_bytes: 8388608
    max_file_bytes: 21474836480
    max_staging_bytes: 21474836480
    min_free_disk_bytes: 1073741824
    max_active_per_session: 4
    staging_ttl_seconds: 21600
```

For a 12 MiB file with 8 MiB chunks, send offsets `0` and `8388608`. A successful first response returns `next_offset: 8388608`; the second returns `next_offset: 12582912` and the implant task.
