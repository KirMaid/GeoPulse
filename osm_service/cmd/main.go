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
	// Инициализация репозиториев
	postgresRepo := repository.NewPostgresRepository(os.Getenv("POSTGRES_URL"))
	overpassRepo := repository.NewOverpassRepository(os.Getenv("OVERPASS_URL"), 5)
	mlClient := mlclient.NewHTTPMLClient(os.Getenv("ML_SERVICE_URL"))

	// Инициализация рекордера для обучения
	trainingRecorder := repository.NewPostgresTrainingRecorder(postgresRepo.DB)
	saveTrainingData := os.Getenv("SAVE_TRAINING_DATA") == "true"

	// Создание сервиса предсказаний
	predictionService := core.NewPredictionService(
		overpassRepo,
		postgresRepo,
		mlClient,
		trainingRecorder,
		saveTrainingData,
	)

	// Настройка HTTP-обработчиков
	handler := api.NewHandler(predictionService)
	http.HandleFunc("/api/predict", handler.Predict)

	// Запуск сервера
	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
