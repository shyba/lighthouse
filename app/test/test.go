package test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/sirupsen/logrus"
)

type searchResult struct {
	Name    string
	ClaimID string `json:"claimId"`
}

func RunTests() {
	results := make([]searchResult, 0)

	resp, err := http.Get("http://0.0.0.0:50005/search?s=" + url.QueryEscape("interesting and amazing facts") + "&size=1")
	if err != nil {
		logrus.Fatalf("search %s failed with %s", "interesting and amazing facts", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Fatalf("search %s failed with %s", "interesting and amazing facts", err)
	}
	err = json.Unmarshal(body, &results)
	if err != nil {
		logrus.Fatalf("search %s failed with %s", "interesting and amazing facts", err)
	}
	logrus.Info(results)

}
