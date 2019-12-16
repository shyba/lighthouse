package search

import (
	"time"

	"gopkg.in/olivere/elastic.v6"
)

const effectiveFactor = 0.0000000000001

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

func channelWeightFuncScoreQuery() *elastic.FunctionScoreQuery {
	score := elastic.NewFieldValueFactorFunction().
		Field("certificate_amount").
		Factor(effectiveFactor).
		Missing(1)

	return elastic.NewFunctionScoreQuery().AddScoreFunc(score)
}

func releaseTimeFuncScoreQuery() *elastic.FunctionScoreQuery {
	pastWeekScore := elastic.NewGaussDecayFunction().
		FieldName("release_time").
		Origin(time.Now()).
		Scale("7d").
		Weight(10).
		Decay(0.75)

	pastMonthScore := elastic.NewGaussDecayFunction().
		FieldName("release_time").
		Origin(time.Now()).
		Scale("30d").
		Weight(6).
		Decay(0.65)

	pastQuarterScore := elastic.NewGaussDecayFunction().
		FieldName("release_time").
		Origin(time.Now()).
		Scale("90d").
		Weight(4).
		Decay(0.55)

	pastYearScore := elastic.NewGaussDecayFunction().
		FieldName("release_time").
		Origin(time.Now()).
		Scale("365d").
		Weight(2).
		Decay(0.45)

	return elastic.NewFunctionScoreQuery().
		AddScoreFunc(pastWeekScore).
		AddScoreFunc(pastMonthScore).
		AddScoreFunc(pastQuarterScore).
		AddScoreFunc(pastYearScore)
}
