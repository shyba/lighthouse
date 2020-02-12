package actions

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/lbryio/lighthouse/app/es"
	"github.com/lbryio/lighthouse/app/internal/metrics"
	"github.com/sirupsen/logrus"

	"gopkg.in/olivere/elastic.v6"

	"github.com/lbryio/lbry.go/extras/errors"
	"github.com/lbryio/lbry.go/v2/extras/api"
	v "github.com/lbryio/ozzo-validation"
)

type autoCompleteRequest struct {
	S    string
	Size *int
	From *int
	NSFW *bool
	//Debug params
	Source *bool
	Debug  *bool
}

// AutoComplete returns the name of claims that it matches against for auto completion.
func AutoComplete(r *http.Request) api.Response {
	acRequest := autoCompleteRequest{}
	start := time.Now()
	err := api.FormValues(r, &acRequest, []*v.FieldRules{
		v.Field(&acRequest.S, v.Required, v.Length(1, 0)),
		v.Field(&acRequest.Size, v.Max(10000)),
		v.Field(&acRequest.From, v.Max(9999)),
	})
	if err != nil {
		return api.Response{Error: errors.Err(err), Status: http.StatusBadRequest}
	}
	replacer := strings.NewReplacer("/", "\\/", "[", "\\[", "]", "\\]")
	acRequest.S = replacer.Replace(acRequest.S)

	mmATD := elastic.NewMultiMatchQuery(acRequest.S).
		Type("phrase_prefix").Slop(5).MaxExpansions(50).
		Field("value.Claim.stream.metadata.author^3").
		Field("value.Claim.stream.metadata.title^5").
		Field("value.stream.metadata.description^2")
	nested := elastic.NewNestedQuery("value", mmATD)
	mmName := elastic.NewMultiMatchQuery(acRequest.S).
		Type("phrase_prefix").Slop(5).MaxExpansions(50).
		Field("name^4")
	query := elastic.NewBoolQuery().Should(nested, mmName)
	if acRequest.NSFW != nil {
		query = query.Must(elastic.NewMatchQuery("nsfw", *acRequest.NSFW))
	}

	t, err := query.Source()
	if err != nil {
		return api.Response{Error: errors.Err("%s: for query -s %s", err, t)}
	}
	sourceContext := elastic.NewFetchSourceContext(true)
	if acRequest.Source == nil {
		sourceContext = sourceContext.Include("name", "claimId")
	}
	service := es.Client.
		Search("claims").
		Query(query).
		FetchSourceContext(sourceContext)
	if acRequest.Size != nil {
		service = service.Size(*acRequest.Size)
	}
	if acRequest.From != nil {
		service = service.From(*acRequest.From)
	}

	if acRequest.Debug != nil {
		searchResults, err := service.Explain(true).Do(context.Background())
		if err != nil {
			return api.Response{Error: errors.Err(err)}
		}
		return api.Response{Data: searchResults}
	}
	searchResults, err := service.Do(context.Background())
	if err != nil {
		return api.Response{Error: errors.Err(err)}
	}
	type lighthouseResult struct {
		Name string `json:"name"`
	}
	names := make([]string, 0, len(searchResults.Hits.Hits))
	for _, hit := range searchResults.Hits.Hits {
		if hit.Source != nil {
			data, err := hit.Source.MarshalJSON()
			if err != nil {
				logrus.Error(err)
				continue
			}
			result := lighthouseResult{}
			err = json.Unmarshal(data, &result)
			if err != nil {
				logrus.Error(err)
				continue
			}
			names = append(names, result.Name)
		}
	}
	metrics.AutoCompleteDuration.Observe(time.Since(start).Seconds())
	return api.Response{Data: names}

}
