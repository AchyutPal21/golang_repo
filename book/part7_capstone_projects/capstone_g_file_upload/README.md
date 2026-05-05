# Capstone G — File Upload Service

A production-grade file upload service supporting multipart uploads, resumable uploads, content-type validation, virus-scan hooks, and S3-compatible storage — all without any external SDK dependencies.

## What you build

- `POST /upload` — single multipart upload (up to 50MB)
- `POST /upload/init` — initialise a resumable upload, returns `uploadID`
- `PUT /upload/:uploadID/part/:n` — upload a part (5MB minimum per part)
- `POST /upload/:uploadID/complete` — assemble parts into final object
- `DELETE /upload/:uploadID` — abort and clean up parts
- `GET /files/:key` — serve/download a stored file

## Architecture

```
Client
  │
  ▼
Upload Handler
  ├── ContentValidator  (MIME type check, size limit)
  ├── VirusScanHook     (pluggable pre-store check)
  ├── StorageBackend    (interface: local / S3-compat)
  │     ├── MemoryStorage   (dev/test)
  │     └── FileStorage     (local disk, prod-ish)
  ├── MultipartAssembler (orders parts, merges byte slices)
  └── UploadTracker     (uploadID → parts state)
```

## Key components

| Component | Pattern | Chapter ref |
|-----------|---------|-------------|
| Storage interface | Repository pattern | Ch 34 |
| Multipart assembly | Ordered part map | Ch 18 |
| Content validation | Input validation | Ch 62 |
| Resumable upload state | In-memory tracker with expiry | Ch 71 |
| Streaming write | io.Reader pipeline | Ch 38 |
| Graceful cleanup | context + TTL expiry | Ch 47 |

## Running

```bash
go run ./book/part7_capstone_projects/capstone_g_file_upload
```

## What this capstone tests

- Can you implement resumable uploads without an SDK?
- Can you stream large uploads without loading them fully into memory?
- Can you enforce content-type and size limits at the boundary?
- Can you assemble out-of-order parts correctly?
