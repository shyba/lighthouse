package search

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lbryio/lighthouse/app/es"
	"github.com/lbryio/lighthouse/app/internal/metrics"
	"github.com/lbryio/lighthouse/app/validator"

	"github.com/lbryio/lbry.go/v2/extras/api"
	"github.com/lbryio/lbry.go/v2/extras/errors"
	v "github.com/lbryio/ozzo-validation"

	"github.com/karlseguin/ccache"
	"github.com/sirupsen/logrus"
	"gopkg.in/olivere/elastic.v6"
)

var searchCache = ccache.New(ccache.Configure().MaxSize(10000))

type searchRequest struct {
	S         string
	Size      *int
	From      *int
	Channel   *string
	ChannelID *string
	RelatedTo *string
	SortBy    *string
	Include   *string
	//Should change these calls in the app
	ContentType *string `json:"contentType"`
	MediaType   *string `json:"mediaType"`
	ClaimType   *string `json:"claimType"`
	NSFW        *bool
	FreeOnly    *bool
	Resolve     bool
	//Debug params
	ClaimID    *string
	Score      bool
	Source     bool
	Debug      bool
	searchType string
	terms      int
}

// Search API returns the name and claim id of the results based on the query passed.
func Search(r *http.Request) api.Response {
	start := time.Now()
	searchRequest := searchRequest{}

	err := api.FormValues(r, &searchRequest, []*v.FieldRules{
		v.Field(&searchRequest.S, v.Length(3, 99999), v.Required),
		v.Field(&searchRequest.Size, v.Max(10000)),
		v.Field(&searchRequest.From, v.Max(9999)),
		//There is a bug in the app https://github.com/lbryio/lbry-desktop/issues/3377
		//v.Field(&searchRequest.ClaimType, validator.ClaimTypeValidator),
		v.Field(&searchRequest.MediaType, validator.MediaTypeValidator),
	})
	if err != nil {
		return api.Response{Error: errors.Err(err), Status: http.StatusBadRequest}
	}
	searchRequest.searchType = "general"
	searchRequest.S = truncate(searchRequest.S)
	searchRequest.S = checkForSpecialHandling(searchRequest.S)
	searchRequest.terms = len(strings.Split(searchRequest.S, " "))
	if searchRequest.RelatedTo != nil {
		searchRequest.searchType = "related_content"
	}
	query := searchRequest.newQuery()
	t, err := query.Source()
	if err != nil {
		return api.Response{Error: errors.Err("%s: for query -s %s", err, t)}
	}
	includes := []string{"name", "claimId"}
	if searchRequest.Include != nil {
		additionfields := strings.Split(*searchRequest.Include, ",")
		includes = append(includes, additionfields...)
	}
	sourceContext := elastic.NewFetchSourceContext(true).Exclude("value")
	if !searchRequest.Source {
		includes = append(includes)
		sourceContext = sourceContext.Include(includes...)
		if searchRequest.Resolve {
			sourceContext = sourceContext.Include("channel", "channel_claim_id", "title", "thumbnail_url", "release_time", "fee", "nsfw", "duration")
		}
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

	if searchRequest.Debug {
		searchResults, err := service.
			Explain(true).
			ErrorTrace(true).
			Do(context.Background())
		if err != nil {
			return api.Response{Error: errors.Err(err)}
		}
		return api.Response{Data: searchResults}
	}
	if searchRequest.SortBy != nil {
		sortBy := strings.TrimPrefix(*searchRequest.SortBy, "^")
		service.Sort(sortBy, strings.Contains(*searchRequest.SortBy, "^"))
	}
	results, err := searchCache.Fetch(r.URL.RequestURI(), 5*time.Minute, func() (interface{}, error) {
		searchResults, err := service.Do(context.Background())
		if err != nil {
			return nil, errors.Err(err)
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
		return results, nil
	})
	if err != nil {
		return api.Response{Error: errors.Err(err)}
	}
	metrics.SearchDuration.WithLabelValues(
		searchRequest.searchType,
		strconv.Itoa(searchRequest.terms)).
		Observe(time.Since(start).Seconds())
	return api.Response{Data: results.Value()}
}
