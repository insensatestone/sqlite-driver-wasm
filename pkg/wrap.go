package wrap

import (
	"context"
	"database/sql"
	"errors"
)

var (
	ErrNotImplemented = errors.New("not implemented")
	ErrMaxNOExceed    = errors.New("max number exceed")
	ErrRowChanged     = errors.New("row changed")
)

type DBWrap struct {
	db *sql.DB
	// conn *sql.Conn
}

type TxWrap struct {
	Tx        *sql.Tx
	Count     uint32
	Committed bool
}

var ctx_key_tx = struct{ name string }{name: "tx"}

func NewDBWrap(db *sql.DB) *DBWrap {
	// conn, _ := db.Conn(context.Background())
	return &DBWrap{
		db: db,
		// conn: conn,
	}
}

func (dbw *DBWrap) Begin() (context.Context, error) {
	tx, err := dbw.db.Begin()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	return context.WithValue(ctx, ctx_key_tx, &TxWrap{
		Tx:        tx,
		Count:     1,
		Committed: false,
	}), nil
}

func (dbw *DBWrap) BeginContext(ctx context.Context) (context.Context, error) {
	if tw, ok := ctx.Value(ctx_key_tx).(*TxWrap); ok {
		tw.Count = tw.Count + 1
		return ctx, nil
	} else {
		tx, err := dbw.db.Begin()
		if err != nil {
			return nil, err
		}
		return context.WithValue(ctx, ctx_key_tx, &TxWrap{
			Tx:    tx,
			Count: 1,
		}), nil
	}
}

func (dbw *DBWrap) Close() error {
	// dbw.conn.Close()
	return dbw.db.Close()
}

func (dbw *DBWrap) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if tw, ok := ctx.Value(ctx_key_tx).(*TxWrap); ok {
		return tw.Tx.QueryContext(ctx, query, args...)
	}
	// rows, err := dbw.db.QueryContext(ctx, query, args...)
	// fmt.Println(rows.Next())
	// return rows, err
	// return dbw.conn.QueryContext(ctx, query, args...)
	return dbw.db.QueryContext(ctx, query, args...)
}

func (dbw *DBWrap) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	if tw, ok := ctx.Value(ctx_key_tx).(*TxWrap); ok {
		return tw.Tx.QueryRowContext(ctx, query, args...)
	}
	// return dbw.conn.QueryRowContext(ctx, query, args...)
	return dbw.db.QueryRowContext(ctx, query, args...)
}

func (dbw *DBWrap) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if tw, ok := ctx.Value(ctx_key_tx).(*TxWrap); ok {
		return tw.Tx.ExecContext(ctx, query, args...)
	}
	// return dbw.conn.ExecContext(ctx, query, args...)
	return dbw.db.ExecContext(ctx, query, args...)
}

func (dbw *DBWrap) Prepare(ctx context.Context, query string) (*sql.Stmt, error) {
	if tw, ok := ctx.Value(ctx_key_tx).(*TxWrap); ok {
		return tw.Tx.PrepareContext(ctx, query)
	}
	// slog.Info("prepare")
	return dbw.db.PrepareContext(ctx, query)

	// return nil, errors.New("need transaction")
}

func (dbw *DBWrap) Rollback(ctx context.Context) error {
	if tw, ok := ctx.Value(ctx_key_tx).(*TxWrap); ok {
		var err error
		if !tw.Committed {
			if tw.Count > 0 {
				tw.Count = tw.Count - 1
				if tw.Count == 0 {
					err = tw.Tx.Rollback()
				}
			} else {
				err = tw.Tx.Rollback()
				// return err
			}
		} else {
			tw.Committed = false
		}
		return err
	} else {
		return errors.New("need transaction")
	}
}

// func (dbw *DBWrap) RollbackError(ctx context.Context, rerr *error) error {
// 	if *rerr != nil {
// 		if tw, ok := ctx.Value(ctx_key_tx).(*TxWrap); ok {
// 			if tw.Count > 0 {
// 				tw.Count = tw.Count - 1
// 				if tw.Count == 0 {
// 					return tw.Tx.Rollback()
// 				}
// 			} else {
// 				err := tw.Tx.Rollback()
// 				return err
// 			}
// 		} else {
// 			return errors.New("need transaction")
// 		}
// 	}
// 	return nil
// }

func (dbw *DBWrap) Commit(ctx context.Context) error {
	if tw, ok := ctx.Value(ctx_key_tx).(*TxWrap); ok {
		var err error
		if tw.Count > 0 {
			tw.Count = tw.Count - 1
			if tw.Count == 0 {
				err = tw.Tx.Commit()
			}
		} else {
			err = tw.Tx.Commit()
		}
		if err != nil {
			tw.Count = tw.Count + 1
		} else {
			tw.Committed = true
		}
		return err
	} else {
		return errors.New("need transaction")
	}
}
