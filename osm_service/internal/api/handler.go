package api

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"osm_service/internal/core"
	"osm_service/internal/domain/model"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Dataset types
type ClusterData struct {
	Index int        `json:"index"`
	Bbox  string     `json:"bbox"`
	Data  []YearData `json:"data"`
}

type YearData struct {
	Year int           `json:"year"`
	Data []FeatureData `json:"data"`
}

type FeatureData struct {
	AvgArea          float64 `json:"avg_area"`
	AvgDistToPrimary float64 `json:"avg_dist_to_primary"`
	AvgDistToSubway  float64 `json:"avg_dist_to_subway"`
	ClosureRate      float64 `json:"closure_rate"`
	NewObjectRate    float64 `json:"new_object_rate"`
	ObjectDensity    float64 `json:"object_density"`
	TotalObjects     int     `json:"total_objects"`
}

type Handler struct {
	service *core.PredictionService
}

func NewHandler(service *core.PredictionService) *Handler {
	return &Handler{service: service}
}

type PredictionRequest struct {
	Index int    `json:"index"`
	Bbox  string `json:"bbox"`
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

type TrainingRequest struct {
	BBox        string  `json:"bbox"`         // Area coordinates
	ShopType    string  `json:"shop_type"`    // Type of objects to predict
	ClusterSize float64 `json:"cluster_size"` // Size of cluster in square kilometers
	StartDate   string  `json:"start_date"`   // Start date in format "2006-01-02"
	EndDate     string  `json:"end_date"`     // End date in format "2006-01-02"
}

type TrainingResponse struct {
	Message string `json:"message"`
	Path    string `json:"path"`
}

type PredictRequest struct {
	BBox           string `json:"bbox"`
	ShopType       string `json:"shop_type"`
	PredictionYear string `json:"prediction_year"`
	TrainPeriod    string `json:"train_period"`
}

type ModelInfo struct {
	Name      string `json:"name"`
	TrainYear string `json:"train_year"`
	Metrics   struct {
		MSE  float64 `json:"mse"`
		RMSE float64 `json:"rmse"`
		R2   float64 `json:"r2"`
	} `json:"metrics"`
}

func (h *Handler) Predict(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PredictRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.BBox == "" {
		http.Error(w, "BBox is required", http.StatusBadRequest)
		return
	}

	if req.ShopType == "" {
		http.Error(w, "Shop type is required", http.StatusBadRequest)
		return
	}

	if req.PredictionYear == "" {
		http.Error(w, "Prediction year is required", http.StatusBadRequest)
		return
	}

	if req.TrainPeriod == "" {
		http.Error(w, "Train period is required", http.StatusBadRequest)
		return
	}

	// Получаем предсказание
	prediction, err := h.service.GetPrediction(req.BBox, req.ShopType, req.PredictionYear, req.TrainPeriod)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting prediction: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prediction)
}

