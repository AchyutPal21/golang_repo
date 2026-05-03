# Chapter 61 Exercises — Authorization

## Exercise 1 — Permission System (`exercises/01_permission_system`)

Build a document store API with a combined RBAC + ABAC authorization layer.

### Roles and permissions

| Role | Permissions |
|---|---|
| `viewer` | `doc:read` |
| `editor` | `doc:read`, `doc:write`, `doc:create` |
| `admin` | all editor permissions + `doc:delete`, `user:manage` |

### Authorization rules (combine both models)

1. **RBAC gate**: user's role must include the required permission
2. **Ownership override**: a user who is the document's `OwnerID` may `write` their own document even if their role doesn't grant `doc:write` (e.g. a `viewer` can edit their own doc)
3. **Admin bypass**: `admin` role always passes — skip ownership checks

### Users for testing

| User | Role |
|---|---|
| `alice` | `admin` |
| `bob` | `editor` |
| `carol` | `viewer` |
| `dave` | `viewer` (owns doc-2) |

### Documents

| ID | Title | OwnerID |
|---|---|---|
| `doc-1` | "Getting Started" | `alice` |
| `doc-2` | "My Draft" | `dave` |
| `doc-3` | "Reference" | `bob` |

### Endpoints

| Method | Path | Required permission |
|---|---|---|
| `GET` | `/docs/{id}` | `doc:read` |
| `PUT` | `/docs/{id}` | `doc:write` (or owner) |
| `DELETE` | `/docs/{id}` | `doc:delete` |

### Auth header

Use `Authorization: Bearer <username>` for test simplicity — the middleware resolves the username to the user struct and stores it in context.

### Expected behaviour

| User | Action | Doc | Result |
|---|---|---|---|
| `carol` | GET | doc-1 | 200 (viewer can read) |
| `carol` | PUT | doc-1 | 403 (viewer, not owner) |
| `dave` | PUT | doc-2 | 200 (viewer but owner) |
| `dave` | PUT | doc-1 | 403 (viewer, not owner) |
| `bob` | PUT | doc-1 | 200 (editor) |
| `alice` | DELETE | doc-1 | 200 (admin) |
| `carol` | DELETE | doc-1 | 403 (viewer, no delete perm) |
| anonymous | GET | doc-1 | 401 (no token) |

### Hints

- Implement `CanAct(user User, doc Document, action string) bool` combining both models
- Store the user in context with a typed key; retrieve in each handler
- A `403` means authenticated but forbidden; `401` means not authenticated
