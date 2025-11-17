package pgxtx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type ctxKey struct{}

type manager struct {
	pool txOwner
	log  *slog.Logger
}

type txOwner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	PgxExecutor
}

var errConnRequired = errors.New("pgxpool connection is required for create transaction manager")

type transaction struct {
	pgx.Tx
}

func Init(conn txOwner, l ...*slog.Logger) (Manager, error) {
	if conn == nil {
		return nil, errConnRequired
	}
	return &manager{
		pool: conn,
		log:  l[0],
	}, nil
}

type PgxExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Extractor interface {
	ExtractTx(ctx context.Context) PgxExecutor
}

type Manager interface {
	Wrap(ctx context.Context, fn func(context.Context) error) error
	GetExtractor() Extractor
}

func (m *manager) GetExtractor() Extractor {
	return m
}

func (m *manager) Wrap(ctx context.Context, fn func(context.Context) error) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("pool.Begin: cannot start transaction: %w", err)
	}

	defer func() {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			if m.log != nil {
				m.log.Error("error when try to rollback transaction", slog.String("error", rollbackErr.Error()))
			}
		} else if rollbackErr == nil {
			if m.log != nil {
				m.log.Info("transaction successfully rollbacked")
			}
		}
	}()

	ctx = context.WithValue(ctx, ctxKey{}, &transaction{tx})

	err = fn(ctx)

	if err != nil {
		return fmt.Errorf("error when execute transaction fn: %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return fmt.Errorf("error when commit: %w", err)
	}

	return nil
}

func (m *manager) ExtractTx(ctx context.Context) PgxExecutor {
	executor, ok := ctx.Value(ctxKey{}).(*transaction)
	if !ok {
		return m.pool
	}

	return executor
}
