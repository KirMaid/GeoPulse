package core

import (
	"osm_service/internal/domain/model"
	"sort"
)

type TemporalAnalyzer struct{}

func (a *TemporalAnalyzer) Analyze(data []model.HistoricalData, years int) model.TemporalFeatures {
	sort.Slice(data, func(i, j int) bool {
		return data[i].Period < data[j].Period
	})

	features := model.TemporalFeatures{
		YearsAnalyzed: years,
	}

	if len(data) == 0 {
		return features
	}

	var totalObjects, newObjects int
	for i, d := range data {
		totalObjects += d.TotalObjects
		if i > 0 {
			delta := d.TotalObjects - data[i-1].TotalObjects
			if delta > 0 {
				newObjects += delta
			}
		}
	}

	if years > 0 {
		features.ObjectDensity = float64(totalObjects) / float64(years)
	}

	if totalObjects > 0 {
		features.NewObjectRate = float64(newObjects) / float64(totalObjects)
	}

	if len(data) > 1 {
		first := data[0].TotalObjects
		last := data[len(data)-1].TotalObjects
		features.TrendSlope = float64(last-first) / float64(years)
	}

	return features
}
