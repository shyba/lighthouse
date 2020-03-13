package blocked

import (
	"context"
	"strings"

	"github.com/lbryio/lighthouse/app/es"
	"github.com/lbryio/lighthouse/app/model"

	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/lbryio/lbry.go/v2/extras/lbryinc"

	"github.com/sirupsen/logrus"
)

func ProcessedBlockedList() {
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
	err = p.Flush()
	if err != nil {
		logrus.Error(err)
	}
	err = p.Close()
	if err != nil {
		logrus.Error(err)
	}
}
