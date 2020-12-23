package internalapis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/lbryio/lighthouse/app/es"
	"github.com/lbryio/lighthouse/app/es/index"
	"github.com/lbryio/lighthouse/app/model"
	"github.com/lbryio/lighthouse/app/util"

	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/lbryio/lbry.go/v2/extras/null"
	"github.com/sirupsen/logrus"
	"gopkg.in/olivere/elastic.v6"
)

func syncViewCounts() {
	s := elastic.NewSearchSource()
	s.Query(elastic.NewMatchAllQuery())
	s.FetchSourceContext(elastic.NewFetchSourceContext(false))
	s.Size(batchSize)
	scroll := es.Client.Scroll(index.Claims).SearchSource(s).Scroll("10m")
	p, err := es.Client.BulkProcessor().Name("IncSync").After(es.AfterBulkSend).Workers(2).Do(context.Background())
	if err != nil {
		logrus.Error(errors.Err(err))
		return
	}

	finished := false
	iteration := 0
	for !finished {
		result, err := scroll.Do(context.Background())
		if err != nil && !errors.Is(err, io.EOF) {
			logrus.Error(errors.Prefix(fmt.Sprintf("inc batch %d failed:", iteration+1), err))
			continue
		}
		scroll.ScrollId(result.ScrollId)
		claimIDs := make([]string, len(result.Hits.Hits))
		for i, h := range result.Hits.Hits {
			claimIDs[i] = h.Id
		}

		err = updateViewCounts(claimIDs, iteration, p)
		if err != nil {
			logrus.Error(err)
		}

		iteration++
		if iteration%10 == 0 {
			logrus.Debugf("Processed %d claims", iteration*batchSize)
		}
		finished = len(result.Hits.Hits) < batchSize
	}
	err = scroll.Clear(context.Background())
	if err != nil {
		logrus.Error(errors.Err(err))
	}
	err = p.Flush()
	if err != nil {
		logrus.Error(errors.Err(err))
	}
	err = p.Close()
	if err != nil {
		logrus.Error(errors.Err(err))
	}
}

func updateViewCounts(claimIDs []string, iteration int, p *elastic.BulkProcessor) error {
	if len(claimIDs) == 0 {
		logrus.Warningf("there are no claimids to update!")
		return nil
	}
	result, err := getViewCnts(claimIDs)
	if err != nil {
		return errors.Prefix(fmt.Sprintf("failed to get view counts at iteration %d: ", iteration), err)
	}
	if len(result) != len(claimIDs) {
		return errors.Err("sent %d claimIDs, returned array only has %d entries, failed to get views.", len(claimIDs), len(result))
	}
	for i, claimID := range claimIDs {
		if result[i] > 0 {
			logrus.Tracef("Found %d views for %s", result[i], claimID)
			count := null.Uint64From(uint64(result[i]))
			c := model.Claim{ClaimID: claimID, ViewCnt: &count}
			c.Update(p)
		}
	}
	return nil
}

type viewCntResponse struct {
	Success bool        `json:"success"`
	Error   interface{} `json:"error"`
	Data    []int64     `json:"data"`
}

func getViewCnts(claimIDs []string) ([]int64, error) {
	c := http.Client{}
	form := make(url.Values)
	form.Set("auth_token", APIToken)
	form.Set("claim_id", strings.Join(claimIDs, ","))

	response, err := c.PostForm(APIURL+"/file/view_count", form)
	if err != nil {
		return nil, errors.Err(err)
	}
	defer util.CloseBody(response.Body)
	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Err(err)
	}
	var me viewCntResponse
	err = json.Unmarshal(b, &me)
	if err != nil {
		return nil, errors.Err(err)
	}
	return me.Data, nil
}
