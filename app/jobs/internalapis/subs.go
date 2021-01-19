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

	"github.com/lbryio/lighthouse/app/actions/search"
	"github.com/lbryio/lighthouse/app/es"
	"github.com/lbryio/lighthouse/app/es/index"
	"github.com/lbryio/lighthouse/app/model"
	"github.com/lbryio/lighthouse/app/util"

	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/lbryio/lbry.go/v2/extras/null"

	"github.com/sirupsen/logrus"
	"github.com/olivere/elastic/v7"
)

type subCntResponse struct {
	Success bool             `json:"success"`
	Error   interface{}      `json:"error"`
	Data    map[string]int64 `json:"data"`
}

func syncSubCounts() {
	s := elastic.NewSearchSource()
	s.Query(search.ChannelOnlyMatch)
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
		claimIds := make([]string, len(result.Hits.Hits))
		for i, h := range result.Hits.Hits {
			claimIds[i] = h.Id
		}

		err = updateSubCounts(claimIds, iteration, p)
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

func updateSubCounts(claimIDs []string, iteration int, p *elastic.BulkProcessor) error {

	if len(claimIDs) == 0 {
		logrus.Warningf("there are no claimids to update!")
		return nil
	}
	result, err := getSubCnts(claimIDs)
	if err != nil {
		return err
	}
	logrus.Debugf("found subs for %d claims", len(result))
	for _, claimID := range claimIDs {
		cnt, ok := result[claimID]
		if ok && cnt > 0 {
			logrus.Tracef("Found %d subscriptions for %s", cnt, claimID)
			count := null.Uint64From(uint64(cnt))
			c := model.Claim{ClaimID: claimID, SubCnt: &count}
			c.Update(p)
		}

	}

	return nil
}

func getSubCnts(claimIDs []string) (map[string]int64, error) {
	c := http.Client{}
	form := make(url.Values)
	form.Set("auth_token", APIToken)
	form.Set("is_map", "true")
	form.Set("claim_id", strings.Join(claimIDs, ","))

	response, err := c.PostForm(APIURL+"/subscription/sub_count", form)
	if err != nil {
		return nil, errors.Err(err)
	}
	defer util.CloseBody(response.Body)
	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Err(err)
	}
	var me subCntResponse
	err = json.Unmarshal(b, &me)
	if err != nil {
		return nil, errors.Err(err)
	}
	return me.Data, nil
}
