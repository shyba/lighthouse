package app

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/lbryio/lighthouse/app/actions"
	"github.com/lbryio/lighthouse/app/es"
	"github.com/lbryio/lighthouse/app/es/index"
	"github.com/lbryio/lighthouse/app/util"

	"github.com/lbryio/lbry.go/v2/extras/api"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/olivere/elastic.v6"
)

//DoYourThing launches the app
func DoYourThing() {
	initElasticSearch()
	initAPIServer()
}

func initElasticSearch() {
	opts := []elastic.ClientOptionFunc{elastic.SetErrorLog(logrus.StandardLogger())}
	if es.ElasticSearchURL != "" {
		opts = append(opts, elastic.SetURL(es.ElasticSearchURL))
	}
	if util.Debugging {
		opts = append(opts, elastic.SetInfoLog(logrus.StandardLogger()))
	}
	if viper.GetBool("tracemode") {
		// Uncomment next line to show request/response to/from Elasticsearch
		opts = append(opts, elastic.SetTraceLog(logrus.StandardLogger()))

	}
	client, err := elastic.NewClient(opts...)
	if err != nil {
		panic(err)
	}
	client.Start()
	es.Client = client
	exists, err := client.IndexExists(index.Claims).Do(context.Background())
	if err != nil {
		logrus.Panic(err)
	}
	if !exists {
		_, err := client.CreateIndex(index.Claims).BodyString(index.ClaimMapping).Do(context.Background())
		if err != nil {
			logrus.Panic(err)
		}
	}
}

func initAPIServer() {
	host := viper.GetString("host")
	port := viper.GetInt("port")
	logrus.Infof("API Server started @ %s", "http://"+host+":"+viper.GetString("port")+"/search?s=test")
	hs := make(map[string]string)
	hs["Access-Control-Allow-Methods"] = "GET, POST, OPTIONS"
	hs["Content-Type"] = "application/json; charset=utf-8; application/x-www-form-urlencoded"
	hs["X-Content-Type-Options"] = "nosniff"
	hs["Content-Security-Policy"] = "default-src 'none'"
	hs["Server"] = "lbry.com"
	hs["Access-Control-Allow-Origin"] = "*"
	api.ResponseHeaders = hs
	api.Log = func(request *http.Request, response *api.Response, err error) {
		consoleText := request.RemoteAddr + " [" + strconv.Itoa(response.Status) + "]: " + request.Method + " " + request.URL.Path
		if err == nil {
			logrus.Debug(color.GreenString(consoleText))
		} else {
			logrus.Error(color.RedString(consoleText + ": " + err.Error()))
			if response.Status >= http.StatusInternalServerError && !util.Debugging {
				//util.SendToSlack(strconv.Itoa(response.Status) + " " + request.Method + " " + request.URL.Path + ": " + errors.FullTrace(response.Error))
			}
		}
	}

	httpServeMux := http.NewServeMux()
	actions.GetRoutes().Each(func(pattern string, handler http.Handler) {
		httpServeMux.Handle(pattern, handler)
	})
	api.BuildJSONResponse = func(response api.ResponseInfo) ([]byte, error) {
		if response.Error != nil {
			return json.MarshalIndent(&response, "", "  ")
		}
		return json.MarshalIndent(&response.Data, "", "  ")
	}
	mux := http.Handler(httpServeMux)
	logrus.Fatal(http.ListenAndServe(host+":"+strconv.Itoa(port), mux))
}
