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

	if len(data) < 2 {
		return features
	}

	// Вычисляем общее количество объектов
	totalObjects := data[len(data)-1].TotalObjects

	// Используем предварительно вычисленные значения
	newObjects := data[len(data)-1].NewObjects
	closedObjects := data[len(data)-1].ClosedObjects

	// Рассчитываем плотность объектов
	if years > 0 {
		features.ObjectDensity = float64(totalObjects) / float64(years)
	}

	// Рассчитываем темпы роста/сокращения
	startCount := data[0].TotalObjects
	endCount := data[len(data)-1].TotalObjects
	netChange := endCount - startCount

	// Рассчитываем показатели
	if startCount > 0 {
		features.NewObjectRate = float64(newObjects) / float64(startCount)
		features.ClosureRate = float64(closedObjects) / float64(startCount)
		features.NetGrowthRate = float64(netChange) / float64(startCount)
	}

	// Рассчитываем наклон тренда
	features.TrendSlope = float64(netChange) / float64(years)

	return features
}
