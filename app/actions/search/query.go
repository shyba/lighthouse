package search

import (
	"regexp"
	"strings"

	"github.com/lbryio/lighthouse/app/es/index"

	"github.com/lbryio/lbry.go/extras/errors"
	"github.com/lbryio/lbry.go/v2/extras/util"

	"github.com/sirupsen/logrus"
	"gopkg.in/olivere/elastic.v6"
)

var streamOnlyMatch = elastic.NewMatchQuery("claim_type", "stream")

// ChannelOnlyMatch is a default query that matches only channels.
var ChannelOnlyMatch = elastic.NewMatchQuery("claim_type", "channel")

func (r searchRequest) newQuery() *elastic.FunctionScoreQuery {
	base := elastic.NewBoolQuery()

	//Things that should bee scaled once a match is found
	base.Should(claimWeightFuncScoreQuery())
	base.Should(channelWeightFuncScoreQuery())
	base.Should(controllingBoostQuery())
	base.Should(thumbnailBoostQuery())
	base.Should(viewCountFuncScoreQuery())
	base.Should(subscriptionCountFuncScoreQuery())
	base.Should(claimCountFuncScoreQuery())

	//The minimum things that should match for it to be considered a valid result.
	//Anything in here will allow it to be scaled and returned
	min := elastic.NewBoolQuery()
	min.Should(r.moreLikeThis())
	min.Should(r.matchPhraseName())
	min.Should(r.matchName())
	min.Should(r.matchChannelName())
	//min.Should(r.nameContains())
	//min.Should(r.titleContains())
	//min.Should(r.descriptionContains())
	min.Should(r.matchTitle())
	min.Should(r.matchPhraseTitle())
	min.Should(r.matchDescription())
	min.Should(r.matchPhraseDescription())
	min.Should(r.matchCompressedName())
	min.Should(r.matchChannel())
	min.Should(r.matchCompressedChannel())
	base.Must(min)

	if r.RelatedTo != nil {
		base = elastic.NewBoolQuery()
		base.Should(r.moreLikeThis())
		base.Filter(r.getFilters()...)
		return elastic.NewFunctionScoreQuery().
			ScoreMode("sum").
			Query(base)
	}
	//Any parameters that should filter but not impact scores
	base.Filter(r.getFilters()...)

	return elastic.NewFunctionScoreQuery().
		ScoreMode("sum").
		Query(base).
		//Boosting overall relevance over time
		AddScoreFunc(releaseTime7dFuncScoreQuery()).
		AddScoreFunc(releaseTime30dFuncScoreQuery()).
		AddScoreFunc(releaseTime90dFuncScoreQuery()).
		AddScoreFunc(releaseTime1yFuncScoreQuery())
}

func (r searchRequest) escaped() string {
	// https://www.elastic.co/guide/en/elasticsearch/reference/current/query-dsl-query-string-query.html#_reserved_characters
	// The reserved characters are: + - = && || > < ! ( ) { } [ ] ^ " ~ * ? : \ /
	replacer := strings.NewReplacer(
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"&&", "\\&\\&",
		"||", "\\|\\|",
		">", "\\>",
		"<", "\\<",
		"!", "\\!",
		"(", "\\(",
		")", "\\)",
		"{", "\\{",
		"}", "\\}",
		"[", "\\[",
		"]", "\\]",
		"^", "\\^",
		"\"", "\\\"",
		"~", "\\~",
		"*", "\\*",
		"?", "\\?",
		":", "\\:",
		"/", "\\/",
	)
	return replacer.Replace(r.S)
}

func (r searchRequest) washed() string {
	return r.S
}

func (r searchRequest) moreLikeThis() *elastic.MoreLikeThisQuery {
	mlt := elastic.NewMoreLikeThisQuery().QueryName("more-like-this").
		Field("name").
		Field("title").
		Field("channel").
		//MinWordLength(5).
		IgnoreLikeText("https")
	if r.RelatedTo != nil {
		item := elastic.NewMoreLikeThisQueryItem().
			Index(index.Claims).
			Id(util.StrFromPtr(r.RelatedTo))
		return elastic.NewMoreLikeThisQuery().QueryName("more-like-this").LikeItems(item).
			Boost(2)
	}
	return mlt.LikeText(r.S)
}

