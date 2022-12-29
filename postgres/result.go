package postgres

import "github.com/jackc/pgx/v5/pgconn"

type Result struct {
	pgconn.CommandTag
}

func newResult(commandTag pgconn.CommandTag) *Result {
	return &Result{CommandTag: commandTag}
}

func (r *Result) RowsAffected() (int64, error) {
	return r.CommandTag.RowsAffected(), nil
}
