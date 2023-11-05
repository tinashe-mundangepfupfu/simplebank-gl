package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Store provides all functions to execute db queries and transactions
type Store struct {
	*Queries // Composition the preferred way of extending struct function
	db       *sql.DB
}

// NewStore creates a new Store
func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		Queries: New(db),
	}
}

// execTx executes a function within a database transaction
func (store *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

// TransferTxParams contains input params of the transfer transaction
type TransferTxParams struct {
	FromAccountID int32 `json:"from_account_id"`
	ToAccountID   int32 `json:"to_account_id"`
	Amount        int64 `json:"amount"`
}

// TransferTxResult is the result of the transfer transaction
// TODO Fix Account Type int32 -> int64
type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount int32    `json:"from_account"`
	ToAccount   int32    `json:"to_account"`
	FromEntry   Entry    `json:"from_entry"`
	ToEntry     Entry    `json:"to_entry"`
}

// TransferTx Performs a money transfer from one account to another.
// It creates a record, add account entries, and update accounts' within a single db tx.
func (store *Store) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: int64(arg.FromAccountID),
			ToAccountID:   int64(arg.ToAccountID),
			Amount:        arg.Amount,
		})
		if err != nil {
			return err
		}

		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: int64(arg.FromAccountID),
			Amount:    -arg.Amount,
		})
		if err != nil {
			return err
		}

		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: int64(arg.FromAccountID),
			Amount:    arg.Amount,
		})
		if err != nil {
			return err
		}

		// TODO update accounts' balance
		return nil
	})

	return result, err
}