func (r searchRequest) titleContains() *elastic.QueryStringQuery {
	return elastic.NewQueryStringQuery("*" + r.escaped() + "*").
		QueryName("title-contains").
		Field("title").
		Boost(2)
}

func (r searchRequest) matchTitle() *elastic.MatchQuery {
	return elastic.NewMatchQuery("title", r.S).Fuzziness("AUTO").
		QueryName("title-match").
		Boost(1)
}

func (r searchRequest) matchPhraseTitle() *elastic.MatchPhraseQuery {
	return elastic.NewMatchPhraseQuery("title", r.escaped()).
		QueryName("title-match-phrase").
		Boost(10)
}

func (r searchRequest) descriptionContains() *elastic.QueryStringQuery {
	return elastic.NewQueryStringQuery("*" + r.escaped() + "*").
		QueryName("description-contains").
		Field("description").
		Boost(1)
}

func (r searchRequest) matchDescription() *elastic.MatchQuery {
	return elastic.NewMatchQuery("description", r.washed()). //Fuzziness("AUTO").
									QueryName("description-match").
									Boost(1)
}

func (r searchRequest) matchPhraseDescription() *elastic.MatchPhraseQuery {
	return elastic.NewMatchPhraseQuery("description", r.escaped()).
		QueryName("description-match-phrase").
		Boost(2)
}

func (r searchRequest) matchPhraseName() *elastic.MatchPhraseQuery {
	boost := 2.0
	if r.S[0] == '@' {
		boost = boost * 10
	}
	return elastic.NewMatchPhraseQuery("name", r.S).
		QueryName("name-match-phrase").
		Boost(boost)
}

func (r searchRequest) matchName() *elastic.BoolQuery {
	boost := 1.0
	if r.S[0] == '@' {
		boost = boost * 10
	}
	return elastic.NewBoolQuery().
		Should(elastic.NewMatchQuery("name", r.S).Fuzziness("AUTO")).
		Boost(boost).
		QueryName("name-match")
}

func (r searchRequest) matchChannelName() *elastic.BoolQuery {
	//This is what returns a channel as the first result when searching
	return elastic.NewBoolQuery().
		Must(elastic.NewMatchPhraseQuery("name", r.S)).
		Must(ChannelOnlyMatch).
		Boost(10).
		QueryName("channel-phrase-match")
}

func (r searchRequest) matchCompressedName() *elastic.BoolQuery {
	//This is what returns channels with multiple words as the first result when searching
	compressed := strings.Replace(r.S, " ", "", -1)
	matchName := elastic.NewMatchQuery("name", compressed).Fuzziness("AUTO").
		Boost(10)
	return elastic.NewBoolQuery().
		QueryName("name-match-@compressed").
		Must(ChannelOnlyMatch).
		Must(matchName)
}

func (r searchRequest) matchChannel() *elastic.BoolQuery {
	channelMatch := elastic.NewMatchQuery("channel", r.S)
	return elastic.NewBoolQuery().
		QueryName("channel-match-@boost").
		Must(streamOnlyMatch).
		Must(channelMatch).
		Boost(5)
}

func (r searchRequest) matchCompressedChannel() *elastic.BoolQuery {
	compressed := strings.Replace(r.S, " ", "", -1)
	matchChannel := elastic.NewMatchPhraseQuery("channel", compressed).
		Boost(5)
	return elastic.NewBoolQuery().
		QueryName("channel-match-@compressed").
		Must(streamOnlyMatch).
		Must(matchChannel)
}

func (r searchRequest) nameContains() *elastic.QueryStringQuery {
	return elastic.NewQueryStringQuery("*" + r.escaped() + "*").
		QueryName("name-contains").
		AnalyzeWildcard(true).
		AllowLeadingWildcard(true).
		Field("name").
		Boost(1)
}

