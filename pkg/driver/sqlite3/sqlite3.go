//go:build js

package sqlite3

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"
	"syscall/js"

	"github.com/insensatestone/sqlite-driver-wasm/internal/async"

	"github.com/rs/xid"
)

var driverName = "sqlite3_wasm"

const (
	SQLITE_INTEGER = 1
	SQLITE_FLOAT   = 2
	SQLITE3_TEXT   = 3
	SQLITE_BLOB    = 4
	SQLITE_NULL    = 5
)

func init() {
	if driverName != "" {
		sql.Register(driverName, &SQLiteDriver{})
	}
}

type SQLiteDriver struct {
}

type SQLiteConn struct {
	db js.Value
	// tx   *SQLiteTx
}

type SQLiteStmt struct {
	ID    string
	query string
	// args  [][]driver.Value
	stmt js.Value
	conn *SQLiteConn
}

type SQLiteTx struct {
	ID   string
	conn *SQLiteConn
	// stmt map[string]*SQLiteStmt
}

type SQLiteResult struct {
	rows_affected  int64
	last_insert_id int64
}

type SQLiteRows struct {
	st *SQLiteStmt
}

func (d *SQLiteDriver) Open(dsn string) (conn driver.Conn, err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Debug("err", r)
			err = errors.New("driver open failed")
		}
	}()
	db_object := js.Global().Get("sqlite3").Get("oo1").Get("OpfsDb")
	if db_object.IsUndefined() || db_object.IsNull() || db_object.IsNaN() {
		return nil, errors.New("open db failed,need support for opfs")
	}
	db := db_object.New(dsn)
	return &SQLiteConn{
		db: db,
	}, nil
}

func (c *SQLiteConn) Exec(query string, args []driver.Value) (rs driver.Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Debug("err", r)
			err = errors.New("driver conn exec failed")
		}
	}()
	stmt, err := c.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	return stmt.Exec(args)
	// return nil, nil
}
func (c *SQLiteConn) Prepare(query string) (ds driver.Stmt, err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Debug("err", r)
			err = errors.New("driver conn prepare failed")
		}
	}()
	stmt := c.db.Call("prepare", query)
	// count := stmt.Get("parameterCount")
	// slog.Info("parameterCount", "count", count.Int())
	return &SQLiteStmt{
		stmt:  stmt,
		conn:  c,
		query: query,
	}, nil
}

func (c *SQLiteConn) Begin() (driver.Tx, error) {
	// c.tx = &SQLiteTx{
	// 	ID: uuid.NewString(),
	// 	conn: c,
	// 	stmt: make([][]driver.Value, 0),
	// }
	_, err := c.Exec("BEGIN", nil)
	if err != nil {
		return nil, err
	}
	return &SQLiteTx{
		ID:   xid.New().String(),
		conn: c,
	}, nil
}
func (c *SQLiteConn) Backup(path string) error {
	js_dump := js.Global().Get("sqlite3").Get("capi").Call("sqlite3_js_db_export", c.db)
	dirs, filename := filepath.Split(path)
	if filename == "" {
		return fs.ErrInvalid
	}

	dir, err := async.Await(js.Global().Get("navigator").Get("storage").Call("getDirectory"))
	if err != nil {
		return err
	}
	if dirs != "" {
		dira := strings.Split(strings.Trim(dirs, string(filepath.Separator)), string(filepath.Separator))
		var err error
		for _, dirname := range dira {
			dir, err = async.Await(dir.Call("getDirectoryHandle", dirname, map[string]interface{}{"create": true}))
			if err != nil {
				return err
			}
		}
	}
	file, err := async.Await(dir.Call("getFileHandle", filename, map[string]interface{}{"create": true}))
	if err != nil {
		return err
	}
	fa, err := async.Await(file.Call("createSyncAccessHandle"))
	if err != nil {
		return err
	}
	defer fa.Call("close")
	fa.Call("truncate", 0)
	fa.Call("write", js_dump, map[string]interface{}{"at": 0})
	fa.Call("flush")

	return nil
}

func (c *SQLiteConn) Close() error {
	c.db.Call("close")
	return nil
}

