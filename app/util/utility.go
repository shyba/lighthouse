package util

import (
	"database/sql"

	"github.com/lbryio/lbry.go/extras/errors"
	"github.com/prometheus/common/log"
)

// Debugging is a variable to tell whether lighthouse is in debug mode
var Debugging bool

//CloseRows Closes SQL Rows for custom SQL queries.
func CloseRows(rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		closeRowsError := errors.Prefix("error closing rows: ", errors.Err(err))
		log.Error(closeRowsError)
	}
}
