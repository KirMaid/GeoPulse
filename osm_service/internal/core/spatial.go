package core

import "osm_service/internal/domain/model"

type SpatialAnalyzer struct{}

func (a *SpatialAnalyzer) Analyze(
	elements []model.OSMElement,
	subways []model.OSMElement,
	roads []model.OSMElement,
) model.SpatialFeatures {
	features := model.SpatialFeatures{
		TotalObjects:   len(elements),
		SubwayStations: len(subways),
	}

	var totalArea float64
	for _, el := range elements {
		totalArea += calculateArea(el)
	}
	features.AvgArea = totalArea / float64(len(elements))

	features.AvgDistToSubway = avgDistance(elements, subways)
	features.AvgDistToPrimary = avgDistance(elements, roads)

	return features
}

func calculateArea(el model.OSMElement) float64 {
	width := el.Bounds.MaxLon - el.Bounds.MinLon
	height := el.Bounds.MaxLat - el.Bounds.MinLat
	return width * height * 111 * 111 // TODO Проверить расчеты
}

func avgDistance(elements, references []model.OSMElement) float64 {
	if len(references) == 0 {
		return 0
	}
	var totalDist float64
	for _, el := range elements {
		minDist := 1e9
		for _, ref := range references {
			dx := (el.Lon - ref.Lon) * 111 * 0.5 // TODO Проверить расчеты
			dy := (el.Lat - ref.Lat) * 111       // TODO Проверить расчеты
			dist := dx*dx + dy*dy
			if dist < minDist {
				minDist = dist
			}
		}
		totalDist += minDist
	}
	return totalDist / float64(len(elements))
}