func (s *SQLiteStmt) bind(args []driver.Value) {
	if len(args) > 0 {
		values := make([]any, len(args))
		for i := 0; i < len(args); i++ {
			if b, ok := args[i].([]byte); ok {
				jb := js.Global().Get("Uint8Array").New(len(b))
				js.CopyBytesToJS(jb, b)
				values[i] = jb
			} else {
				values[i] = args[i]
			}
		}
		s.stmt.Call("bind", values)
	}
}
func (s *SQLiteStmt) Exec(args []driver.Value) (rs driver.Result, err error) {
	// slog.Info("args", "len", len(args), "args", args)
	defer func() {
		if r := recover(); r != nil {
			slog.Debug("err", r)
			err = errors.New("driver st exec failed")
		}
	}()
	s.stmt.Call("reset", true)
	s.bind(args)
	// slog.Info("exec sql begin", "sql", s.query)
	// s.stmt.Call("stepFinalize")
	s.stmt.Call("step")
	// slog.Info("exec sql sucess", "sql", s.query)
	// slog.Info("call step sucess")
	affected := js.Global().Get("sqlite3").Get("capi").Call("sqlite3_changes", s.conn.db).Int()
	// slog.Info("call sqlite3_changes sucess", "changes", affected, "sql", s.query)
	// rowid_str := js.Global().Get("sqlite3").Get("capi").Call("sqlite3_last_insert_rowid", s.conn.db)
	// slog.Info("call sqlite3_last_insert_rowid sucess")
	// rowid, err := strconv.ParseInt(rowid_str, 10, 64)
	// if err != nil {
	// 	rowid = 0
	// }
	return &SQLiteResult{
		last_insert_id: 0,
		rows_affected:  int64(affected),
	}, nil
}

func (s *SQLiteStmt) Query(args []driver.Value) (rows driver.Rows, err error) {
	// slog.Info("parameterCount", "count", len(args))
	defer func() {
		if r := recover(); r != nil {
			slog.Debug("err", r)
			err = errors.New("driver st query failed")
		}
	}()
	s.stmt.Call("reset", true)
	// count := s.stmt.Get("parameterCount")
	// slog.Info("reset")
	s.bind(args)
	// if len(args) > 0 {
	// 	values := make([]any, len(args))
	// 	for i := 0; i < len(args); i++ {
	// 		if b, ok := args[i].([]byte); ok {
	// 			jb := js.Global().Get("Uint8Array").New(len(b))
	// 			js.CopyBytesToJS(jb, b)
	// 			values[i] = jb
	// 		} else {
	// 			values[i] = args[i]
	// 		}
	// 	}
	// 	s.stmt.Call("bind", values)
	// }
	return &SQLiteRows{
		st: s,
	}, nil
}

func (s *SQLiteStmt) NumInput() int {
	count := s.stmt.Get("parameterCount")
	// slog.Info("parameterCount", "count", count.Int())
	return count.Int()
}

func (s *SQLiteStmt) Close() error {
	s.stmt.Call("finalize")
	return nil
}

func (tx *SQLiteTx) Commit() error {
	// if len(tx.stmt) > 0 {
	// 	for _, v := range tx.stmt {
	// 		for _, args := range v.args {
	// 			v.Exec(args)
	// 		}
	// 	}
	// }
	// tx.conn.tx = nil
	_, err := tx.conn.Exec("COMMIT", nil)
	if err != nil {
		return err
	}
	return nil
}

func (tx *SQLiteTx) Rollback() error {
	// tx.conn.tx = nil
	_, err := tx.conn.Exec("ROLLBACK", nil)
	if err != nil {
		return err
	}
	return nil
}

func (r *SQLiteResult) LastInsertId() (int64, error) {
	return r.last_insert_id, nil
}

func (r *SQLiteResult) RowsAffected() (int64, error) {
	return r.rows_affected, nil
}

func (rc *SQLiteRows) Columns() []string {
	names_js := rc.st.stmt.Call("getColumnNames")
	len := names_js.Length()
	names := make([]string, len)
	for i := 0; i < len; i++ {
		names[i] = names_js.Index(i).String()
	}
	return names
}

func (rc *SQLiteRows) Next(dest []driver.Value) (err error) {
	// slog.Debug("driver next")
	defer func() {
		if r := recover(); r != nil {
			slog.Debug("err", r)
			err = errors.New("driver row next failed")
		}
	}()
	hasrow := rc.st.stmt.Call("step")
	if !hasrow.Bool() {
		return io.EOF
	}
	// defer rc.stmt.Call("reset")
	// slog.Info("rs code", "code", rc.st.stmt.Call("getInt", 0).Int())
	// slog.Info("rs len", "dest", len(dest))
	for i := 0; i < len(dest); i++ {
		defaultType := js.Global().Get("sqlite3").Get("capi").Call("sqlite3_column_type", rc.st.stmt, i).Int()
		// slog.Info("rs type", "type", defaultType)
		switch defaultType {
		case SQLITE_INTEGER:
			dest[i] = rc.st.stmt.Call("getInt", i).Int()
		case SQLITE3_TEXT:
			dest[i] = rc.st.stmt.Call("getString", i).String()
		case SQLITE_FLOAT:
			dest[i] = rc.st.stmt.Call("getFloat", i).Float()
		case SQLITE_BLOB:
			blob_js := rc.st.stmt.Call("getBlob", i)
			blob := make([]byte, blob_js.Length())
			js.CopyBytesToGo(blob, blob_js)
			dest[i] = blob
		}
	}
	return nil
}

func (rc *SQLiteRows) Close() error {
	rc.st.stmt.Call("reset")
	return nil
}
