package sqlxutil

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

type txContextKey struct{}

type namedQueryExecutor interface {
	sqlx.ExtContext
}

type namedExecExecutor interface {
	sqlx.ExtContext
}

type getExecutor interface {
	GetContext(ctx context.Context, dest any, query string, args ...any) error
}

type selectExecutor interface {
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
}

// WithTx 把当前事务绑定到 context，方便 repository 在不改方法签名的情况下复用同一事务。
func WithTx(ctx context.Context, tx *sqlx.Tx) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

// TxFromContext 返回 context 中携带的事务。
func TxFromContext(ctx context.Context) *sqlx.Tx {
	tx, _ := ctx.Value(txContextKey{}).(*sqlx.Tx)
	return tx
}

// WithTransaction 统一开启事务，并把事务对象注入新的 context 传给回调函数。
func WithTransaction(ctx context.Context, db *sqlx.DB, fn func(context.Context) error) (err error) {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	txCtx := WithTx(ctx, tx)
	defer func() {
		if recoverValue := recover(); recoverValue != nil {
			_ = tx.Rollback()
			panic(recoverValue)
		}
		if err != nil {
			_ = tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	err = fn(txCtx)
	return err
}

func namedQueryDB(ctx context.Context, db *sqlx.DB) namedQueryExecutor {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return db
}

func namedExecDB(ctx context.Context, db *sqlx.DB) namedExecExecutor {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return db
}

func getDB(ctx context.Context, db *sqlx.DB) getExecutor {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return db
}

func selectDB(ctx context.Context, db *sqlx.DB) selectExecutor {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return db
}

// NamedSelect 执行 NamedQuery 并把结果集扫描为结构体切片。
func NamedSelect[T any](ctx context.Context, db *sqlx.DB, query string, arg any) ([]T, error) {
	rows, err := sqlx.NamedQueryContext(ctx, namedQueryDB(ctx, db), query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]T, 0)
	for rows.Next() {
		var item T
		if err := rows.StructScan(&item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// NamedExec 统一执行 NamedExec。
func NamedExec(ctx context.Context, db *sqlx.DB, query string, arg any) error {
	_, err := sqlx.NamedExecContext(ctx, namedExecDB(ctx, db), query, arg)
	return err
}

// NamedInsertID 执行带 RETURNING id 的 NamedQuery，并返回新生成的主键。
func NamedInsertID(ctx context.Context, db *sqlx.DB, query string, arg any) (int64, error) {
	rows, err := sqlx.NamedQueryContext(ctx, namedQueryDB(ctx, db), query, arg)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		return id, nil
	}
	return 0, sql.ErrNoRows
}

// GetOne 执行单行查询并扫描到结构体。
func GetOne[T any](ctx context.Context, db *sqlx.DB, query string, args ...any) (*T, error) {
	var item T
	if err := getDB(ctx, db).GetContext(ctx, &item, query, args...); err != nil {
		return nil, err
	}
	return &item, nil
}

// SelectMany 执行普通 Select 查询并扫描为结构体切片。
func SelectMany[T any](ctx context.Context, db *sqlx.DB, query string, args ...any) ([]T, error) {
	items := make([]T, 0)
	if err := selectDB(ctx, db).SelectContext(ctx, &items, query, args...); err != nil {
		return nil, err
	}
	return items, nil
}
