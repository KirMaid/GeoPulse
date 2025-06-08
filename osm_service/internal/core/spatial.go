package core

import (
	"math"
	"osm_service/internal/domain/model"
)

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

	if len(elements) == 0 {
		return features
	}

	// Расчет средней площади
	var totalArea float64
	for _, el := range elements {
		totalArea += calculateArea(el.Bounds)
	}
	features.AvgArea = totalArea / float64(len(elements))

	// Расчет средних расстояний
	features.AvgDistToSubway = avgDistance(elements, subways)
	features.AvgDistToPrimary = avgDistance(elements, roads)

	return features
}

func calculateArea(bounds model.Bounds) float64 {
	// Более точный расчет площади с учетом кривизны Земли
	latMid := (bounds.MinLat + bounds.MaxLat) / 2 * math.Pi / 180
	dLat := bounds.MaxLat - bounds.MinLat
	dLon := bounds.MaxLon - bounds.MinLon

	// Коэффициенты перевода градусов в метры
	kx := 111132.92 - 559.82*math.Cos(2*latMid)
	ky := 111412.84 * math.Cos(latMid)

	return math.Abs(dLat*kx*dLon*ky) / 1000000 // Площадь в км²
}

func avgDistance(elements, references []model.OSMElement) float64 {
	if len(references) == 0 || len(elements) == 0 {
		return 0
	}

	var totalDist float64
	for _, el := range elements {
		minDist := haversine(el.Lat, el.Lon, references[0].Lat, references[0].Lon)
		for _, ref := range references[1:] {
			dist := haversine(el.Lat, el.Lon, ref.Lat, ref.Lon)
			if dist < minDist {
				minDist = dist
			}
		}
		totalDist += minDist
	}
	return totalDist / float64(len(elements))
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Радиус Земли в км
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
