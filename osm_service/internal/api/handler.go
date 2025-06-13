package api

import (
	"encoding/json"
	"net/http"
	"osm_service/internal/core"
	"osm_service/internal/domain/model"
)

type Handler struct {
	service *core.PredictionService
}

func NewHandler(service *core.PredictionService) *Handler {
	return &Handler{service: service}
}

type PredictionRequest struct {
	BBox     string `json:"bbox"`
	Years    int    `json:"years"`
	ShopType string `json:"shop_type"`
}

type PredictionResponse struct {
	ActivityLevel float64   `json:"activity_level"`
	Trend         Trend     `json:"trend"`
	Hotspots      []Hotspot `json:"hotspots"`
}

type Trend struct {
	Slope    float64 `json:"slope"`
	Strength float64 `json:"strength"`
}

type Hotspot struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Score     float64 `json:"score"`
}

func convertHotspots(hotspots []model.Hotspot) []Hotspot {
	result := make([]Hotspot, len(hotspots))
	for i, h := range hotspots {
		result[i] = Hotspot{
			Latitude:  h.Lat,
			Longitude: h.Lon,
			Score:     h.Score,
		}
	}
	return result
}

func (h *Handler) Predict(w http.ResponseWriter, r *http.Request) {
	var req PredictionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Years < 5 || req.Years > 20 {
		http.Error(w, "Years must be between 1 and 20", http.StatusBadRequest)
		return
	}

	result, err := h.service.Predict(r.Context(), model.PredictionRequest{
		BBox:     req.BBox,
		Years:    req.Years,
		ShopType: req.ShopType,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(PredictionResponse{
		ActivityLevel: result.ActivityLevel,
		Trend: Trend{
			Slope:    result.TrendSlope,
			Strength: result.TrendStrength,
		},
		Hotspots: convertHotspots(result.Hotspots),
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
