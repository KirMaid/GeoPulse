package mlclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"osm_service/internal/domain/model"
)

type HTTPMLClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPMLClient(baseURL string) *HTTPMLClient {
	return &HTTPMLClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

// GetAvailableModels возвращает список доступных моделей
func (c *HTTPMLClient) GetAvailableModels() ([]model.ModelInfo, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/models", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("error getting models: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var models []model.ModelInfo
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return models, nil
}

// GetPrediction получает предсказание для указанной области
func (c *HTTPMLClient) GetPrediction(bbox string, modelName string, predictionYear string) (*model.Prediction, error) {
	// Формируем запрос
	reqBody := map[string]interface{}{
		"bbox":            bbox,
		"shop_type":       modelName,
		"prediction_year": predictionYear,
		"train_period":    "2010-2024",
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	// Отправляем запрос
	resp, err := c.client.Post(
		fmt.Sprintf("%s/predict", c.baseURL),
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Декодируем ответ
	var prediction model.Prediction
	if err := json.NewDecoder(resp.Body).Decode(&prediction); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &prediction, nil
}
