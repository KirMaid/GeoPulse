package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"osm_service/internal/domain/model"
)

type mlClient struct {
	client  *http.Client
	baseURL string
}

func NewMLClient(baseURL string) model.MLClient {
	return &mlClient{
		client:  &http.Client{},
		baseURL: baseURL,
	}
}

// GetAvailableModels returns a list of available models
func (c *mlClient) GetAvailableModels() ([]model.ModelInfo, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/models", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to get models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ML service returned error: %s", resp.Status)
	}

	var models []model.ModelInfo
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return models, nil
}

// GetPrediction gets a prediction for the specified area
func (c *mlClient) GetPrediction(bbox string, modelName string, predictionYear string) (*model.Prediction, error) {
	// Form request body
	reqBody := map[string]interface{}{
		"bbox":            bbox,
		"model_name":      modelName,
		"prediction_year": predictionYear,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(context.Background(), "POST", c.baseURL+"/predict", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ML service returned error: %s", resp.Status)
	}

	// Decode response
	var result model.Prediction
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
