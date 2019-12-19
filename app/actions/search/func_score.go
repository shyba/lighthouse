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

func releaseTime7dFuncScoreQuery() *elastic.FunctionScoreQuery {
	pastWeekScore := elastic.NewGaussDecayFunction().
		FieldName("release_time").
		Origin(time.Now()).
		Scale("7d").
		Weight(10).
		Decay(0.50)

	return elastic.NewFunctionScoreQuery().
		AddScoreFunc(pastWeekScore).MinScore(1.0)
}

func releaseTime30dFuncScoreQuery() *elastic.FunctionScoreQuery {
	pastMonthScore := elastic.NewGaussDecayFunction().
		FieldName("release_time").
		Origin(time.Now()).
		Scale("30d").
		Weight(8).
		Decay(0.45)

	return elastic.NewFunctionScoreQuery().
		AddScoreFunc(pastMonthScore).MinScore(1.0)
}

func releaseTime90dFuncScoreQuery() *elastic.FunctionScoreQuery {
	pastQuarterScore := elastic.NewGaussDecayFunction().
		FieldName("release_time").
		Origin(time.Now()).
		Scale("90d").
		Weight(6).
		Decay(0.40)

	return elastic.NewFunctionScoreQuery().
		AddScoreFunc(pastQuarterScore).MinScore(1.0)
}

func releaseTime1yFuncScoreQuery() *elastic.FunctionScoreQuery {
	pastYearScore := elastic.NewGaussDecayFunction().
		FieldName("release_time").
		Origin(time.Now()).
		Scale("365d").
		Weight(4).
		Decay(0.35)

	return elastic.NewFunctionScoreQuery().
		AddScoreFunc(pastYearScore).MinScore(1.0)
}
