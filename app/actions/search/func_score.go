package search

import (
	"time"

	"gopkg.in/olivere/elastic.v6"
)

const effectiveFactor = 0.0000000000001

func controllingBoostQuery() *elastic.MatchQuery {
	return elastic.NewMatchQuery("bid_state", "Controlling")
}

func claimWeightFuncScoreQuery() *elastic.FunctionScoreQuery {
	score := elastic.NewFieldValueFactorFunction().
		Field("effective_amount").
		Factor(effectiveFactor).
		Missing(1)

	return elastic.NewFunctionScoreQuery().AddScoreFunc(score)
}

func channelWeightFuncScoreQuery() *elastic.FunctionScoreQuery {
	score := elastic.NewFieldValueFactorFunction().
		Field("certificate_amount").
		Factor(effectiveFactor).
		Missing(1)

	return elastic.NewFunctionScoreQuery().AddScoreFunc(score)
}

func releaseTime7dFuncScoreQuery() *elastic.GaussDecayFunction {
	return elastic.NewGaussDecayFunction().
		FieldName("release_time").
		Origin(time.Now()).
		Scale("7d").
		//Weight(40).
		Decay(0.50).Weight(0.1)
}

func releaseTime30dFuncScoreQuery() *elastic.GaussDecayFunction {
	return elastic.NewGaussDecayFunction().
		FieldName("release_time").
		Origin(time.Now()).
		Scale("30d").
		//Weight(20).
		Decay(0.50).Weight(0.05)

}

func releaseTime90dFuncScoreQuery() *elastic.GaussDecayFunction {
	return elastic.NewGaussDecayFunction().
		FieldName("release_time").
		Origin(time.Now()).
		Scale("90d").
		//Weight(10).
		Decay(0.50).Weight(0.02)

}

func releaseTime1yFuncScoreQuery() *elastic.GaussDecayFunction {
	//For the first year(offset), it gets a base weight of 1.0. This allows the other functions to impact the
	//relevance score in a normalized way. Once it starts get over a year old its relevance score will start to
	//decay slowly
	return elastic.NewGaussDecayFunction().
		FieldName("release_time").
		Origin(time.Now()).
		Scale("1825d"). //5 years
		Offset("365d"). //1 year
		Decay(0.1).
		Weight(1.0)

}
