# Chapter 61 — Authorization

## What you'll learn

How to control what authenticated users are allowed to do — the difference between authentication ("who are you?") and authorization ("what can you do?"). You'll implement two complementary models: Role-Based Access Control (RBAC) for coarse-grained permission checks, and Attribute-Based Access Control (ABAC) for fine-grained policy decisions that depend on context.

## RBAC vs ABAC

| | RBAC | ABAC |
|---|---|---|
| Decision based on | User's role | Subject + resource + action + environment attributes |
| Complexity | Low | High |
| Expressiveness | "editors can write" | "owner can edit before 5pm on weekdays" |
| Best for | Stable permission sets | Dynamic, contextual rules |

## Key concepts

**RBAC**
- Roles contain sets of permissions (`"article:read"`, `"article:write"`)
- Role inheritance: `admin` inherits all permissions of `editor` and `viewer`
- `HasPermission(role, permission) bool` — single entry point for checks
- HTTP middleware `requirePermission(perm)` enforces at the route level

**ABAC**
- `Subject` (who) + `Resource` (what) + `Action` (verb) + `Environment` (when/where)
- `Policy func(AccessRequest) (bool, error)` — a decision function
- `PolicySet` evaluates all policies; deny unless any permits (deny-by-default)
- Ownership check: resource owner can act regardless of role

## Files

| File | Topic |
|---|---|
| `examples/01_rbac/main.go` | Role hierarchy, `HasPermission`, HTTP middleware gates |
| `examples/02_abac/main.go` | Policy functions, PolicySet, ownership + time-based policies |
| `exercises/01_permission_system/main.go` | Combined RBAC+ABAC for a document store |

## RBAC pattern

```go
type RBAC struct {
    roles map[string][]string // role → []permission
}

func (rbac *RBAC) HasPermission(role, perm string) bool {
    for _, p := range rbac.roles[role] {
        if p == perm { return true }
    }
    return false
}

// HTTP middleware
func requirePermission(rbac *RBAC, perm string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            role := roleFromCtx(r.Context())
            if !rbac.HasPermission(role, perm) {
                http.Error(w, "forbidden", http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

## ABAC policy pattern

```go
type AccessRequest struct {
    Subject     Subject
    Resource    Resource
    Action      string
    Environment Environment
}

type Policy func(AccessRequest) (bool, error)

type PolicySet struct{ policies []Policy }

func (ps *PolicySet) Evaluate(req AccessRequest) (bool, error) {
    for _, p := range ps.policies {
        if ok, err := p(req); err != nil { return false, err } else if ok { return true, nil }
    }
    return false, nil // deny by default
}
```

## Production considerations

- Store roles in the database, not hardcoded — users change roles over time.
- Cache permission lookups; re-check on role change events.
- Audit log every authorization decision for compliance.
- Prefer RBAC for APIs; add ABAC policies for ownership and time-sensitive operations.
