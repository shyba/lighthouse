package config

import (
	"github.com/johntdyer/slackrus"
	"github.com/lbryio/lighthouse/app/db"
	"github.com/lbryio/lighthouse/app/env"
	"github.com/lbryio/lighthouse/app/es"
	"github.com/lbryio/lighthouse/app/jobs/chainquery"
	"github.com/lbryio/lighthouse/app/util"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// InitializeConfiguration inits the base configuration of lighthouse
func InitializeConfiguration() {
	config, err := env.NewWithEnvVars()
	if err != nil {
		logrus.Panic(err)
	}
	InitSlack(config)
	db.InitChainquery(config.ChainQueryDsn)
	//db.InitInternalAPIs(config.InternalAPIDSN)
	es.ElasticSearchURL = config.ElasticSearchURL
	chainquery.SyncStateDir = config.SyncStateDir
	if viper.GetBool("debugmode") {
		util.Debugging = true
		logrus.SetLevel(logrus.DebugLevel)
	}
	if viper.GetBool("tracemode") {
		util.Debugging = true
		logrus.SetLevel(logrus.TraceLevel)
	}

}

// InitSlack initializes the slack connection and posts info level or greater to the set channel.
func InitSlack(config *env.Config) {
	slackURL := config.SlackHookURL
	slackChannel := config.SlackChannel
	if slackURL != "" && slackChannel != "" {
		logrus.AddHook(&slackrus.SlackrusHook{
			HookURL:        slackURL,
			AcceptedLevels: slackrus.LevelThreshold(logrus.InfoLevel),
			Channel:        slackChannel,
			IconEmoji:      ":lighthouse:",
			Username:       "Lighthouse",
		})
	}
}
