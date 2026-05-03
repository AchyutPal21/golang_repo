# Chapter 61 Checkpoint — Authorization

## Self-assessment questions

1. What is the difference between authentication and authorization?
2. In RBAC, how do you implement role inheritance cleanly?
3. What does "deny by default" mean in an authorization system, and why is it the safe default?
4. What four attributes make up an ABAC `AccessRequest`?
5. When should you reach for ABAC instead of (or in addition to) RBAC?
6. Where in the middleware chain should authorization checks live?

## Checklist

- [ ] Can define roles with permission sets and implement `HasPermission`
- [ ] Can implement role inheritance (admin inherits editor which inherits viewer)
- [ ] Can write an HTTP middleware that extracts a role from context and denies forbidden requests
- [ ] Can define an ABAC `Policy` function and compose policies into a `PolicySet`
- [ ] Can implement an ownership policy that allows resource owners to act
- [ ] Can combine RBAC (coarse) and ABAC (fine-grained ownership) in a single decision path
- [ ] Understand deny-by-default: if no policy permits, deny

## Answers

1. Authentication verifies identity ("you are alice"). Authorization determines capability ("alice can create articles but not delete them"). Authentication must succeed before authorization is checked.

2. Expand inherited permissions at construction time or recursively walk the inheritance chain in `HasPermission`. Eager expansion (flatten at startup) is faster at runtime; lazy recursion is more flexible. Store the inheritance graph, not flattened lists.

3. "Deny by default" means if no rule explicitly grants access, access is denied. It is the safe default because missing rules cause denial (which breaks features but doesn't expose data) rather than accidental permission grants (which expose data).

4. `Subject` (who is acting), `Resource` (what they're acting on), `Action` (verb: read/write/delete), `Environment` (context: time of day, IP, request metadata).

5. When the decision depends on more than the user's role — e.g. "is this the resource's owner?", "is it business hours?", "is the user in the same organization as the resource?". ABAC handles these contextual conditions that RBAC cannot express.

6. As close to the route as possible — after authentication middleware has set the identity in context, either as a route-specific middleware or inside the handler itself. Global middleware is too broad; it can't know which resource is being accessed.
