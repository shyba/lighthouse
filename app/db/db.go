package db

import (
	_ "github.com/go-sql-driver/mysql" // import mysql
	"github.com/jmoiron/sqlx"
	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/sirupsen/logrus"
)

var Chainquery *sqlx.DB

func Init(dsn string) {
	var err error
	Chainquery, err = dbInitConnection(dsn, "mysql")
	if err != nil {
		logrus.Panic(err)
	} else if Chainquery == nil {
		logrus.Panic("Chainquery connection could not be created")
	}
}

func dbInitConnection(dsn string, driverName string) (*sqlx.DB, error) {
	dsn += "?parseTime=1&collation=utf8mb4_unicode_ci"
	dbConn, err := sqlx.Connect(driverName, dsn)
	if err != nil {
		return nil, errors.Err(err)
	}

	err = dbConn.Ping()
	if err != nil {
		return nil, errors.Err(err)
	}

	return dbConn, nil
}
