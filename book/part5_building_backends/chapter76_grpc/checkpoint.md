# Chapter 76 Checkpoint — gRPC

## Self-assessment questions

1. What are the four gRPC call types and a concrete use case for each?
2. How do gRPC status codes differ from HTTP status codes? Why does gRPC have its own?
3. What is the difference between gRPC metadata and message fields? When should you use each?
4. Why should you always set a deadline on a gRPC call? What happens if you don't?
5. How do interceptors differ from HTTP middleware? What are `UnaryInterceptor` and `StreamInterceptor`?
6. What does embedding `pb.UnimplementedProductServiceServer` do in a server struct, and why is it best practice?

## Checklist

- [ ] Can write a proto service definition with unary and streaming methods
- [ ] Can implement a gRPC server struct satisfying the generated interface
- [ ] Can return structured errors using `status.Errorf(codes.X, ...)`
- [ ] Can inspect incoming errors with `status.FromError(err)`
- [ ] Can implement a unary interceptor that wraps logging, auth, and tracing
- [ ] Can chain multiple interceptors using `grpc.ChainUnaryInterceptor`
- [ ] Can attach metadata to outgoing calls and read it on the server
- [ ] Can set and propagate deadlines with `context.WithTimeout`
- [ ] Can implement a server streaming RPC using `stream.Send`
- [ ] Can implement a bidirectional stream with concurrent send and receive goroutines

## Answers

1. **Unary**: `GetProduct(req) → resp` — simple queries, CRUD operations. **Server streaming**: `ListProducts(req) → stream of Product` — large result sets, real-time feeds. **Client streaming**: `stream of InventoryUpdate → BulkResult` — batch mutations, large file upload in chunks. **Bidirectional**: `stream of PriceRequest ↔ stream of PriceUpdate` — chat, live price feeds where the client dynamically adjusts subscriptions.

2. HTTP status codes are numeric conventions without a formal spec for every value and no structured message. gRPC status codes are a defined set in the gRPC spec (0–16), each with a precise meaning (e.g., `UNAVAILABLE` = temporarily overloaded, safe to retry; `NOT_FOUND` = resource does not exist). They carry a `Code` plus a free-form human message. gRPC uses its own because it runs over HTTP/2 and the response payload, not the HTTP status line, carries the error.

3. Metadata is call-level context (auth token, trace ID, correlation ID) that travels alongside the RPC but is not part of the domain message. Message fields are application data. Use metadata for cross-cutting concerns that apply to every call in a service; use message fields for data specific to that operation. Reading metadata in every method handler is boilerplate — interceptors are the right place to extract metadata.

4. Without a deadline, a slow or unresponsive server holds the call open indefinitely, consuming goroutine stack, file descriptor, and memory until the process is restarted. Deadlines are propagated via HTTP/2 headers to downstream services, so setting a deadline on one call also bounds the cascading calls it triggers. `context.WithTimeout(ctx, 5*time.Second)` is the standard pattern.

5. HTTP middleware wraps `http.Handler` and works at the request/response level. gRPC interceptors wrap the stub-generated handler and work at the RPC level. `UnaryInterceptor` handles one request → one response; it has the same structure as HTTP middleware. `StreamInterceptor` wraps streaming RPCs and receives a `grpc.ServerStream` interface, letting you intercept individual `Send`/`Recv` calls within the stream.

6. `pb.UnimplementedProductServiceServer` is a generated struct where every method returns `codes.Unimplemented`. Embedding it means your server automatically satisfies the generated interface even if you haven't implemented all methods yet — adding a new method to the proto only breaks compilation where the method is missing, not everywhere. Without it, adding a new proto method would require updating all server implementations immediately.
