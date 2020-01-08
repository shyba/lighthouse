package es

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/olivere/elastic.v6"
)

// Client is the elasticsearch client created on lighthouse startup and is used to make queries to the db.
var Client *elastic.Client

// ElasticSearchURL is the url that the client uses to connect with. This can be overriden with then respective
// env var
var ElasticSearchURL string

// AfterBulkSend checks for errors in bulk processing a logs them.
func AfterBulkSend(executionID int64, requests []elastic.BulkableRequest, response *elastic.BulkResponse, err error) {
	if response.Errors {
		for _, failure := range response.Failed() {
			logrus.Error(failure.Error)
		}
	}
}
