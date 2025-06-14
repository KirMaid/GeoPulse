package core

import (
	"context"
	"fmt"
	"log"
	"math"
	"osm_service/internal/domain/model"
	"osm_service/internal/domain/repository"
	"strconv"
	"strings"
	"time"
)

type PredictionService struct {
	overpassRepo     repository.OverpassRepository
	postgisRepo      repository.PostGISRepository
	mlClient         model.MLClient
	trainingRecorder repository.TrainingDataRecorder
	saveData         bool
}

func NewPredictionService(
	overpassRepo repository.OverpassRepository,
	postgisRepo repository.PostGISRepository,
	mlClient model.MLClient,
	recorder repository.TrainingDataRecorder,
	saveData bool,
) *PredictionService {
	return &PredictionService{
		overpassRepo:     overpassRepo,
		postgisRepo:      postgisRepo,
		mlClient:         mlClient,
		trainingRecorder: recorder,
		saveData:         saveData,
	}
}

func (s *PredictionService) getHistoricalData(
	ctx context.Context,
	bbox string,
	years int,
	shopType string,
) ([]model.HistoricalData, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(-years, 0, 0)

	// Форматируем даты для Overpass
	startStr := startDate.Format("2006-01-02T15:04:05Z")
	endStr := endDate.Format("2006-01-02T15:04:05Z")

	// Получаем данные на начало периода
	startElements, err := s.overpassRepo.GetCommercialDataByDate(ctx, bbox, shopType, startStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get start date data: %w", err)
	}

	// Получаем данные на конец периода
	endElements, err := s.overpassRepo.GetCommercialDataByDate(ctx, bbox, shopType, endStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get end date data: %w", err)
	}

	// Создаем карты для быстрого поиска
	startSet := make(map[int64]struct{})
	for _, el := range startElements {
		startSet[el.ID] = struct{}{}
	}

	endSet := make(map[int64]struct{})
	for _, el := range endElements {
		endSet[el.ID] = struct{}{}
	}

	// Вычисляем новые и закрытые объекты
	newObjects := 0
	for id := range endSet {
		if _, exists := startSet[id]; !exists {
			newObjects++
		}
	}

	closedObjects := 0
	for id := range startSet {
		if _, exists := endSet[id]; !exists {
			closedObjects++
		}
	}

	// Формируем исторические данные
	return []model.HistoricalData{
		{
			Period:        startDate.Format("2006-01-02"),
			BBox:          bbox,
			TotalObjects:  len(startElements),
			NewObjects:    0,
			ClosedObjects: 0,
		},
		{
			Period:        endDate.Format("2006-01-02"),
			BBox:          bbox,
			TotalObjects:  len(endElements),
			NewObjects:    newObjects,
			ClosedObjects: closedObjects,
		},
	}, nil
}

func (s *PredictionService) generatePeriods(start, end time.Time) []string {
	var periods []string
	for current := start; current.Before(end); current = current.AddDate(0, 3, 0) {
		periods = append(periods, fmt.Sprintf("%d-Q%d", current.Year(), (current.Month()-1)/3+1))
	}
	return periods
}

func (s *PredictionService) calculateFeatures(
	ctx context.Context,
	current []model.OSMElement,
	historical []model.HistoricalData,
	years int,
) model.FeatureSet {
	spatialAnalyzer := SpatialAnalyzer{}
	temporalAnalyzer := TemporalAnalyzer{}

	// Если нет текущих элементов, возвращаем пустой набор признаков
	if len(current) == 0 {
		return model.FeatureSet{
			Spatial:  model.SpatialFeatures{},
			Temporal: temporalAnalyzer.Analyze(historical, years),
			Elements: current,
		}
	}

	// Получаем границы кластера из первого элемента
	var bounds model.Bounds
	if len(current) > 0 {
		bounds = current[0].Bounds
	} else {
		// Если нет элементов, используем границы из исторических данных
		if len(historical) > 0 {
			// Парсим bbox из исторических данных
			bbox := historical[0].Period // Предполагаем, что Period содержит bbox
			parts := strings.Split(bbox, ",")
			if len(parts) == 4 {
				minLat, _ := strconv.ParseFloat(parts[0], 64)
				minLon, _ := strconv.ParseFloat(parts[1], 64)
				maxLat, _ := strconv.ParseFloat(parts[2], 64)
				maxLon, _ := strconv.ParseFloat(parts[3], 64)
				bounds = model.Bounds{
					MinLat: minLat,
					MinLon: minLon,
					MaxLat: maxLat,
					MaxLon: maxLon,
				}
			}
		}
	}

	// Формируем bbox строку
	bbox := fmt.Sprintf("%f,%f,%f,%f",
		bounds.MinLat,
		bounds.MinLon,
		bounds.MaxLat,
		bounds.MaxLon)

	log.Printf("Using bbox for queries: %s", bbox)

	// Получаем данные о метро для текущего кластера
	subways, err := s.overpassRepo.GetSubwayData(ctx, bbox, time.Now().Format("2006-01-02T15:04:05Z"))
	if err != nil {
		log.Printf("Warning: failed to get subway data: %v", err)
	}

	// Получаем данные о дорогах для текущего кластера
	roads, err := s.overpassRepo.GetRoadData(ctx, bbox, time.Now().Format("2006-01-02T15:04:05Z"))
	if err != nil {
		log.Printf("Warning: failed to get road data: %v", err)
	}

	spatial := spatialAnalyzer.Analyze(current, subways, roads)
	temporal := temporalAnalyzer.Analyze(historical, years)

	return model.FeatureSet{
		Spatial:  spatial,
		Temporal: temporal,
		Elements: current,
	}
}

