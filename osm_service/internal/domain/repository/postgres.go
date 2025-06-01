package repository

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	"osm_service/internal/domain/model"
	"strconv"
	"strings"
)

type PostGISRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(connStr string) *PostGISRepository {
	db := sqlx.MustConnect("postgres", connStr)
	return &PostGISRepository{db: db}
}

func (r *PostGISRepository) GetByPeriod(
	ctx context.Context,
	bbox string,
	period string,
	shopType string,
) (model.HistoricalData, error) {
	// Parse bbox string: "lat1,lon1,lat2,lon2"
	minLat, minLon, maxLat, maxLon, err := parseBBox(bbox)
	if err != nil {
		return model.HistoricalData{}, fmt.Errorf("invalid bbox format: %w", err)
	}

	const query = `
		SELECT
			period,
			total_objects,
			new_objects,
			closed_objects
		FROM commercial_features
		WHERE period = $1
		AND shop_type = $2
		AND ST_Intersects(bbox, ST_MakeEnvelope($3, $4, $5, $6, 4326))`

	var data model.HistoricalData
	err = r.db.GetContext(ctx, &data, query,
		period,
		shopType,
		minLon, minLat, maxLon, maxLat,
	)
	if err != nil {
		return model.HistoricalData{}, fmt.Errorf("failed to query historical data: %w", err)
	}

	return data, nil
}

// parseBBox parses a bbox string in format "lat1,lon1,lat2,lon2" into minLat, minLon, maxLat, maxLon.
func parseBBox(bbox string) (minLat, minLon, maxLat, maxLon float64, err error) {
	parts := strings.Split(bbox, ",")
	if len(parts) != 4 {
		return 0, 0, 0, 0, fmt.Errorf("bbox must have 4 components, got %d", len(parts))
	}

	minLat, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("invalid minLat: %w", err)
	}
	minLon, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("invalid minLon: %w", err)
	}
	maxLat, err = strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("invalid maxLat: %w", err)
	}
	maxLon, err = strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("invalid maxLon: %w", err)
	}

	// Validate ranges
	if minLat < -90 || minLat > 90 || maxLat < -90 || maxLat > 90 {
		return 0, 0, 0, 0, fmt.Errorf("latitude out of range [-90, 90]")
	}
	if minLon < -180 || minLon > 180 || maxLon < -180 || maxLon > 180 {
		return 0, 0, 0, 0, fmt.Errorf("longitude out of range [-180, 180]")
	}
	if minLat > maxLat || minLon > maxLon {
		return 0, 0, 0, 0, fmt.Errorf("minLat must be <= maxLat and minLon must be <= maxLon")
	}

	return minLat, minLon, maxLat, maxLon, nil
}
