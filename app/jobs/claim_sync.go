package jobs

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"time"

	"github.com/lbryio/lighthouse/app/db"
	"github.com/lbryio/lighthouse/app/es"
	"github.com/lbryio/lighthouse/app/es/index"

	"github.com/lbryio/lbry.go/extras/errors"
	"github.com/lbryio/lbry.go/v2/extras/null"

	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"gopkg.in/olivere/elastic.v6"
)

var claimSyncRunning bool
var batchSize = 1000
var maxClaimsToProcessPerIteration = 5000

// SyncStateDir holds the direction location of where to store the sync state json file.
var SyncStateDir string

const query = `
SELECT c.id, 
	c.name, 
	p.name as channel, 
	p.claim_id as channel_id, 
	c.bid_state, 
	c.effective_amount, 
	c.transaction_time, 
	COALESCE(p.effective_amount,1) as certificate_amount, 
	c.claim_id as claimId, 
	c.value_as_json as value 
FROM claim c LEFT JOIN claim p on p.claim_id = c.publisher_id 
WHERE c.id > ? 
AND c.modified_at >= ? 
ORDER BY c.id 
LIMIT ?`

func SyncClaims() {
	if claimSyncRunning {
		return
	}
	claimSyncRunning = true
	defer endClaimSync()
	logrus.Debugf("running claim sync job...")
	syncState, err := loadSynState()
	if err != nil {
		logrus.Error(err)
		return
	}
	if syncState.StartSyncTime.IsZero() || syncState.LastID == 0 {
		syncState.StartSyncTime = time.Now()
	}
	p, err := es.Client.BulkProcessor().Name("ClaimSync").After(AfterBulkSend).Workers(4).Do(context.Background())
	if err != nil {
		logrus.Error(errors.Err(err))
		return
	}
	finished := false
	iteration := 0
	for !finished {
		rows, err := db.Chainquery.Query(query, syncState.LastID, syncState.LastSyncTime, batchSize)
		if err != nil {
			logrus.Error(err)
			return
		}
		claims := make([]claimInfo, 0, batchSize)
		for rows.Next() {
			claim := claimInfo{}
			err := rows.Scan(
				&claim.ID,
				&claim.Name,
				&claim.Channel,
				&claim.ChannelClaimID,
				&claim.BidState,
				&claim.EffectiveAmount,
				&claim.TransactionTimeUnix,
				&claim.ChannelEffectiveAmount,
				&claim.ClaimID,
				&claim.JSONValue)
			if err != nil {
				logrus.Error(err)
			}
			value := map[string]interface{}{}
			err = json.Unmarshal([]byte(claim.JSONValue.String), &value)
			if err != nil {
				logrus.Error(errors.Prefix("could not parse json for value: ", err))
			}
			claim.Value = value
			syncState.LastID = int(claim.ID)
			claims = append(claims, claim)
		}
		for _, claim := range claims {
			if claim.JSONValue.IsNull() {
				logrus.Debug("Claim: ", claim.AsJSON())
				logrus.Error("Failed to process claim ", claim.ClaimID, " due to missing value")
				continue
			}
			claim.TransactionTime = time.Unix(int64(claim.TransactionTimeUnix), 0)
			if claim.BidState == "Spent" || claim.BidState == "Expired" {
				claim.Delete(p)
			} else {
				claim.Add(p)
			}

		}
		logrus.Debugf("Processed %d claims", len(claims))
		finished = len(claims) < batchSize || (iteration*batchSize+batchSize >= maxClaimsToProcessPerIteration)
		iteration++
	}

	// If not finished, store last id to run again later where we left off, otherwise update last sync time.
	if iteration*batchSize+batchSize >= maxClaimsToProcessPerIteration {
	} else {
		syncState.LastID = 0
		syncState.LastSyncTime = syncState.StartSyncTime
	}

	err = syncState.Save()
	if err != nil {
		logrus.Error(err)
	}
	err = p.Flush()
	if err != nil {
		logrus.Error(err)
	}
	err = p.Close()
	if err != nil {
		logrus.Error(err)
	}
}

func AfterBulkSend(executionId int64, requests []elastic.BulkableRequest, response *elastic.BulkResponse, err error) {
	if response.Errors {
		for _, failure := range response.Failed() {
			logrus.Error(failure.Error)
		}
	}
}

func endClaimSync() {
	claimSyncRunning = false
	syncState, _ := loadSynState()
	if syncState != nil && syncState.LastID != 0 {
		go SyncClaims()
	}
}

type claimInfo struct {
	ID                     uint64                 `json:"id"`
	Name                   string                 `json:"name"`
	ClaimID                string                 `json:"claimId"`
	Channel                null.String            `json:"channel,omitempty"`
	ChannelClaimID         null.String            `json:"channel_claim_id,omitempty"`
	BidState               string                 `json:"bid_state"`
	EffectiveAmount        uint64                 `json:"effective_amount"`
	TransactionTimeUnix    uint64                 `json:"-"`
	TransactionTime        time.Time              `json:"transaction_time"`
	ChannelEffectiveAmount uint64                 `json:"certificate_amount"`
	JSONValue              null.String            `json:"-"`
	Value                  map[string]interface{} `json:"value"`
	SuggestName            struct {
		Input  string `json:"input"`
		Weight uint64 `json:"weight"`
	} `json:"suggest_name"`
}

func (c claimInfo) Add(p *elastic.BulkProcessor) {
	r := elastic.NewBulkIndexRequest().Index(index.Claims).Type(index.ClaimType).Id(c.ClaimID).Doc(c)
	p.Add(r)
}

func (c claimInfo) Delete(p *elastic.BulkProcessor) {
	r := elastic.NewBulkDeleteRequest().Index(index.Claims).Type(index.ClaimType).Id(c.ClaimID)
	p.Add(r)
}

type claimSyncState struct {
	StartSyncTime time.Time `json:"StartSyncTime"`
	LastSyncTime  time.Time `json:"LastSyncTime"`
	LastID        int       `json:"LastID"`
}

func (c claimSyncState) Save() error {
	data, err := json.Marshal(c)
	if err != nil {
		return errors.Err(err)
	}
	err = ioutil.WriteFile(SyncStateDir+"/syncstate.json", data, 0644)
	if err != nil {
		return errors.Err(err)
	}
	return nil
}

func (c claimInfo) AsJSON() string {
	data, err := json.Marshal(&c)
	if err != nil {
		logrus.Error(errors.Err(err))
		return ""
	}
	return string(data)

}

func loadSynState() (*claimSyncState, error) {
	if SyncStateDir == "" {
		var err error
		SyncStateDir, err = homedir.Dir()
		logrus.Debug("Home Dir: ", SyncStateDir)
		if err != nil {
			return nil, errors.Err(err)
		}

	}

	data, err := ioutil.ReadFile(SyncStateDir + "/syncstate.json")
	if err != nil {
		if !os.IsExist(err) {
			return &claimSyncState{}, nil
		}
		return nil, errors.Err(err)
	}
	state := &claimSyncState{}
	return state, json.Unmarshal(data, state)
	//"LastSyncTime":"2019-10-04 00:50:19","LastID":0,"StartSyncTime":"2019-10-04 00:50:19"}
}