// GetHistoricalData retrieves historical data for a specific cluster and time period
func (s *PredictionService) GetHistoricalData(ctx context.Context, bounds model.Bounds, years int, shopType string) ([]model.HistoricalData, error) {
	bbox := fmt.Sprintf("%f,%f,%f,%f", bounds.MinLat, bounds.MinLon, bounds.MaxLat, bounds.MaxLon)
	return s.getHistoricalData(ctx, bbox, years, shopType)
}

// GetCurrentData возвращает текущие данные о коммерческих объектах
func (s *PredictionService) GetCurrentData(ctx context.Context, bounds model.Bounds, shopType string) ([]model.OSMElement, error) {
	// Формируем bbox строку
	bbox := fmt.Sprintf("%f,%f,%f,%f",
		bounds.MinLat,
		bounds.MinLon,
		bounds.MaxLat,
		bounds.MaxLon)

	// Получаем данные из OSM
	elements, err := s.overpassRepo.GetCommercialData(ctx, bbox, shopType)
	if err != nil {
		return nil, fmt.Errorf("failed to get commercial data: %w", err)
	}

	// Получаем данные о дорогах и метро
	roads, err := s.overpassRepo.GetRoadData(ctx, bbox, time.Now().Format("2006-01-02T15:04:05Z"))
	if err != nil {
		return nil, fmt.Errorf("failed to get roads: %w", err)
	}

	subways, err := s.overpassRepo.GetSubwayData(ctx, bbox, time.Now().Format("2006-01-02T15:04:05Z"))
	if err != nil {
		return nil, fmt.Errorf("failed to get subways: %w", err)
	}

	// Рассчитываем дополнительные характеристики для каждого объекта
	for i := range elements {
		// Рассчитываем площадь
		if building, ok := elements[i].Tags["building"]; ok && building != "" {
			// TODO: Реализовать расчет площади здания
			elements[i].Area = 100.0 // Временное значение
		}

		// Находим ближайшую главную дорогу
		minDistToPrimary := math.MaxFloat64
		for _, road := range roads {
			if highway, ok := road.Tags["highway"]; ok && highway == "primary" {
				dist := calculateDistance(
					elements[i].Lat, elements[i].Lon,
					road.Lat, road.Lon,
				)
				if dist < minDistToPrimary {
					minDistToPrimary = dist
				}
			}
		}
		elements[i].DistToPrimary = minDistToPrimary

		// Находим ближайшую станцию метро
		minDistToSubway := math.MaxFloat64
		for _, subway := range subways {
			if transport, ok := subway.Tags["public_transport"]; ok && transport == "station" {
				dist := calculateDistance(
					elements[i].Lat, elements[i].Lon,
					subway.Lat, subway.Lon,
				)
				if dist < minDistToSubway {
					minDistToSubway = dist
				}
			}
		}
		elements[i].DistToSubway = minDistToSubway
	}

	return elements, nil
}

// calculateDistance рассчитывает расстояние между двумя точками в метрах
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000 // радиус Земли в метрах
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	dlat := lat2Rad - lat1Rad
	dlon := lon2Rad - lon1Rad

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// CalculateFeatures calculates spatial and temporal features for the given data
func (s *PredictionService) CalculateFeatures(ctx context.Context, current []model.OSMElement, historical []model.HistoricalData, years int) model.FeatureSet {
	return s.calculateFeatures(ctx, current, historical, years)
}

