package main

import (
	"log"
	"net/http"
	"os"
	"osm_service/internal/api"
	"osm_service/internal/core"
	"osm_service/internal/domain/repository"
	"osm_service/internal/infrastructure/mlclient"
	"time"
)

func main() {
	//testPostgresUrl := "postgresql://user:pass@postgres/osm?sslmode=disable"
	testPostgresUrl := "postgresql://user:pass@localhost:5432/osm?sslmode=disable"
	testOverpassAPI := "https://maps.mail.ru/osm/tools/overpass/api/interpreter"
	//postgresRepo := repository.NewPostgresRepository(os.Getenv("POSTGRES_URL"))
	postgresRepo := repository.NewPostgresRepository(testPostgresUrl)
	overpassRepo := repository.NewOverpassRepository(testOverpassAPI, 15*time.Second)
	//overpassRepo := repository.NewOverpassRepository(os.Getenv("OVERPASS_URL"), 5)
	mlClient := mlclient.NewHTTPMLClient(os.Getenv("ML_SERVICE_URL"))

	// Инициализация рекордера для обучения
	trainingRecorder := repository.NewPostgresTrainingRecorder(postgresRepo.DB)
	saveTrainingData := os.Getenv("SAVE_TRAINING_DATA") == "true"

	// Создание сервиса предсказаний
	predictionService := core.NewPredictionService(
		*overpassRepo,
		*postgresRepo,
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
