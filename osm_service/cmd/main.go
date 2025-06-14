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
	// Конфигурация
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		postgresURL = "postgresql://user:pass@localhost:5432/osm?sslmode=disable"
	}

	overpassAPI := os.Getenv("OVERPASS_URL")
	if overpassAPI == "" {
		overpassAPI = "https://maps.mail.ru/osm/tools/overpass/api/interpreter"
	}

	mlServiceURL := os.Getenv("ML_SERVICE_URL")
	if mlServiceURL == "" {
		mlServiceURL = "http://localhost:5000"
	}

	// Инициализация репозиториев
	postgresRepo := repository.NewPostgresRepository(postgresURL)
	overpassRepo := repository.NewOverpassRepository(overpassAPI, 120*time.Second)
	mlClient := mlclient.NewHTTPMLClient(mlServiceURL)

	// Создание сервиса предсказаний
	predictionService := core.NewPredictionService(
		*overpassRepo,
		*postgresRepo,
		mlClient,
		nil, // Отключаем сохранение данных для обучения
		false,
	)

	// Настройка HTTP-обработчиков
	handler := api.NewHandler(predictionService)

	// Добавляем middleware для логирования
	http.HandleFunc("/api/predict", logMiddleware(handler.Predict))
	http.HandleFunc("/api/training", logMiddleware(handler.Training))
	http.HandleFunc("/api/models", logMiddleware(handler.GetModels))

	// Запуск сервера
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// logMiddleware добавляет логирование запросов
func logMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s", r.Method, r.URL.Path)

		// Создаем ResponseWriter, который перехватывает статус код
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next(rw, r)

		log.Printf("Completed %s %s %d in %v",
			r.Method,
			r.URL.Path,
			rw.statusCode,
			time.Since(start))
	}
}

// responseWriter перехватывает статус код ответа
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
