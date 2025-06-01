package main

import (
	"log"
	"net/http"
	"os"
	"osm_service/internal/api"
	"osm_service/internal/core"
	"osm_service/internal/domain/repository"
	"osm_service/internal/infrastructure/mlclient"
)

func main() {
	postgresRepo := repository.NewPostgresRepository(os.Getenv("POSTGRES_URL"))
	overpassRepo := repository.NewOverpassRepository(os.Getenv("OVERPASS_URL"), 5)
	mlClient := mlclient.NewHTTPMLClient(os.Getenv("ML_SERVICE_URL"))

	predictionService := core.NewPredictionService(
		overpassRepo,
		postgresRepo,
		mlClient,
	)

	handler := api.NewHandler(predictionService)
	http.HandleFunc("/api/predict", handler.Predict)

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
