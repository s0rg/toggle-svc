//nolint:goimports
package db

import (
	"database/sql"
	// driver.
	_ "github.com/lib/pq"
	"io"
)

type app interface {
	GetEnv(string) string
	DeferClose(io.Closer)
}

// ForApp return new *sql.DB (postgres driver)
//
// key is a dependency (see app.GetEnv), that holds connection dsn
//
// this *sql.DB will be closed upon app.Close() invocation.
func ForApp(app app, key string) (*sql.DB, error) {
	db, err := sql.Open("postgres", app.GetEnv(key))
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	app.DeferClose(db)

	return db, nil
}
