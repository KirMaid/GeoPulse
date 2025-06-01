package core

import (
	"context"
	"fmt"
	"osm_service/internal/domain/model"
	"osm_service/internal/domain/repository"
	"time"
)

type PredictionService struct {
	overpassRepo repository.OverpassRepository
	postgisRepo  repository.PostGISRepository
	mlClient     MLClient
}

func NewPredictionService(
	overpassRepo *repository.OverpassRepository,
	postgisRepo *repository.PostGISRepository,
	mlClient MLClient,
) *PredictionService {
	return &PredictionService{
		overpassRepo: *overpassRepo,
		postgisRepo:  *postgisRepo,
		mlClient:     mlClient,
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

	features := s.calculateFeatures(current, historical, req.Years)

	// Pass historical data and years to ML client
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
	end := time.Now()
	start := end.AddDate(-years, 0, 0)

	periods := s.generatePeriods(start, end)
	var data []model.HistoricalData

	for _, period := range periods {
		historical, err := s.postgisRepo.GetByPeriod(ctx, bbox, period, shopType)
		if err != nil {
			return nil, err
		}
		data = append(data, historical)
	}

	return data, nil
}

func (s *PredictionService) generatePeriods(start, end time.Time) []string {
	var periods []string
	for current := start; current.Before(end); current = current.AddDate(0, 3, 0) {
		periods = append(periods, fmt.Sprintf("%d-Q%d", current.Year(), (current.Month()-1)/3+1))
	}
	return periods
}

func (s *PredictionService) calculateFeatures(
	current []model.OSMElement,
	historical []model.HistoricalData,
	years int,
) model.FeatureSet {
	spatialAnalyzer := SpatialAnalyzer{}
	temporalAnalyzer := TemporalAnalyzer{}

	subways, _ := s.overpassRepo.GetSubwayData(context.Background())
	roads, _ := s.overpassRepo.GetRoadData(context.Background())

	spatial := spatialAnalyzer.Analyze(current, subways, roads)
	temporal := temporalAnalyzer.Analyze(historical, years)

	return model.FeatureSet{
		Spatial:  spatial,
		Temporal: temporal,
	}
}
