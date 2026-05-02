// FILE: book/part3_designing_software/chapter35_service_layer/examples/01_service_patterns/main.go
// CHAPTER: 35 — Service Layer
// TOPIC: Service layer patterns — use-case orchestration, input validation,
//        domain coordination, and the boundary between domain and application.
//
// Run (from the chapter folder):
//   go run ./examples/01_service_patterns

package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN LAYER
// ─────────────────────────────────────────────────────────────────────────────

type AccountID int
type TransferID int

type Money struct {
	Amount   int64  // cents — avoid float arithmetic
	Currency string // ISO 4217
}

func (m Money) String() string {
	return fmt.Sprintf("%s %.2f", m.Currency, float64(m.Amount)/100)
}

func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("currency mismatch: %s vs %s", m.Currency, other.Currency)
	}
	return Money{Amount: m.Amount + other.Amount, Currency: m.Currency}, nil
}

func (m Money) Sub(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("currency mismatch: %s vs %s", m.Currency, other.Currency)
	}
	return Money{Amount: m.Amount - other.Amount, Currency: m.Currency}, nil
}

func USD(cents int64) Money { return Money{Amount: cents, Currency: "USD"} }

// Domain errors.
var (
	ErrAccountNotFound    = errors.New("account not found")
	ErrInsufficientFunds  = errors.New("insufficient funds")
	ErrSameAccount        = errors.New("source and destination are the same account")
	ErrNegativeAmount     = errors.New("transfer amount must be positive")
	ErrAccountSuspended   = errors.New("account is suspended")
)

type AccountStatus string

const (
	AccountActive    AccountStatus = "active"
	AccountSuspended AccountStatus = "suspended"
)

type Account struct {
	ID        AccountID
	OwnerName string
	Balance   Money
	Status    AccountStatus
}

// Domain method — the business rule lives on the entity.
func (a *Account) Debit(amount Money) error {
	if a.Status == AccountSuspended {
		return ErrAccountSuspended
	}
	newBalance, err := a.Balance.Sub(amount)
	if err != nil {
		return err
	}
	if newBalance.Amount < 0 {
		return ErrInsufficientFunds
	}
	a.Balance = newBalance
	return nil
}

func (a *Account) Credit(amount Money) error {
	if a.Status == AccountSuspended {
		return ErrAccountSuspended
	}
	newBalance, err := a.Balance.Add(amount)
	if err != nil {
		return err
	}
	a.Balance = newBalance
	return nil
}

type Transfer struct {
	ID          TransferID
	FromAccount AccountID
	ToAccount   AccountID
	Amount      Money
	CreatedAt   time.Time
}

// ─────────────────────────────────────────────────────────────────────────────
// PORTS (defined in application layer)
// ─────────────────────────────────────────────────────────────────────────────

type AccountRepository interface {
	Save(a Account) (Account, error)
	FindByID(id AccountID) (Account, error)
	FindAll() ([]Account, error)
}

type TransferRepository interface {
	Save(t Transfer) (Transfer, error)
	FindByAccount(id AccountID) ([]Transfer, error)
}

type Clock interface {
	Now() time.Time
}

type AuditLog interface {
	Record(action, detail string)
}

// ─────────────────────────────────────────────────────────────────────────────
// SERVICE LAYER
//
// The service layer:
//   1. Validates inputs (not domain rules — those live on entities)
//   2. Loads domain objects from repositories
//   3. Calls domain methods (business logic)
//   4. Persists results
//   5. Coordinates side effects (notifications, audit logs)
//   6. Returns results to the caller
// ─────────────────────────────────────────────────────────────────────────────

// TransferRequest — input DTO (Data Transfer Object).
// The service receives plain data from the transport layer.
type TransferRequest struct {
	FromAccountID AccountID
	ToAccountID   AccountID
	AmountCents   int64
	Currency      string
	Note          string
}

// Validate checks inputs before loading domain objects — fail fast.
func (r TransferRequest) Validate() error {
	if r.FromAccountID == r.ToAccountID {
		return ErrSameAccount
	}
	if r.AmountCents <= 0 {
		return ErrNegativeAmount
	}
	if strings.TrimSpace(r.Currency) == "" {
		return fmt.Errorf("currency is required")
	}
	return nil
}

type BankingService struct {
	accounts  AccountRepository
	transfers TransferRepository
	clock     Clock
	audit     AuditLog
	nextTxID  TransferID
}

func NewBankingService(
	accounts AccountRepository,
	transfers TransferRepository,
	clock Clock,
	audit AuditLog,
) *BankingService {
	return &BankingService{
		accounts:  accounts,
		transfers: transfers,
		clock:     clock,
		audit:     audit,
	}
}

