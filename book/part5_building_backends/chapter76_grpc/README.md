# Chapter 76 — gRPC

## What you'll learn

How gRPC works: protobuf contracts, four call types (unary, server streaming, client streaming, bidirectional), interceptors, status codes, metadata, deadlines, and retry policies. Examples use in-process simulation so no external tooling is required; see the real gRPC section for connecting to a live server.

## Key concepts

| Concept | Description |
|---|---|
| Protobuf | Binary serialization format; smaller and faster than JSON |
| `.proto` file | Contract-first schema; `protoc` generates Go types and stubs |
| Unary RPC | Request → Response (like HTTP POST) |
| Server streaming | Request → stream of responses (like SSE) |
| Client streaming | Stream of requests → single response (like batch upload) |
| Bidirectional streaming | Concurrent request/response streams (like WebSocket) |
| Status code | Structured error with `Code` + human message (not HTTP status) |
| Interceptor | Middleware for unary and streaming RPCs |
| Metadata | Key-value headers attached to a call (auth, tracing, correlation IDs) |
| Deadline | `context.WithDeadline` propagated via HTTP/2; always set one |
| Retry policy | Automatic retry for `UNAVAILABLE` / `RESOURCE_EXHAUSTED` errors |

## Files

| File | Topic |
|---|---|
| `examples/01_grpc_basics/main.go` | Messages, status codes, unary, streaming, interceptors |
| `examples/02_grpc_patterns/main.go` | Client streaming, bidi streaming, deadlines, retry, connection pool |
| `exercises/01_product_service/main.go` | CRUD service, search streaming, stock watch, tracing interceptor |

## Proto → Go workflow (real gRPC)

```proto
// product.proto
service ProductService {
  rpc GetProduct(GetProductRequest) returns (Product);
  rpc ListProducts(ListProductsRequest) returns (stream Product);
  rpc BulkUpdate(stream InventoryUpdate) returns (BulkUpdateResult);
  rpc WatchPrices(stream PriceRequest) returns (stream PriceUpdate);
}
```

```bash
# Install tools
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate
protoc --go_out=. --go-grpc_out=. product.proto
```

```go
// Add dependencies
go get google.golang.org/grpc
go get google.golang.org/protobuf
```

## Service implementation pattern

```go
// Implement the generated interface.
type productServer struct {
    pb.UnimplementedProductServiceServer
    store *Store
}

func (s *productServer) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.Product, error) {
    p, err := s.store.Find(req.ProductId)
    if err != nil {
        return nil, status.Errorf(codes.NotFound, "product %q not found", req.ProductId)
    }
    return toProto(p), nil
}
```

## Server streaming pattern

```go
func (s *productServer) ListProducts(req *pb.ListRequest, stream pb.ProductService_ListProductsServer) error {
    for _, p := range s.store.All() {
        if err := stream.Send(toProto(p)); err != nil {
            return err // client disconnected
        }
    }
    return nil
}
```

## Status codes (common subset)

| Code | Value | Use when |
|---|---|---|
| OK | 0 | Success |
| InvalidArgument | 3 | Bad request field |
| NotFound | 5 | Entity does not exist |
| AlreadyExists | 6 | Conflict on create |
| PermissionDenied | 7 | Auth check failed |
| ResourceExhausted | 8 | Rate limit hit |
| Internal | 13 | Unexpected server error |
| Unavailable | 14 | Server overloaded — safe to retry |
| DeadlineExceeded | 4 | Context expired |

```go
import "google.golang.org/grpc/codes"
import "google.golang.org/grpc/status"

// Return a status error.
return nil, status.Errorf(codes.NotFound, "user %d not found", id)

// Inspect an incoming error.
if st, ok := status.FromError(err); ok {
    log.Printf("code=%s msg=%s", st.Code(), st.Message())
}
```

## Interceptors

```go
// Unary server interceptor.
func loggingInterceptor(
    ctx context.Context, req interface{},
    info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
) (interface{}, error) {
    start := time.Now()
    resp, err := handler(ctx, req)
    log.Printf("%s dur=%s err=%v", info.FullMethod, time.Since(start), err)
    return resp, err
}

s := grpc.NewServer(
    grpc.ChainUnaryInterceptor(authInterceptor, loggingInterceptor),
)
```

## Deadlines (always set one)

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
resp, err := client.GetProduct(ctx, req)
// If the deadline expires, err code = codes.DeadlineExceeded
```

## Metadata

```go
// Client: attach metadata to outgoing call.
md := metadata.Pairs("authorization", "Bearer "+token, "x-request-id", reqID)
ctx = metadata.NewOutgoingContext(ctx, md)

// Server: read metadata from incoming call.
md, _ := metadata.FromIncomingContext(ctx)
token := md.Get("authorization")[0]
```

## Production notes

- Always embed `pb.Unimplemented*Server` in your server struct — forwards-compatibility
- Set `keepalive.ServerParameters` to detect dead connections
- Use `grpc.WithBlock()` on client Dial only in tests; production should connect lazily
- Set `MaxRecvMsgSize` / `MaxSendMsgSize` to prevent oversized messages
- Enable server-side reflection (`reflection.Register(s)`) for `grpcurl` debugging
- For load balancing across multiple server instances, use `grpc.WithDefaultServiceConfig` with round-robin or least-loaded policy
