package actions

import (
	"context"
	"net/http"

	"github.com/lbryio/lighthouse/meta"

	"github.com/lbryio/lighthouse/app/es"
	"github.com/lbryio/lighthouse/app/es/index"
	"gopkg.in/olivere/elastic.v6"

	"github.com/lbryio/lbry.go/v2/extras/api"
	"github.com/lbryio/lbry.go/v2/extras/errors"
)

type status struct {
	Version         string
	SemanticVersion string
	VersionLong     string
	VersionMsg      string
	Health          elastic.CatHealthResponse
	ClaimCount      elastic.CatCountResponse
	Allocations     elastic.CatAllocationResponse
	ClaimStats      elastic.IndicesStatsResponse
}

func Status(r *http.Request) api.Response {
	health, err := es.Client.CatHealth().Do(context.Background())
	if err != nil {
		return api.Response{Error: errors.Err(err)}
	}
	counts, err := es.Client.CatCount().Index(index.Claims).Do(context.Background())
	if err != nil {
		return api.Response{Error: errors.Err(err)}
	}
	alloc, err := es.Client.CatAllocation().Human(true).Do(context.Background())
	if err != nil {
		return api.Response{Error: errors.Err(err)}
	}
	claimStats, err := es.Client.IndexStats(index.Claims).Human(true).Do(context.Background())
	if err != nil {
		return api.Response{Error: errors.Err(err)}
	}

	return api.Response{Data: status{
		Version:         meta.GetVersion(),
		SemanticVersion: meta.GetSemVersion(),
		VersionLong:     meta.GetVersionLong(),
		VersionMsg:      meta.GetCommitMessage(),
		Health:          health,
		ClaimCount:      counts,
		Allocations:     alloc,
		ClaimStats:      *claimStats}}
}
