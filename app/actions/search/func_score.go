package search

import (
	"time"

	"gopkg.in/olivere/elastic.v6"
)

func controllingBoostQuery() *elastic.ConstantScoreQuery {
	return elastic.NewConstantScoreQuery(elastic.NewMatchQuery("bid_state", "Controlling")).Boost(50)
}

func thumbnailBoostQuery() *elastic.ConstantScoreQuery {
	emptyThumbnail := elastic.NewMatchQuery("thumbnail_url", "")
	thumbnailExists := elastic.NewExistsQuery("thumbnail_url")
	notEmptyThumbnail := elastic.NewBoolQuery().
		MustNot(emptyThumbnail).
		Must(thumbnailExists).
		QueryName("not-empty-thumbnail")
	return elastic.NewConstantScoreQuery(notEmptyThumbnail).Boost(50)

}

func claimWeightFuncScoreQuery() *elastic.FunctionScoreQuery {
	score := elastic.NewFieldValueFactorFunction().
		Field("effective_amount").
		//Factor(effectiveFactor).
		Modifier("log1p").
		Missing(1)

	return elastic.NewFunctionScoreQuery().AddScoreFunc(score)
}

func channelWeightFuncScoreQuery() *elastic.FunctionScoreQuery {
	score := elastic.NewFieldValueFactorFunction().
		Field("certificate_amount").
		//Factor(effectiveFactor).
		Modifier("log1p").
		Missing(1)

	return elastic.NewFunctionScoreQuery().AddScoreFunc(score)
}

func releaseTime7dFuncScoreQuery() *elastic.GaussDecayFunction {
	//Each day it looses 10% of its boost.
	return elastic.NewGaussDecayFunction().
		FieldName("release_time").
		Origin(time.Now()).
		Scale("1d").
		Decay(0.20).
		Weight(0.1)
}

func releaseTime30dFuncScoreQuery() *elastic.GaussDecayFunction {
	//After 30 days it loses 10% of boost each day
	return elastic.NewGaussDecayFunction().
		FieldName("release_time").
		Origin(time.Now()).
		Offset("30d").
		Scale("1d").
		Decay(0.20).
		Weight(0.1)

}

func releaseTime90dFuncScoreQuery() *elastic.GaussDecayFunction {
	//After 90 days it loses 50% of boost every month
	return elastic.NewGaussDecayFunction().
		FieldName("release_time").
		Origin(time.Now()).
		Offset("90d").
		Scale("30d").
		Decay(0.50).
		Weight(0.1)

}

func releaseTime1yFuncScoreQuery() *elastic.GaussDecayFunction {
	//The first year gets full credit, after every month loses 10%
	return elastic.NewGaussDecayFunction().
		FieldName("release_time").
		Origin(time.Now()).
		Offset("365d").
		Scale("1m"). //5 years
		Decay(0.1).
		Weight(1.0)
}

func viewCountFuncScoreQuery() *elastic.FunctionScoreQuery {
	score := elastic.NewFieldValueFactorFunction().Field("view_cnt").Missing(0.0).
		Modifier("log1p")

	return elastic.NewFunctionScoreQuery().AddScoreFunc(score)
}

func subscriptionCountFuncScoreQuery() *elastic.FunctionScoreQuery {
	score := elastic.NewFieldValueFactorFunction().Field("sub_cnt").Missing(0.0).
		Modifier("log1p")

	return elastic.NewFunctionScoreQuery().AddScoreFunc(score)
}
