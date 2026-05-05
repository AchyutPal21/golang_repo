# Capstone G — Scaling Discussion

## Why multipart uploads matter

Single-shot uploads fail on poor connections — the entire transfer must restart on error. Multipart uploads:
- Retry only the failed part, not the whole file
- Upload parts in parallel (5 connections × 5MB parts = 5× throughput)
- Resume across browser refreshes or network drops

S3 requires parts ≥ 5MB (except the last). This prevents excessive part metadata overhead.

## Streaming vs buffering

**Never buffer the entire file in memory on the server:**

```go
// BAD — loads 50MB into RAM per concurrent upload
data, _ := io.ReadAll(r.Body)

// GOOD — stream directly to storage
io.Copy(storageWriter, r.Body)
```

With 1000 concurrent uploads of 50MB each, buffering requires 50GB of RAM. Streaming keeps memory at O(buffer_size) per connection (typically 32KB).

## Content-type detection

The `Content-Type` header is user-controlled and untrustworthy. Detect the real MIME type from the file's magic bytes:

```go
// Read first 512 bytes for MIME sniffing
buf := make([]byte, 512)
n, _ := r.Body.Read(buf)
contentType := http.DetectContentType(buf[:n])
// Then re-combine buf+rest for actual storage
```

## Virus scanning integration

Real virus scanning (ClamAV, VirusTotal) adds 200ms–2s latency. Strategies:

| Approach | Latency | Risk |
|----------|---------|------|
| Sync pre-store scan | +2s | Blocks upload |
| Async post-store scan | 0ms | File briefly accessible |
| Quarantine queue | 0ms | File held until scanned |

**Recommended:** Quarantine. Store to a `quarantine/` prefix, scan asynchronously, move to `public/` on pass, delete on fail. Use S3 bucket events or a Postgres outbox to trigger the scanner.

## S3-compatible storage

Replace `memoryStorage` with any S3-compatible backend via the same `StorageBackend` interface:

```go
type s3Storage struct{ client *s3.Client; bucket string }

func (s *s3Storage) Put(key, ct string, r io.Reader) (Object, error) {
    _, err := s.client.PutObject(ctx, &s3.PutObjectInput{
        Bucket:      &s.bucket,
        Key:         &key,
        ContentType: &ct,
        Body:        r,
    })
    ...
}
```

For resumable uploads, use S3's native `CreateMultipartUpload` / `UploadPart` / `CompleteMultipartUpload` — the same API we simulated here.

## Presigned URLs (recommended pattern)

Instead of proxying file data through your server, issue presigned URLs:

```
Client → POST /upload/sign → returns presigned S3 PUT URL (15 min expiry)
Client → PUT directly to S3 (bypasses your server)
Client → POST /upload/confirm → record metadata in DB
```

This removes your servers from the data path entirely. Throughput scales with S3, not your pod count.

## Kubernetes deployment

```yaml
upload-service:
  replicas: 5
  resources: {cpu: "500m", memory: "256Mi"}  # low memory because streaming
  config:
    MAX_UPLOAD_SIZE: "52428800"   # 50MB
    UPLOAD_TIMEOUT:  "300s"       # 5 min for large files
    PART_MIN_SIZE:   "5242880"    # 5MB minimum per part
```

Use a `readinessProbe` that checks storage connectivity — if S3/MinIO is unreachable, stop accepting traffic.
