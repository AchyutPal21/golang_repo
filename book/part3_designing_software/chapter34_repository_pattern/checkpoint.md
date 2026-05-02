# Chapter 34 — Revision Checkpoint

## Questions

1. What are the four rules a repository interface must follow?
2. Why does `Save` use upsert semantics (zero-ID = insert, non-zero = update) rather than separate `Insert`/`Update` methods?
3. What problem does the Specification pattern solve compared to having many `FindBy*` methods?
4. What information does `PagedResult` carry and why is `TotalCount` necessary?
5. Should a repository perform domain validation (e.g., reject a negative price)? Why or why not?

## Answers

1. The four rules:
   - **Domain types only** — accept and return domain structs, not driver types (`*sql.Rows`, bson.M, etc.)
   - **Domain sentinel errors** — wrap all driver/infrastructure errors into domain-layer errors before returning
   - **No leaking** — never expose storage-level abstractions (`*sql.Tx`, cursors) through the interface
   - **No business logic** — a repository stores and retrieves; domain rules belong in the domain or service layer

2. A single `Save` method simplifies the caller: it does not need to know whether
   the entity already exists. The repository decides based on the ID. This also
   prevents a class of bugs where the caller calls `Insert` on an entity that
   already exists (causing a duplicate key error) or calls `Update` on a new entity
   (causing a not-found error). The upsert contract is easier to reason about and
   test.

3. Adding a new query combination like "active products in the gadgets category
   under $50 that are in stock" would require a new `FindActiveGadgetsUnder50InStock`
   method. With N dimensions (active, category, price range, stock) the interface
   explodes combinatorially. The Specification pattern composes leaf specs with
   `And`/`Or`/`Not` — new combinations require zero new interface methods. Each
   spec is testable in isolation, and the composition is readable.

4. `PagedResult` carries: `Items []T` (the current page), `TotalCount int` (total
   matching records across all pages), and `Page` (the requested page number and
   size). `TotalCount` is necessary because the caller needs to know how many pages
   exist (`TotalPages = ceil(TotalCount / Page.Size)`). Without it, the caller
   cannot tell whether the current page is the last one or whether there are more
   pages to fetch.

5. No. A repository must not perform domain validation. Validation is a domain or
   application concern — it belongs in the domain entity constructor, domain methods,
   or the service layer. If the repository validates, the same rule must be duplicated
   in both the repository and the service, creating inconsistency when one is updated
   and the other is not. The repository's only job is faithful persistence and
   retrieval; it trusts the data it receives from the layer above.
