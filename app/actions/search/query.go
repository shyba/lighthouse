package search

import (
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/lbryio/lbry.go/extras/errors"
	"github.com/lbryio/lbry.go/v2/extras/util"
	"github.com/olivere/elastic"
)

const (
	effectiveFactor = 0.0000000000001
	authorPath      = "value.Claim.stream.metadata.author"
	titlePath       = "value.Claim.stream.metadata.title"
	descPath        = "value.Claim.stream.metadata.description"

	nsfwPath = "value.Claim.stream.metadata.nsfw"
)

func (r searchRequest) NewQuery() *elastic.BoolQuery {
	base := elastic.NewBoolQuery()
	if exact := r.exactMatchQueries(); exact != nil {
		base.Must(exact)
	}
	base.Should(claimWeightFuncScoreQuery())
	base.Should(channelWeightFuncScoreQuery())
	base.Should(controllingBoostQuery())
	base.Should(r.matchPhraseClaimName())
	base.Should(r.matchClaimName())
	base.Should(r.containsTermName())
	base.Should(r.atdSearch())
	base.Must(r.dynamicQueries()...)
	base.Should(splitNameQueries()...)
	base.Filter(r.getFilters()...)

	return base
}

func (r searchRequest) escaped() string {
	return r.S
}

func (r searchRequest) washed() string {
	return r.S
}

func (r searchRequest) dynamicQueries() []elastic.Query {
	var queries []elastic.Query
	if channelID := r.channelIDFilter(); channelID != nil {
		queries = append(queries, channelID)
	}
	if channel := r.channelFilter(); channel != nil {
		queries = append(queries, channel)
	}
	return queries
}

func splitNameQueries() []elastic.Query {
	//Add split ATD
	return nil
}

func splitATDQueries() []elastic.Query {
	//Add split ATD
	return nil
}

func (r searchRequest) atdSearch() *elastic.BoolQuery {
	b := elastic.NewBoolQuery()

	//Add queries for splits of the search query A, AB, ABC, ABCD
	b.Should(splitATDQueries()...)

	// Contains search in Author, Title, Description
	b.Should(elastic.NewQueryStringQuery("*" + r.escaped() + "*").
		QueryName("contains_atd").
		//Field(authorPath).
		Field("title").
		Field("description").
		Boost(1))

	// Match search terms - Title
	b.Should(elastic.NewMatchQuery("title", r.washed()).
		QueryName("match_term_title").
		Boost(3))
	// Match search terms - Description
	b.Should(elastic.NewMatchQuery("description", r.washed()).
		QueryName("match_term_desc").
		Boost(3))

	// Match Phrase search - Title
	b.Should(elastic.NewMatchPhrasePrefixQuery("title", r.escaped()).
		QueryName("matchphrase_term_title").
		Boost(2))
	// Match Phrase search - Description
	b.Should(elastic.NewMatchPhrasePrefixQuery("description", r.escaped()).
		QueryName("matchphrase_term_desc").
		Boost(2))

	return b
}

func (r searchRequest) matchPhraseClaimName() *elastic.MatchPhraseQuery {
	boost := 2.0
	if r.S[0] == '@' {
		boost = boost * 10
	}
	return elastic.NewMatchPhraseQuery("name", r.S).
		QueryName("match_phrase_claim_name*10").
		Boost(boost)
}

func (r searchRequest) matchClaimName() *elastic.MatchQuery {
	boost := 10.0
	if r.S[0] == '@' {
		boost = boost * 10
	}
	return elastic.NewMatchQuery("name", r.S).
		QueryName("match_claim_name*10").
		Boost(boost)
}

func (r searchRequest) containsTermName() *elastic.QueryStringQuery {
	return elastic.NewQueryStringQuery("*" + r.S + "*").
		QueryName("contains_term_name*3").
		Field("name").
		Boost(5)
}

func channelSearchBoolQuery(search string) *elastic.BoolQuery {
	q := elastic.NewQueryStringQuery(search).
		QueryName("channel_search_bool").
		Field("channel")
	return elastic.NewBoolQuery().Must(q)
}

func controllingBoostQuery() *elastic.MatchQuery {
	return elastic.NewMatchQuery("bid_state", "Controlling").
		QueryName("controlling_boost*20")
}

func claimWeightFuncScoreQuery() *elastic.FunctionScoreQuery {
	score := elastic.NewFieldValueFactorFunction().
		Field("effective_amount").
		Factor(effectiveFactor).
		Missing(1)

	return elastic.NewFunctionScoreQuery().AddScoreFunc(score)
}