// Transfer is a use-case method — one business operation, beginning to end.
func (s *BankingService) Transfer(req TransferRequest) (Transfer, error) {
	// Step 1: validate inputs
	if err := req.Validate(); err != nil {
		return Transfer{}, fmt.Errorf("Transfer: %w", err)
	}

	// Step 2: load domain objects
	from, err := s.accounts.FindByID(req.FromAccountID)
	if err != nil {
		return Transfer{}, fmt.Errorf("Transfer: from account: %w", err)
	}
	to, err := s.accounts.FindByID(req.ToAccountID)
	if err != nil {
		return Transfer{}, fmt.Errorf("Transfer: to account: %w", err)
	}

	amount := Money{Amount: req.AmountCents, Currency: req.Currency}

	// Step 3: invoke domain logic (entity methods enforce business rules)
	if err := from.Debit(amount); err != nil {
		return Transfer{}, fmt.Errorf("Transfer: debit failed: %w", err)
	}
	if err := to.Credit(amount); err != nil {
		return Transfer{}, fmt.Errorf("Transfer: credit failed: %w", err)
	}

	// Step 4: persist
	if _, err := s.accounts.Save(from); err != nil {
		return Transfer{}, fmt.Errorf("Transfer: save from: %w", err)
	}
	if _, err := s.accounts.Save(to); err != nil {
		return Transfer{}, fmt.Errorf("Transfer: save to: %w", err)
	}

	s.nextTxID++
	tx := Transfer{
		ID:          s.nextTxID,
		FromAccount: req.FromAccountID,
		ToAccount:   req.ToAccountID,
		Amount:      amount,
		CreatedAt:   s.clock.Now(),
	}
	saved, err := s.transfers.Save(tx)
	if err != nil {
		return Transfer{}, fmt.Errorf("Transfer: save transfer: %w", err)
	}

	// Step 5: side effects
	s.audit.Record("transfer",
		fmt.Sprintf("tx=%d from=%d to=%d amount=%s",
			saved.ID, req.FromAccountID, req.ToAccountID, amount))

	return saved, nil
}

func (s *BankingService) OpenAccount(ownerName string, initialDepositCents int64) (Account, error) {
	if strings.TrimSpace(ownerName) == "" {
		return Account{}, fmt.Errorf("OpenAccount: owner name is required")
	}
	a := Account{
		OwnerName: ownerName,
		Balance:   USD(initialDepositCents),
		Status:    AccountActive,
	}
	created, err := s.accounts.Save(a)
	if err != nil {
		return Account{}, fmt.Errorf("OpenAccount: %w", err)
	}
	s.audit.Record("open_account", fmt.Sprintf("id=%d owner=%s", created.ID, ownerName))
	return created, nil
}

func (s *BankingService) GetStatement(accountID AccountID) {
	acc, err := s.accounts.FindByID(accountID)
	if err != nil {
		fmt.Println("  error:", err)
		return
	}
	txs, _ := s.transfers.FindByAccount(accountID)
	fmt.Printf("  Account #%d  %s  balance=%s\n", acc.ID, acc.OwnerName, acc.Balance)
	for _, tx := range txs {
		dir := "sent"
		if tx.ToAccount == accountID {
			dir = "received"
		}
		fmt.Printf("    tx#%d %-8s  %s  at %s\n",
			tx.ID, dir, tx.Amount, tx.CreatedAt.Format("15:04:05"))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// INFRASTRUCTURE
// ─────────────────────────────────────────────────────────────────────────────

type memAccountRepo struct {
	data   map[AccountID]Account
	nextID AccountID
}

func newMemAccountRepo() *memAccountRepo {
	return &memAccountRepo{data: make(map[AccountID]Account), nextID: 1}
}

func (r *memAccountRepo) Save(a Account) (Account, error) {
	if a.ID == 0 {
		a.ID = r.nextID
		r.nextID++
	}
	r.data[a.ID] = a
	return a, nil
}

func (r *memAccountRepo) FindByID(id AccountID) (Account, error) {
	a, ok := r.data[id]
	if !ok {
		return Account{}, ErrAccountNotFound
	}
	return a, nil
}

func (r *memAccountRepo) FindAll() ([]Account, error) {
	result := make([]Account, 0, len(r.data))
	for _, a := range r.data {
		result = append(result, a)
	}
	return result, nil
}

type memTransferRepo struct{ transfers []Transfer }

func (r *memTransferRepo) Save(t Transfer) (Transfer, error) {
	r.transfers = append(r.transfers, t)
	return t, nil
}

func (r *memTransferRepo) FindByAccount(id AccountID) ([]Transfer, error) {
	var result []Transfer
	for _, t := range r.transfers {
		if t.FromAccount == id || t.ToAccount == id {
			result = append(result, t)
		}
	}
	return result, nil
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

type stdoutAuditLog struct{}

func (stdoutAuditLog) Record(action, detail string) {
	fmt.Printf("  [AUDIT] action=%s %s\n", action, detail)
}

func main() {
	svc := NewBankingService(
		newMemAccountRepo(),
		&memTransferRepo{},
		realClock{},
		stdoutAuditLog{},
	)

	fmt.Println("=== Open accounts ===")
	alice, _ := svc.OpenAccount("Alice", 100000) // $1000.00
	bob, _ := svc.OpenAccount("Bob", 50000)      // $500.00
	carol, _ := svc.OpenAccount("Carol", 200000) // $2000.00

	fmt.Println()
	fmt.Println("=== Transfers ===")
	tx1, err := svc.Transfer(TransferRequest{
		FromAccountID: alice.ID,
		ToAccountID:   bob.ID,
		AmountCents:   2500, // $25.00
		Currency:      "USD",
	})
	if err != nil {
		fmt.Println("  error:", err)
	} else {
		fmt.Printf("  tx#%d  %s transferred\n", tx1.ID, tx1.Amount)
	}

	_, err = svc.Transfer(TransferRequest{
		FromAccountID: bob.ID,
		ToAccountID:   carol.ID,
		AmountCents:   90000, // $900 — exceeds bob's $525 balance
		Currency:      "USD",
	})
	fmt.Println("  expected error:", err)
	fmt.Println("  is ErrInsufficientFunds:", errors.Is(err, ErrInsufficientFunds))

	// Validation error.
	_, err = svc.Transfer(TransferRequest{
		FromAccountID: alice.ID,
		ToAccountID:   alice.ID,
		AmountCents:   100,
		Currency:      "USD",
	})
	fmt.Println("  same-account error:", err)

	fmt.Println()
	fmt.Println("=== Statements ===")
	svc.GetStatement(alice.ID)
	fmt.Println()
	svc.GetStatement(bob.ID)
}
