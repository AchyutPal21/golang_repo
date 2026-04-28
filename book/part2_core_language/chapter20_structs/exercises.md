# Chapter 20 — Exercises

## 20.1 — BankAccount with embedded AuditLog

Run [`exercises/01_bank_account`](exercises/01_bank_account/main.go).

The `BankAccount` embeds `AuditLog`, whose `record` method is unexported
but `Entries()` is promoted to `BankAccount`.

Try:
- Add a `Transfer(to *BankAccount, amount float64) error` method.
- Add a `WithdrawAll() float64` that drains the account.
- What happens if you change `AuditLog` embedding to `*AuditLog`? What must you fix?

## 20.2 ★ — Struct as map key

Design a `CacheKey` struct that encodes a method, path, and user tier.
Use it as a map key for a simple response cache. Verify two identical keys
hash to the same bucket (i.e., cache hits work).

## 20.3 ★ — JSON round-trip

Write a `Config` struct with nested `Database` and `Server` structs.
Serialise to JSON, modify the JSON string, and deserialise back.
Verify that `omitempty` fields disappear when empty and reappear when set.
Add a custom `UnmarshalJSON` to validate that `port` is in range 1–65535.
