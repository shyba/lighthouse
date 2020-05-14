package blocked

import (
	"context"
	"strings"

	"github.com/lbryio/lighthouse/app/db"
	"github.com/lbryio/lighthouse/app/es"
	"github.com/lbryio/lighthouse/app/model"

	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/lbryio/lbry.go/v2/extras/lbryinc"

	"github.com/sirupsen/logrus"
)

var blockedChannels = []string{
	"565be843d5f231d37a037ee6d5276dc1618b5ca3",
	"3dc1703d218fdc6c1cdaa1b32dbd6c143554ba4b",
	"b8b4f68a4e9d9189552e70c508c92cf7b52e9763",
}

// ProcessBlockedList runs through the current blocked list and tries to delete the entry if it exists.
func ProcessBlockedList() {
	c := lbryinc.NewClient("", nil)
	r, err := c.Call("file", "list_blocked", nil)
	if err != nil {
		logrus.Error(errors.Err(err))
		return
	}
	data, ok := r["outpoints"]
	if !ok {
		logrus.Error("Could not grab outputs from return for blocked list")
		return
	}
	outpoints, ok := data.([]interface{})
	if !ok {
		logrus.Error("Could not convert data to string array")
		return
	}
	p, err := es.Client.BulkProcessor().Name("ClaimSync").After(es.AfterBulkSend).Workers(4).Do(context.Background())
	if err != nil {
		logrus.Error(errors.Err(err))
		return
	}
	for _, value := range outpoints {
		outpoint, ok := value.(string)
		if !ok {
			logrus.Error("Could not convert outpoint to string")
			continue
		}
		claimID := strings.Split(outpoint, ":")[0]
		claim := model.NewClaim()
		claim.ClaimID = claimID
		claim.Delete(p)
	}
	for _, channel := range blockedChannels {
		rows, err := db.Chainquery.Query("SELECT claim_id FROM claim WHERE publisher_id =?", channel)
		if err != nil {
			logrus.Error(errors.Err(err))
		}
		for rows.Next() {
			claim := model.NewClaim()
			err := rows.Scan(&claim.ClaimID)
			if err != nil {
				logrus.Error(errors.Err(err))
				continue
			}
			claim.Delete(p)
		}
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
