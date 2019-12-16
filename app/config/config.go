package config

import (
	"github.com/lbryio/lighthouse/app/db"
	"github.com/lbryio/lighthouse/app/env"
	"github.com/lbryio/lighthouse/app/es"
	"github.com/lbryio/lighthouse/app/jobs"
	"github.com/lbryio/lighthouse/app/util"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func InitializeConfiguration() {
	config, err := env.NewWithEnvVars()
	if err != nil {
		logrus.Panic(err)
	}
	db.Init(config.ChainQueryDsn)
	es.ElasticSearchURL = config.ElasticSearchURL
	jobs.SyncStateDir = config.SyncStateDir
	if viper.GetBool("debugmode") {
		util.Debugging = true
		logrus.SetLevel(logrus.DebugLevel)
	}
	if viper.GetBool("tracemode") {
		util.Debugging = true
		logrus.SetLevel(logrus.TraceLevel)
	}

}