func (r searchRequest) exactMatchQueries() elastic.Query {
	exact := elastic.NewBoolQuery().QueryName("exact-match")

	regex, err := regexp.Compile(`"([^"]*)"$`)
	if err != nil {
		logrus.Error(errors.Err(err))
		return nil
	}

	exactMatches := regex.FindAllStringSubmatch(r.S, -1)
	if len(exactMatches) == 0 {
		return nil
	}
	for _, exactMatch := range exactMatches {
		exact.Should(elastic.NewMatchPhraseQuery("channel", exactMatch[len(exactMatch)-1]))
		exact.Should(elastic.NewMatchPhraseQuery("name", exactMatch[len(exactMatch)-1]))
		exact.Should(elastic.NewMatchPhraseQuery("title", exactMatch[len(exactMatch)-1]).QueryName("exact-title"))
		exact.Should(elastic.NewMatchPhraseQuery("description", exactMatch[len(exactMatch)-1]).QueryName("exact-description"))

	}
	//nested := elastic.NewNestedQuery("value", b)
	return exact
}

func channelWeightFuncScoreQuery() *elastic.FunctionScoreQuery {
	score := elastic.NewFieldValueFactorFunction().
		Field("certificate_amount").
		Factor(effectiveFactor).
		Missing(1)

	return elastic.NewFunctionScoreQuery().AddScoreFunc(score)
}

func (r searchRequest) getFilters() []elastic.Query {
	var filters []elastic.Query
	bidstateFilter := r.bidStateFilter()

	if nsfwFilter := r.nsfwFilter(); nsfwFilter != nil {
		filters = append(filters, nsfwFilter)
	}

	if contentTypeFilter := r.contentTypeFilter(); contentTypeFilter != nil {
		filters = append(filters, contentTypeFilter)
	}

	if mediaTypeFilters := r.mediaTypeFilter(); len(mediaTypeFilters) > 0 {
		b := elastic.NewBoolQuery().Should(mediaTypeFilters...)
		filters = append(filters, b)
	} else if r.MediaType != nil {
		filters = append(filters, elastic.NewMatchNoneQuery())
	}

	if claimTypeFilter := r.claimTypeFilter(); claimTypeFilter != nil {
		filters = append(filters, claimTypeFilter)
	}

	if len(filters) > 0 {
		return append(filters, bidstateFilter)

	} else {
		return []elastic.Query{bidstateFilter}
	}
}

var cadTypes = []interface{}{"SKP", "simplify3d_stl"}
var contains = func(slice []string, value string) bool {
	for _, s := range slice {
		if s == value {
			return true
		}
	}
	return false
}
var possibleMediaTypes = []string{"audio", "video", "text", "application", "image"}

func (r searchRequest) mediaTypeFilter() []elastic.Query {
	if r.MediaType != nil {
		mediaTypes := strings.Split(util.StrFromPtr(r.MediaType), ",")
		var queries []elastic.Query
		for _, t := range mediaTypes {
			if contains(possibleMediaTypes, t) && t != "" {
				queries = append(queries, elastic.NewPrefixQuery("content_type.keyword", t+"/"))
			} else if t == "cad" {
				queries = append(queries, elastic.NewTermsQuery("content_type.keyword", cadTypes...))
			}
		}
		return queries
	}
	return nil
}

var claimTypeMap = map[string]string{"channel": "channel", "file": "stream"}

func (r searchRequest) claimTypeFilter() *elastic.MatchQuery {
	if r.ClaimType != nil {
		if t, ok := claimTypeMap[util.StrFromPtr(r.ClaimType)]; ok {
			return elastic.NewMatchQuery("claim_type", t)
		}
	}
	return nil
}

func (r searchRequest) contentTypeFilter() *elastic.TermsQuery {
	if r.ContentType != nil {
		contentTypeStr := strings.Split(util.StrFromPtr(r.ContentType), ",")
		contentTypes := make([]interface{}, len(contentTypeStr))
		for i, t := range contentTypeStr {
			contentTypes[i] = t
		}
		return elastic.NewTermsQuery("content_type", contentTypes...)
	}
	return nil
}

func (r searchRequest) nsfwFilter() *elastic.MatchQuery {
	if r.NSFW != nil {
		return elastic.NewMatchQuery("nsfw", r.NSFW)
	}
	return nil
}

func (r searchRequest) bidStateFilter() *elastic.BoolQuery {
	return elastic.NewBoolQuery().MustNot(elastic.NewMatchQuery("bid_state", r.S))
}

func (r searchRequest) channelIDFilter() *elastic.BoolQuery {
	if r.ChannelID != nil {
		b := elastic.NewBoolQuery()
		channelID := elastic.NewQueryStringQuery(r.escaped()).
			Field("channel_claim_id")
		return b.Must(channelID)
	}
	return nil
}

func (r searchRequest) channelFilter() *elastic.BoolQuery {
	if r.Channel != nil {
		b := elastic.NewBoolQuery()
		channel := elastic.NewQueryStringQuery(r.escaped()).
			Field("channel")
		return b.Must(channel)
	}
	return nil
}
