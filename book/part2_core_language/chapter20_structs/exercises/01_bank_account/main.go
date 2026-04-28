// EXERCISE 20.1 — BankAccount with transaction history.
//
// Implement BankAccount with Deposit, Withdraw, Balance, and Statement.
// Use a struct with an embedded AuditLog that records all operations.
//
// Run (from the chapter folder):
//   go run ./exercises/01_bank_account

package main

import (
	"errors"
	"fmt"
	"time"
)

type TxType string

const (
	TxDeposit  TxType = "deposit"
	TxWithdraw TxType = "withdraw"
)

type Transaction struct {
	Type   TxType
	Amount float64
	At     time.Time
}

type AuditLog struct {
	entries []Transaction
}

func (a *AuditLog) record(typ TxType, amount float64) {
	a.entries = append(a.entries, Transaction{
		Type:   typ,
		Amount: amount,
		At:     time.Now(),
	})
}

func (a *AuditLog) Entries() []Transaction {
	// return a copy so callers cannot mutate the log
	cp := make([]Transaction, len(a.entries))
	copy(cp, a.entries)
	return cp
}

type BankAccount struct {
	owner   string
	balance float64
	AuditLog
}

func NewBankAccount(owner string, initial float64) *BankAccount {
	acc := &BankAccount{owner: owner}
	if initial > 0 {
		acc.balance = initial
		acc.record(TxDeposit, initial)
	}
	return acc
}

func (a *BankAccount) Deposit(amount float64) error {
	if amount <= 0 {
		return errors.New("deposit amount must be positive")
	}
	a.balance += amount
	a.record(TxDeposit, amount)
	return nil
}

func (a *BankAccount) Withdraw(amount float64) error {
	if amount <= 0 {
		return errors.New("withdraw amount must be positive")
	}
	if amount > a.balance {
		return fmt.Errorf("insufficient funds: have %.2f, need %.2f", a.balance, amount)
	}
	a.balance -= amount
	a.record(TxWithdraw, amount)
	return nil
}

func (a *BankAccount) Balance() float64 { return a.balance }

func (a *BankAccount) Statement() {
	fmt.Printf("=== Statement for %s ===\n", a.owner)
	for _, tx := range a.Entries() {
		sign := "+"
		if tx.Type == TxWithdraw {
			sign = "-"
		}
		fmt.Printf("  %s %s%.2f\n", tx.Type, sign, tx.Amount)
	}
	fmt.Printf("  Balance: %.2f\n", a.balance)
}

func main() {
	acc := NewBankAccount("Alice", 1000)

	_ = acc.Deposit(500)
	_ = acc.Withdraw(200)
	_ = acc.Deposit(100)

	err := acc.Withdraw(2000)
	if err != nil {
		fmt.Println("error:", err)
	}

	acc.Statement()

	fmt.Println()

	// AuditLog promoted methods accessible directly
	fmt.Println("Transaction count:", len(acc.Entries()))
}
