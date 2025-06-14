package model

// MLClient определяет интерфейс для взаимодействия с ML сервисом
type MLClient interface {
	// GetAvailableModels возвращает список доступных моделей
	GetAvailableModels() ([]ModelInfo, error)

	// GetPrediction получает предсказание для указанной области
	GetPrediction(bbox string, modelName string, predictionYear string) (*Prediction, error)
}

// ModelInfo содержит информацию о модели
type ModelInfo struct {
	Name      string `json:"name"`
	ShopType  string `json:"shop_type"`
	TrainYear string `json:"train_year"`
	BBox      string `json:"bbox"`
	Metrics   struct {
		MSE  float64 `json:"mse"`
		RMSE float64 `json:"rmse"`
		R2   float64 `json:"r2"`
	} `json:"metrics"`
}

// Prediction содержит результат предсказания
type Prediction struct {
	ActivityLevel   float64   `json:"activity_level"`
	Trend           float64   `json:"trend"`
	PredictedNew    int       `json:"predicted_new"`
	PredictedClosed int       `json:"predicted_closed"`
	TotalObjects    int       `json:"total_objects"`
	ModelUsed       string    `json:"model_used"`
	Hotspots        []Hotspot `json:"hotspots"`
}

// Hotspot представляет точку интереса
type Hotspot struct {
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
	Score float64 `json:"score"`
}