// GetHistoricalDataForPeriod retrieves historical data for a specific cluster and time period
func (s *PredictionService) GetHistoricalDataForPeriod(
	ctx context.Context,
	bounds model.Bounds,
	startDate time.Time,
	endDate time.Time,
	shopType string,
) ([]model.HistoricalData, error) {
	bbox := fmt.Sprintf("%f,%f,%f,%f", bounds.MinLat, bounds.MinLon, bounds.MaxLat, bounds.MaxLon)

	// Создаем слайс для хранения исторических данных
	var historicalData []model.HistoricalData

	// Получаем данные за каждый год в периоде
	currentDate := startDate
	for currentDate.Before(endDate) || currentDate.Equal(endDate) {
		// Форматируем дату для Overpass
		dateStr := currentDate.Format("2006-01-02T15:04:05Z")

		// Получаем данные за текущий год
		elements, err := s.overpassRepo.GetCommercialDataByDate(ctx, bbox, shopType, dateStr)
		if err != nil {
			return nil, fmt.Errorf("failed to get data for date %s: %w", dateStr, err)
		}

		// Если это не первый год, вычисляем новые и закрытые объекты
		var newObjects, closedObjects int
		if len(historicalData) > 0 {
			prevElementsSet := make(map[int64]struct{})
			for _, el := range elements {
				prevElementsSet[el.ID] = struct{}{}
			}

			// Вычисляем новые объекты
			for id := range prevElementsSet {
				if _, exists := prevElementsSet[id]; !exists {
					newObjects++
				}
			}

			// Вычисляем закрытые объекты
			for id := range prevElementsSet {
				if _, exists := prevElementsSet[id]; !exists {
					closedObjects++
				}
			}
		}

		// Добавляем данные за текущий год
		historicalData = append(historicalData, model.HistoricalData{
			Period:        currentDate.Format("2006-01-02"),
			BBox:          bbox,
			TotalObjects:  len(elements),
			NewObjects:    newObjects,
			ClosedObjects: closedObjects,
		})

		// Переходим к следующему году
		currentDate = currentDate.AddDate(1, 0, 0)
	}

	return historicalData, nil
}

// CalculateFeaturesForPeriod calculates spatial and temporal features for the given data and time period
func (s *PredictionService) CalculateFeaturesForPeriod(
	ctx context.Context,
	current []model.OSMElement,
	historical []model.HistoricalData,
	startDate time.Time,
	endDate time.Time,
	bounds model.Bounds,
) model.FeatureSet {
	spatialAnalyzer := SpatialAnalyzer{}
	temporalAnalyzer := TemporalAnalyzer{}

	// Если нет текущих элементов, возвращаем пустой набор признаков
	if len(current) == 0 {
		return model.FeatureSet{
			Spatial:  model.SpatialFeatures{},
			Temporal: temporalAnalyzer.Analyze(historical, int(endDate.Sub(startDate).Hours()/(24*365.25))),
			Elements: current,
		}
	}

	// Формируем bbox строку из переданных границ
	bbox := fmt.Sprintf("%f,%f,%f,%f",
		bounds.MinLat,
		bounds.MinLon,
		bounds.MaxLat,
		bounds.MaxLon)

	log.Printf("Using bbox for queries: %s", bbox)

	// Получаем данные о метро для текущего кластера и периода
	subways, err := s.overpassRepo.GetSubwayData(ctx, bbox, endDate.Format("2006-01-02T15:04:05Z"))
	if err != nil {
		log.Printf("Warning: failed to get subway data: %v", err)
	}

	// Получаем данные о дорогах для текущего кластера и периода
	roads, err := s.overpassRepo.GetRoadData(ctx, bbox, endDate.Format("2006-01-02T15:04:05Z"))
	if err != nil {
		log.Printf("Warning: failed to get road data: %v", err)
	}

	spatial := spatialAnalyzer.Analyze(current, subways, roads)

	// Calculate years for temporal analysis
	years := endDate.Sub(startDate).Hours() / (24 * 365.25)
	temporal := temporalAnalyzer.Analyze(historical, int(years))

	return model.FeatureSet{
		Spatial:  spatial,
		Temporal: temporal,
		Elements: current,
	}
}

// GetAvailableModels возвращает список доступных моделей
func (s *PredictionService) GetAvailableModels() ([]model.ModelInfo, error) {
	return s.mlClient.GetAvailableModels()
}

// GetPrediction получает предсказание для указанной области
func (s *PredictionService) GetPrediction(bbox string, shopType string, predictionYear string, trainPeriod string) (*model.Prediction, error) {
	// Формируем имя модели
	modelName := fmt.Sprintf("model_%s_%s", shopType, trainPeriod)

	// Получаем предсказание от ML сервиса
	prediction, err := s.mlClient.GetPrediction(bbox, modelName, predictionYear)
	if err != nil {
		return nil, fmt.Errorf("failed to get prediction: %w", err)
	}

	return prediction, nil
}
