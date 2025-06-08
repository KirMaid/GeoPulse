package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"osm_service/internal/domain/model"

	"github.com/jmoiron/sqlx"
)

type TrainingDataRecorder interface {
	SaveTrainingData(ctx context.Context, features model.FeatureSet, shopType string, bbox string, activityLevel float64) error
}

type PostgresTrainingRecorder struct {
	db *sqlx.DB
}

func NewPostgresTrainingRecorder(db *sqlx.DB) *PostgresTrainingRecorder {
	return &PostgresTrainingRecorder{db: db}
}

func (r *PostgresTrainingRecorder) SaveTrainingData(
	ctx context.Context,
	features model.FeatureSet,
	shopType string,
	bbox string,
	activityLevel float64,
) error {
	const query = `
		INSERT INTO training_data (
			shop_type, bbox,
			total_objects, avg_area, subway_stations,
			avg_dist_to_subway, avg_dist_to_primary,
			object_density, new_object_rate, trend_slope,
			elements, activity_level, recorded_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW()
		)`

	spatial := features.Spatial
	temporal := features.Temporal

	// Сериализация элементов в JSON
	elementsJSON, err := json.Marshal(features.Elements)
	if err != nil {
		return fmt.Errorf("failed to marshal elements: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		shopType, bbox,
		spatial.TotalObjects, spatial.AvgArea, spatial.SubwayStations,
		spatial.AvgDistToSubway, spatial.AvgDistToPrimary,
		temporal.ObjectDensity, temporal.NewObjectRate, temporal.TrendSlope,
		elementsJSON, activityLevel,
	)
	return err
}
