package core

import (
	"context"
	"fmt"
	"log"
	"osm_service/internal/domain/model"
	"osm_service/internal/domain/repository"
	"time"
)

type PredictionService struct {
	overpassRepo     repository.OverpassRepository
	postgisRepo      repository.PostGISRepository
	mlClient         MLClient
	trainingRecorder repository.TrainingDataRecorder
	saveData         bool
}

func NewPredictionService(
	overpassRepo repository.OverpassRepository,
	postgisRepo repository.PostGISRepository,
	mlClient MLClient,
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

func (s *PredictionService) Predict(ctx context.Context, req model.PredictionRequest) (*model.PredictionResult, error) {
	historical, err := s.getHistoricalData(ctx, req.BBox, req.Years, req.ShopType)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical data: %w", err)
	}

	current, err := s.overpassRepo.GetCommercialData(ctx, req.BBox, req.ShopType)
	if err != nil {
		return nil, fmt.Errorf("failed to get current data: %w", err)
	}

	features := s.calculateFeatures(ctx, current, historical, req.Years)

	if s.saveData {
		if err := s.trainingRecorder.SaveTrainingData(ctx, features, req.ShopType, req.BBox, 0); err != nil {
			log.Printf("Failed to save training data: %v", err)
		}
	}

	result, err := s.mlClient.Predict(ctx, features, req.ShopType, historical, req.Years)
	if err != nil {
		return nil, fmt.Errorf("prediction failed: %w", err)
	}

	return result, nil
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
			Period:        startDate.Format("2006"),
			TotalObjects:  len(startElements),
			NewObjects:    0,
			ClosedObjects: 0,
		},
		{
			Period:        endDate.Format("2006"),
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

	subways, err := s.overpassRepo.GetSubwayData(ctx)
	if err != nil {
		log.Printf("Warning: failed to get subway data: %v", err)
	}

	roads, err := s.overpassRepo.GetRoadData(ctx)
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
