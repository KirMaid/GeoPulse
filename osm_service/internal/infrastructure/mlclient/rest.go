package mlclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"osm_service/internal/domain/model"
	"time"
)

type HTTPMLClient struct {
	endpoint string
	client   *http.Client
}

func NewHTTPMLClient(endpoint string) *HTTPMLClient {
	return &HTTPMLClient{
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type MLRequest struct {
	Features       model.FeatureSet       `json:"features"`
	ShopType       string                 `json:"shop_type"`
	HistoricalData []model.HistoricalData `json:"historical_data"`
	Years          int                    `json:"years"`
}

type MLResponse struct {
	ActivityLevel float64         `json:"activity_level"`
	TrendSlope    float64         `json:"trend_slope"`
	TrendStrength float64         `json:"trend_strength"`
	Hotspots      []model.Hotspot `json:"hotspots"`
}

func (c *HTTPMLClient) Predict(ctx context.Context, features model.FeatureSet, shopType string, historical []model.HistoricalData, years int) (*model.PredictionResult, error) {
	reqBody := MLRequest{
		Features:       features,
		ShopType:       shopType,
		HistoricalData: historical,
		Years:          years,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ML request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create ML request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ML service request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ML service returned status: %d", resp.StatusCode)
	}

	var mlResp MLResponse
	if err := json.NewDecoder(resp.Body).Decode(&mlResp); err != nil {
		return nil, fmt.Errorf("failed to decode ML response: %w", err)
	}

	return &model.PredictionResult{
		ActivityLevel: mlResp.ActivityLevel,
		TrendSlope:    mlResp.TrendSlope,
		TrendStrength: mlResp.TrendStrength,
		Hotspots:      mlResp.Hotspots,
	}, nil
}