func (r searchRequest) exactMatchQueries() elastic.Query {
	exact := elastic.NewBoolQuery()

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
		v := exactMatch[len(exactMatch)-1]
		exact.Should(elastic.NewMatchPhraseQuery("channel", v).QueryName("channel-exact"))
		exact.Should(elastic.NewMatchPhraseQuery("name", v).QueryName("name-exact"))
		exact.Should(elastic.NewMatchPhraseQuery("title", v).QueryName("title-exact"))
		exact.Should(elastic.NewMatchPhraseQuery("description", v).QueryName("description-exact"))

	}
	//nested := elastic.NewNestedQuery("value", b)
	return exact
}

func (r searchRequest) getFilters() []elastic.Query {
	var filters []elastic.Query
	bidstateFilter := r.bidStateFilter()

	if exact := r.exactMatchQueries(); exact != nil {
		filters = append(filters, exact)
	}

	if nsfwFilter := r.nsfwFilter(); nsfwFilter != nil {
		filters = append(filters, nsfwFilter)
	}

	if freeFilter := r.freeContentFilter(); freeFilter != nil {
		filters = append(filters, freeFilter)
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

	if channelID := r.channelIDFilter(); channelID != nil {
		filters = append(filters, channelID)
	}

	if channel := r.channelFilter(); channel != nil {
		filters = append(filters, channel)
	}

	if claim := r.claimIDFilter(); claim != nil {
		filters = append(filters, claim)
	}

	if related := r.relatedContentFilter(); related != nil {
		filters = append(filters, related)
	}

	if len(filters) > 0 {
		return append(filters, bidstateFilter) //, r.noClaimChannelFilter())
	}
	return []elastic.Query{bidstateFilter} //, r.noClaimChannelFilter()}
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

func (r searchRequest) relatedContentFilter() *elastic.MatchQuery {
	if r.RelatedTo != nil {
		return streamOnlyMatch
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

func (r searchRequest) nsfwFilter() elastic.Query {
	if r.NSFW != nil {
		termsQuery := elastic.NewTermsQuery("tags", "nsfw", "porn", "mature", "xxx")
		nsfwMatch := elastic.NewMatchQuery("nsfw", true)
		if !*r.NSFW {
			return elastic.NewBoolQuery().MustNot(termsQuery, nsfwMatch)
		}
		return elastic.NewBoolQuery().Should(termsQuery, nsfwMatch).MinimumShouldMatch("1")
	}
	return nil
}

func (r searchRequest) freeContentFilter() elastic.Query {
	if r.FreeOnly != nil && *r.FreeOnly {
		freeMatch := elastic.NewMatchQuery("fee", 0.0)
		return elastic.NewBoolQuery().Must(freeMatch)
	}
	return nil
}

func (r searchRequest) bidStateFilter() *elastic.BoolQuery {
	return elastic.NewBoolQuery().MustNot(elastic.NewMatchQuery("bid_state", "Expired"))
}

func (r searchRequest) noClaimChannelFilter() *elastic.BoolQuery {
	filtered := elastic.NewBoolQuery().Must(ChannelOnlyMatch).Must(elastic.NewMatchQuery("claim_cnt", "1"))
	return elastic.NewBoolQuery().MustNot(filtered)
}

func (r searchRequest) channelIDFilter() *elastic.MatchQuery {
	if r.ChannelID != nil {
		return elastic.NewMatchQuery("channel_claim_id", r.ChannelID)
	}
	return nil
}

func (r searchRequest) channelFilter() *elastic.BoolQuery {
	if r.Channel != nil {
		b := elastic.NewBoolQuery()
		channel := elastic.NewQueryStringQuery(util.StrFromPtr(r.Channel)).
			Field("channel")
		return b.Must(channel)
	}
	return nil
}

func (r searchRequest) claimIDFilter() *elastic.MatchQuery {
	if r.ClaimID != nil {
		return elastic.NewMatchQuery("claimId", r.ClaimID)
	}
	return nil
}
