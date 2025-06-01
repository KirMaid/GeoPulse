package core

import (
	"context"
	"osm_service/internal/domain/model"
)

type MLClient interface {
	Predict(ctx context.Context, features model.FeatureSet, shopType string, historical []model.HistoricalData, years int) (*model.PredictionResult, error)
}
