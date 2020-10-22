package internalapis

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/lbryio/lighthouse/app/internal/metrics"

	"github.com/lbryio/lighthouse/app/actions/search"
	"github.com/lbryio/lighthouse/app/db"
	"github.com/lbryio/lighthouse/app/es"
	"github.com/lbryio/lighthouse/app/es/index"
	"github.com/lbryio/lighthouse/app/model"
	"github.com/lbryio/lighthouse/app/util"

	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/lbryio/lbry.go/v2/extras/null"
	"github.com/lbryio/lbry.go/v2/extras/query"

	"github.com/sirupsen/logrus"
	"gopkg.in/olivere/elastic.v6"
)

var incSyncRunning bool

const batchSize = 1000

// Sync synchronizes view and subscription counts from internal-apis
func Sync() {
	if incSyncRunning || db.InternalAPIs == nil {
		return
	}
	metrics.JobLoad.WithLabelValues("internalapis_sync").Inc()
	defer metrics.JobLoad.WithLabelValues("internalapis_sync").Dec()
	defer metrics.Job(time.Now(), "internalapis_sync")
	incSyncRunning = true
	defer endIncSync()
	//syncViewCounts()
	//syncSubCounts()

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

func endIncSync() {
	incSyncRunning = false
}

func updateViewCounts(claimIDs []string, iteration int, p *elastic.BulkProcessor) error {
	if len(claimIDs) == 0 {
		logrus.Warningf("there are no claimids to update!")
		return nil
	}
	iSet := make([]interface{}, len(claimIDs))
	for i, c := range claimIDs {
		iSet[i] = c
	}
	q := fmt.Sprintf(`
			SELECT file.claim_id,count(file_view.id) 
			FROM file 
			INNER JOIN file_view ON file_view.file_id = file.id 
			WHERE file.claim_id IN (%s) 
			GROUP BY file.claim_id`, query.Qs(len(claimIDs)))

	rows, err := db.InternalAPIs.Query(q, iSet...)
	if err != nil {
		return errors.Prefix(fmt.Sprintf("inc call for batch %d failed:", iteration+1), err)

	}
	defer util.CloseRows(rows)
	type result struct {
		ClaimID string
		ViewCnt int64
	}
	vCntMap := make(map[string]int64)
	for rows.Next() {
		r := result{}
		err := rows.Scan(&r.ClaimID, &r.ViewCnt)
		if err != nil {
			return errors.Err(err)
		}
		vCntMap[r.ClaimID] = r.ViewCnt
	}
	for claimID, viewCount := range vCntMap {
		if viewCount > 0 {
			logrus.Debugf("Found %d views for %s", viewCount, claimID)
			count := null.Uint64From(uint64(viewCount))
			c := model.Claim{ClaimID: claimID, ViewCnt: &count}
			c.Update(p)
		}
	}
	return nil
}

func updateSubCounts(claimIDs []string, iteration int, p *elastic.BulkProcessor) error {
	if len(claimIDs) == 0 {
		logrus.Warningf("there are no claimids to update!")
		return nil
	}
	iSet := make([]interface{}, len(claimIDs))
	for i, c := range claimIDs {
		iSet[i] = c
	}
	q := fmt.Sprintf(`
			SELECT claim_id, COUNT(*) 
			FROM subscription 
			WHERE claim_id IN (%s) 
			GROUP BY claim_id`, query.Qs(len(claimIDs)))

	rows, err := db.InternalAPIs.Query(q, iSet...)
	if err != nil {
		return errors.Prefix(fmt.Sprintf("inc call for batch %d failed:", iteration+1), err)

	}
	defer util.CloseRows(rows)
	type result struct {
		ClaimID string
		SubCnt  int64
	}
	subCntMap := make(map[string]int64)
	for rows.Next() {
		r := result{}
		err := rows.Scan(&r.ClaimID, &r.SubCnt)
		if err != nil {
			return errors.Err(err)
		}
		subCntMap[r.ClaimID] = r.SubCnt
	}
	for claimID, subCount := range subCntMap {
		if subCount > 0 {
			logrus.Debugf("Found %d subscriptions for %s", subCount, claimID)
			count := null.Uint64From(uint64(subCount))
			c := model.Claim{ClaimID: claimID, SubCnt: &count}
			c.Update(p)
		}
	}
	return nil
}
