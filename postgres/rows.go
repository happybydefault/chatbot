package postgres

import "github.com/jackc/pgx/v5"

type Rows struct {
	pgx.Rows
}

func newRows(rows pgx.Rows) *Rows {
	return &Rows{Rows: rows}
}

func (r *Rows) Close() error {
	r.Rows.Close()
	return nil
}