func (h *Handler) Training(w http.ResponseWriter, r *http.Request) {
	var req TrainingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate dates
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		http.Error(w, "Invalid start_date format. Use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		http.Error(w, "Invalid end_date format. Use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	if endDate.Before(startDate) {
		http.Error(w, "end_date must be after start_date", http.StatusBadRequest)
		return
	}

	// Validate cluster size
	if req.ClusterSize <= 0 {
		http.Error(w, "cluster_size must be positive", http.StatusBadRequest)
		return
	}

	// Generate clusters
	clusters, err := generateClusters(req.BBox, req.ClusterSize)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate clusters: %v", err), http.StatusBadRequest)
		return
	}

	// Generate filename with date range and shop type
	filename := fmt.Sprintf("dataset_%s_%s_to_%s.json",
		req.ShopType,
		startDate.Format("20060102"),
		endDate.Format("20060102"))

	// Create datasets directory in parent directory (same level as Dockerfile)
	datasetsDir := filepath.Join("..", "datasets")
	if err := os.MkdirAll(datasetsDir, 0755); err != nil {
		log.Printf("Error creating datasets directory: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create datasets directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Create full filepath
	filepath := filepath.Join(datasetsDir, filename)
	log.Printf("Saving dataset to: %s", filepath)

	// Create file
	file, err := os.Create(filepath)
	if err != nil {
		log.Printf("Error creating file: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create dataset file: %v", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Create JSON encoder
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	dataset := struct {
		Clusters []ClusterData `json:"clusters"`
	}{
		Clusters: make([]ClusterData, len(clusters)),
	}

	// Process each cluster
	for i, cluster := range clusters {
		log.Printf("Processing cluster %d/%d", i+1, len(clusters))

		dataset.Clusters[i].Index = i
		dataset.Clusters[i].Bbox = fmt.Sprintf("%f,%f,%f,%f",
			cluster.MinLat, cluster.MinLon, cluster.MaxLat, cluster.MaxLon)

		// Get historical data for the cluster
		historical, err := h.service.GetHistoricalDataForPeriod(r.Context(), cluster, startDate, endDate, req.ShopType)
		if err != nil {
			log.Printf("Warning: Failed to get historical data for cluster: %v", err)
			continue
		}

		// Get current data for the cluster
		current, err := h.service.GetCurrentData(r.Context(), cluster, req.ShopType)
		if err != nil {
			log.Printf("Warning: Failed to get current data for cluster: %v", err)
			continue
		}

		// Calculate features with cluster bounds
		features := h.service.CalculateFeaturesForPeriod(r.Context(), current, historical, startDate, endDate, cluster)

		// Create yearly data
		yearlyData := make([]YearData, 0)
		currentYear := startDate.Year()
		endYear := endDate.Year()

		// Create map for yearly historical data
		yearlyHistoricalData := make(map[string]model.HistoricalData)
		for _, data := range historical {
			year := data.Period[:4]
			yearlyHistoricalData[year] = data
		}

		// Sort years for correct trend calculation
		var years []string
		for year := currentYear; year <= endYear; year++ {
			years = append(years, fmt.Sprintf("%d", year))
		}
		sort.Strings(years)

		for _, yearStr := range years {
			yearData, exists := yearlyHistoricalData[yearStr]

			// If no data for specific year, use data from last available year
			if !exists {
				var lastYear string
				for y := range yearlyHistoricalData {
					if y <= yearStr && (lastYear == "" || y > lastYear) {
						lastYear = y
					}
				}
				if lastYear != "" {
					yearData = yearlyHistoricalData[lastYear]
				}
			}

			// Calculate metrics for current year
			totalObjects := yearData.TotalObjects
			newObjects := yearData.NewObjects
			closedObjects := yearData.ClosedObjects

			// Calculate growth and closure rates
			var newObjectRate, closureRate float64
			if totalObjects > 0 {
				newObjectRate = float64(newObjects) / float64(totalObjects)
				closureRate = float64(closedObjects) / float64(totalObjects)
			}

			// Calculate object density
			objectDensity := float64(totalObjects) / features.Spatial.AvgArea

			// Create feature data
			featureData := FeatureData{
				AvgArea:          features.Spatial.AvgArea,
				AvgDistToPrimary: features.Spatial.AvgDistToPrimary,
				AvgDistToSubway:  features.Spatial.AvgDistToSubway,
				ClosureRate:      closureRate,
				NewObjectRate:    newObjectRate,
				ObjectDensity:    objectDensity,
				TotalObjects:     totalObjects,
			}

			// Add year data
			yearInt, _ := strconv.Atoi(yearStr)
			yearlyData = append(yearlyData, YearData{
				Year: yearInt,
				Data: []FeatureData{featureData},
			})
		}

		dataset.Clusters[i].Data = yearlyData
	}

	// Write dataset to file
	if err := encoder.Encode(dataset); err != nil {
		log.Printf("Error writing dataset: %v", err)
		http.Error(w, fmt.Sprintf("Failed to write dataset: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Dataset successfully saved to: %s", filepath)

	// Return success response
	response := TrainingResponse{
		Message: "Dataset created successfully",
		Path:    filepath,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// generateClusters splits the bounding box into smaller clusters
func generateClusters(bbox string, clusterSize float64) ([]model.Bounds, error) {
	// Parse bbox string: "lat1,lon1,lat2,lon2"
	parts := strings.Split(bbox, ",")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid bbox format")
	}

	minLat, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid min_lat: %w", err)
	}
	minLon, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid min_lon: %w", err)
	}
	maxLat, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid max_lat: %w", err)
	}
	maxLon, err := strconv.ParseFloat(parts[3], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid max_lon: %w", err)
	}

	// Calculate number of clusters in each direction
	latDiff := maxLat - minLat
	lonDiff := maxLon - minLon

	// Approximate conversion from degrees to kilometers (at equator)
	latKm := latDiff * 111.32
	lonKm := lonDiff * 111.32 * math.Cos(minLat*math.Pi/180)

	clustersLat := int(math.Ceil(latKm / clusterSize))
	clustersLon := int(math.Ceil(lonKm / clusterSize))

	// Generate clusters
	var clusters []model.Bounds
	latStep := latDiff / float64(clustersLat)
	lonStep := lonDiff / float64(clustersLon)

	for i := 0; i < clustersLat; i++ {
		for j := 0; j < clustersLon; j++ {
			cluster := model.Bounds{
				MinLat: minLat + float64(i)*latStep,
				MinLon: minLon + float64(j)*lonStep,
				MaxLat: minLat + float64(i+1)*latStep,
				MaxLon: minLon + float64(j+1)*lonStep,
			}
			clusters = append(clusters, cluster)
		}
	}

	return clusters, nil
}

// GetModels возвращает список доступных моделей
func (h *Handler) GetModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем список моделей из ML сервиса
	models, err := h.service.GetAvailableModels()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting models: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}
