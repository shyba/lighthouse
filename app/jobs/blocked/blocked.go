package blocked

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"gopkg.in/olivere/elastic.v6"

	"github.com/lbryio/lighthouse/app/db"
	"github.com/lbryio/lighthouse/app/es"
	"github.com/lbryio/lighthouse/app/model"
	"github.com/lbryio/lighthouse/app/util"

	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/lbryio/lbry.go/v2/extras/lbryinc"

	"github.com/sirupsen/logrus"
)

var blockedChannels = []string{
	"565be843d5f231d37a037ee6d5276dc1618b5ca3",
	"3dc1703d218fdc6c1cdaa1b32dbd6c143554ba4b",
	"b8b4f68a4e9d9189552e70c508c92cf7b52e9763",
	"565be843d5f231d37a037ee6d5276dc1618b5ca3",
	"3dc1703d218fdc6c1cdaa1b32dbd6c143554ba4b",
	"b8b4f68a4e9d9189552e70c508c92cf7b52e9763",
	"6be2cbc811bf3106c51ebaf154442d1d231a0104",
	"0bc958169c77733b5d22bcc860e426713c9b6fda",
	"5894d3c795fc475b23fbb4e5dca1b59cd6222254",
	"1e2c80f572c79b91ed4801932da6e6b2c95545ac",
	"b5de24be04dcbef2becdcbbcdf32fcd4ed61ca4d",
	"9fc0341d2c44a0c2177983cd56ae57c7bf6f35b9",
	"0db48d6ae29035a62ce460ac40b7b05adab99c29",
	"521dca9f2cdc2962c37237d09c8126fa39c56e1b",
	"aa1b30af054fabf370fe5ca265296e9354844847",
	"1092b755f939893d459dc8b136e8be2b82ecf4ca",
	"7685064d832ced149c5c04e95bcebe5f005d2c46",
}

var blockedClaims []string

// ProcessBlockedList removes any claims and channels associated with the blocked list
func ProcessBlockedList() {
	processListForRemoval("list_blocked")
}

// ProcessFilteredList removes any claims and channels associated with the filtered list
func ProcessFilteredList() {
	processListForRemoval("list_filtered")
}

// processListForRemoval runs through the passed list and tries to delete the entry if it exists or if its a channel to
// delete the claims associated with it from the lighthouse elastic db.
func processListForRemoval(list string) {
	c := lbryinc.NewClient("", nil)
	r, err := c.Call("file", list, nil)
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
		split := strings.Split(outpoint, ":")
		txID := split[0]
		vout := int64(0)
		if len(split) > 0 {
			voutStr := split[1]
			vout, err = strconv.ParseInt(voutStr, 10, 64)
			if err != nil {
				logrus.Errorf("Could not convert outpoint vout to int64[%s]: %s ", outpoint, err.Error())
				continue
			}
		}
		var claimID string
		result := db.Chainquery.QueryRow("SELECT claim_id FROM claim WHERE transaction_hash_update =? AND vout_update=?", txID, vout)
		err := result.Scan(&claimID)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				logrus.Errorf("Could not grab claimID of outpoint from chainquery[%s]: %s", outpoint, err.Error())
			}
			continue
		}
		//If its a channel that is blocked, remove all of its claims as well.
		rows, err := db.Chainquery.Query("SELECT claim_id FROM claim WHERE publisher_id =?", claimID)
		if err != nil {
			logrus.Error(errors.Err(err))
			continue
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
		util.CloseRows(rows)
		claim := model.NewClaim()
		claim.ClaimID = claimID
		claim.Delete(p)
	}
	removeBlockedChannels(p)
	removedBlockedClaims(p)
	err = p.Flush()
	if err != nil {
		logrus.Error(err)
	}
	err = p.Close()
	if err != nil {
		logrus.Error(err)
	}
}

func removeBlockedChannels(p *elastic.BulkProcessor) {
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
		util.CloseRows(rows)
	}
}

func removedBlockedClaims(p *elastic.BulkProcessor) {
	for _, channel := range blockedClaims {
		claim := model.NewClaim()
		claim.ClaimID = channel
		claim.Delete(p)
	}
}
