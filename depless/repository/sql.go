package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lib/pq"
)

type SqlDB struct {
	cfg      ConfigSQL
	db       *sql.DB
	pool     *sync.Pool
	poolSize atomic.Int64
}

type ConfigSQL struct {
	DSN         string
	URI         string
	Host        string
	Port        string
	User        string
	Password    string
	Name        string
	SSLMode     string
	MaxIdleTime time.Duration
}

func (cfg ConfigSQL) getDSN() (string, error) {
	if cfg.DSN != "" {
		return cfg.DSN, nil
	}
	if cfg.URI != "" {
		return pq.ParseURL(cfg.URI)
	}
	params := make(map[string]string)

	params["host"] = cfg.Host
	params["port"] = cfg.Port
	if cfg.User != "" && cfg.Password != "" {
		params["user"] = cfg.User
		params["password"] = cfg.Password
	}
	params["dbname"] = cfg.Name
	params["sslmode"] = cfg.SSLMode

	values := make([]string, 0, len(params))

	for k, v := range params {
		values = append(values, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(values, " "), nil
}

func NewDB(cfg ConfigSQL) (*SqlDB, error) {
	s := &SqlDB{
		cfg: cfg,
	}

	s.pool = &sync.Pool{
		New: func() any {
			db, err := s.connect()
			if err == nil {
				s.poolSize.Add(1)
				go s.handleReconnect()
			}

			return db
		},
	}
	for i := 0; i < 100; i++ {
		s.pool.Get()
	}
	var err error
	if s.db, err = s.connect(); err != nil {
		return nil, err
	}

	return s, nil
}

const (
	connectionCheckInterval = 15 * time.Second
)

func (sdb *SqlDB) reconnect(db *sql.DB) {
	var err error
	if err = db.Ping(); err != nil {
		return
	}
	for {
		if db, err = sdb.connect(); err == nil {
			break
		}
	}
}

func (sdb *SqlDB) handleReconnect() {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for range timer.C {
		sdb.reconnect(sdb.db)

		for i := 0; i < int(sdb.poolSize.Load()); i++ {
			db := sdb.pool.Get().(*sql.DB)
			sdb.reconnect(db)
			sdb.pool.Put(db)
		}

		timer.Reset(connectionCheckInterval)
	}
}

func (sdb *SqlDB) connect() (db *sql.DB, err error) {
	var dsn string
	if dsn, err = sdb.cfg.getDSN(); err != nil {
		return nil, err
	}
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		return db, err
	}
	db.SetConnMaxIdleTime(sdb.cfg.MaxIdleTime)

	return db, err
}

func (sdb *SqlDB) getDB() (*sql.DB, func()) {
	db := sdb.pool.Get().(*sql.DB)
	if db == nil {
		return sdb.db, func() {}
	}
	return db, func() {
		sdb.pool.Put(db)
	}
}

func (sdb *SqlDB) Pipe(ctx context.Context, opts *sql.TxOptions, pipe func(tx *sql.Tx) error) error {
	db, put := sdb.getDB()
	defer put()

	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return err
	}
	if err = pipe(tx); err != nil {
		return errors.Join(err, tx.Rollback())
	}

	return tx.Commit()
}

func (sdb *SqlDB) Exec(query string, args ...any) (sql.Result, error) {
	db, put := sdb.getDB()
	defer put()

	return db.Exec(query, args...)
}

func (sdb *SqlDB) Query(query string, args ...any) (*sql.Rows, error) {
	db, put := sdb.getDB()
	defer put()

	return db.Query(query, args...)
}

func (sdb *SqlDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	db, put := sdb.getDB()
	defer put()

	return db.ExecContext(ctx, query, args...)
}

func (sdb *SqlDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	db, put := sdb.getDB()
	defer put()

	return db.QueryContext(ctx, query, args...)
}

func (sdb *SqlDB) QueryRow(query string, args ...any) *sql.Row {
	db, put := sdb.getDB()
	defer put()

	return db.QueryRow(query, args...)
}

func (sdb *SqlDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	db, put := sdb.getDB()
	defer put()

	return db.BeginTx(ctx, opts)
}
