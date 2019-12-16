package actions

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lbryio/lighthouse/app/es"
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
	//Debug params
	Source *bool
	Debug  *bool
}

func AutoComplete(r *http.Request) api.Response {
	acRequest := autoCompleteRequest{}

	err := api.FormValues(r, &acRequest, []*v.FieldRules{
		v.Field(&acRequest.S, v.Required, v.Length(1, 0)),
		v.Field(&acRequest.Size, v.Max(10000)),
		v.Field(&acRequest.From, v.Max(9999)),
	})
	if err != nil {
		return api.Response{Error: errors.Err(err), Status: http.StatusBadRequest}
	}
	acRequest.S = strings.Replace(acRequest.S, "/", "\\/", -1)

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
