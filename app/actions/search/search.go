package search

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/lbryio/lighthouse/app/es"
	"github.com/lbryio/lighthouse/app/validator"

	"github.com/lbryio/lbry.go/v2/extras/api"
	"github.com/lbryio/lbry.go/v2/extras/errors"
	v "github.com/lbryio/ozzo-validation"

	"github.com/sirupsen/logrus"
	"gopkg.in/olivere/elastic.v6"
)

type searchRequest struct {
	S         string
	Size      *int
	From      *int
	Channel   *string
	ChannelID *string
	RelatedTo *string
	//Should change these calls in the app
	ContentType *string `json:"contentType"`
	MediaType   *string `json:"mediaType"`
	ClaimType   *string `json:"claimType"`
	NSFW        *bool
	//Debug params
	ClaimID *string
	Score   *bool
	Source  *bool
	Debug   *bool
}

// Search API returns the name and claim id of the results based on the query passed.
func Search(r *http.Request) api.Response {

	searchRequest := searchRequest{}

	err := api.FormValues(r, &searchRequest, []*v.FieldRules{
		v.Field(&searchRequest.S, v.Required),
		v.Field(&searchRequest.Size, v.Max(10000)),
		v.Field(&searchRequest.From, v.Max(9999)),
		//There is a bug in the app https://github.com/lbryio/lbry-desktop/issues/3377
		//v.Field(&searchRequest.ClaimType, validator.ClaimTypeValidator),
		v.Field(&searchRequest.MediaType, validator.MediaTypeValidator),
	})
	if err != nil {
		return api.Response{Error: errors.Err(err), Status: http.StatusBadRequest}
	}
	searchRequest.S = CheckForSpecialHandling(searchRequest.S)
	query := searchRequest.NewQuery()
	t, err := query.Source()
	if err != nil {
		return api.Response{Error: errors.Err("%s: for query -s %s", err, t)}
	}
	sourceContext := elastic.NewFetchSourceContext(true).Exclude("value")
	if searchRequest.Source == nil {
		sourceContext = sourceContext.Include("name", "claimId")
	}
	service := es.Client.
		Search("claims").
		Query(query).
		FetchSourceContext(sourceContext)
	if searchRequest.Size != nil {
		service = service.Size(*searchRequest.Size)
	}
	if searchRequest.From != nil {
		service = service.From(*searchRequest.From)
	}

	if searchRequest.Debug != nil {
		searchResults, err := service.
			Explain(true).
			ErrorTrace(true).
			Do(context.Background())
		if err != nil {
			return api.Response{Error: errors.Err(err)}
		}
		return api.Response{Data: searchResults}
	}
	searchResults, err := service.Do(context.Background())
	if err != nil {
		return api.Response{Error: errors.Err(err)}
	}
	results := make([]map[string]interface{}, 0)
	for _, hit := range searchResults.Hits.Hits {
		if hit.Source != nil {
			data, err := hit.Source.MarshalJSON()
			if err != nil {
				logrus.Error(err)
				continue
			}
			result := map[string]interface{}{}
			err = json.Unmarshal(data, &result)
			if err != nil {
				logrus.Error(err)
				continue
			}
			results = append(results, result)
		}
	}

	return api.Response{Data: results}
}
