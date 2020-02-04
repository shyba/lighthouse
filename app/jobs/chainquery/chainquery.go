package chainquery

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/lbryio/lighthouse/app/db"
	"github.com/lbryio/lighthouse/app/es"
	"github.com/lbryio/lighthouse/app/model"

	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/lbryio/lbry.go/v2/extras/null"
	"github.com/lbryio/lbry.go/v2/extras/util"

	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
)

var claimSyncRunning bool
var batchSize = 1000
var maxClaimsToProcessPerIteration = 5000

// SyncStateDir holds the direction location of where to store the sync state json file.
var SyncStateDir string

func query(channelID *string) string {
	channelFilter := ""
	if channelID != nil {
		channelFilter = ` AND p.claim_id = "` + util.StrFromPtr(channelID) + `" `
	}
	var query = `
SELECT c.id, 
	c.name, 
	p.name as channel, 
	p.claim_id as channel_id, 
	c.bid_state, 
	c.effective_amount, 
	c.transaction_time, 
	COALESCE(p.effective_amount,1) as certificate_amount, 
	c.claim_id as claimId,
	c.value_as_json as value,
    c.title,
    c.description,
    c.release_time,
    c.content_type,
    c.is_cert_valid,
    c.type,
    c.frame_width,
    c.frame_height,
    c.duration,
    c.is_nsfw,
	c.thumbnail_url,
	c.fee,
 	GROUP_CONCAT(t.tag) as tags 
FROM claim c LEFT JOIN claim p on p.claim_id = c.publisher_id 
LEFT JOIN claim_tag ct ON ct.claim_id = c.claim_id 
LEFT JOIN tag t ON ct.tag_id = t.id 
WHERE c.id > ? ` + channelFilter + `
AND c.modified_at >= ? 
GROUP BY c.id 
ORDER BY c.id 
LIMIT ?`
	return query
}

// Sync uses Chainquery to sync the claim information to the elasticsearch db.
func Sync(channelID *string) {
	if claimSyncRunning {
		return
	}
	claimSyncRunning = true
	defer endClaimSync(channelID)
	logrus.Debugf("running claim sync job...")
	syncState, err := loadSynState()
	if err != nil {
		logrus.Error(err)
		return
	}
	if syncState.StartSyncTime.IsZero() || syncState.LastID == 0 {
		syncState.StartSyncTime = time.Now()
	}
	p, err := es.Client.BulkProcessor().Name("ClaimSync").After(es.AfterBulkSend).Workers(4).Do(context.Background())
	if err != nil {
		logrus.Error(errors.Err(err))
		return
	}
	finished := false
	iteration := 0
	for !finished {
		rows, err := db.Chainquery.Query(query(channelID), syncState.LastID, syncState.LastSyncTime, batchSize)
		if err != nil {
			logrus.Error(errors.Prefix("Chainquery Err:", err))
			return
		}
		var claims []model.Claim
		claims, syncState.LastID, err = model.GetClaimsFromDBRows(rows)
		for _, claim := range claims {
			if claim.JSONValue.IsNull() {
				logrus.Debug("Claim: ", claim.AsJSON())
				logrus.Error("Failed to process claim ", claim.ClaimID, " due to missing value")
				continue
			}
			txTime := null.NewTime(time.Unix(int64(claim.TransactionTimeUnix.Uint64), 0), true)
			claim.TransactionTime = &txTime
			releaseTime := null.NewTime(time.Unix(int64(claim.ReleaseTimeUnix.Uint64), 0), true)
			claim.ReleaseTime = &releaseTime
			if claim.ReleaseTimeUnix.IsNull() {
				claim.ReleaseTime = claim.TransactionTime
			}
			claim.Tags = strings.Split(claim.TagsStr.String, ",")
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

	// If not finished, store last id to run again later where we left off, otherwise Update last sync time.
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

func endClaimSync(channelID *string) {
	claimSyncRunning = false
	syncState, _ := loadSynState()
	if syncState != nil && syncState.LastID != 0 {
		go Sync(channelID)
	}
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
