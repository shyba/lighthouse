package util

import (
	"database/sql"
	"io"

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

// CloseBody closes the body of an http response
func CloseBody(responseBody io.ReadCloser) {
	if err := responseBody.Close(); err != nil {
		closeBodyError := errors.Prefix("closing body if response error: ", errors.Err(err))
		log.Error(closeBodyError)
	}
}
