# Система предсказания коммерческой активности

Система для анализа и предсказания коммерческой активности на основе исторических данных OpenStreetMap.

## Архитектура

Система состоит из двух основных компонентов:

### 1. Go сервис (osm_service)

Основной сервис, который предоставляет API для:
- Получения предсказаний коммерческой активности
- Создания датасетов для обучения моделей

#### API Endpoints

##### POST /api/predict
Получение предсказания коммерческой активности для указанной области.

Параметры запроса:
```json
{
    "bbox": "55.751244,37.618423,55.755244,37.622423",  // Границы области (min_lat,min_lon,max_lat,max_lon)
    "shop_type": "supermarket",                         // Тип магазина
    "years": 5                                          // Период предсказания в годах
}
```

Ответ:
```json
{
    "activity_level": 0.85,           // Уровень активности (0-1)
    "trend_slope": 0.1,              // Наклон тренда
    "trend_strength": 0.8,           // Сила тренда (0-1)
    "hotspots": [                    // Горячие точки
        {
            "lat": 55.752244,
            "lon": 37.619423,
            "score": 0.9
        }
    ]
}
```

##### POST /api/training
Создание датасета для обучения моделей.

Параметры запроса:
```json
{
    "bbox": "55.751244,37.618423,55.755244,37.622423",  // Границы области
    "shop_type": "supermarket",                         // Тип магазина
    "cluster_size": 0.5,                                // Размер кластера в км²
    "start_date": "2020-01-01",                         // Начальная дата
    "end_date": "2024-01-01"                            // Конечная дата
}
```

Ответ:
```json
{
    "message": "Dataset generated successfully",
    "path": "datasets/dataset_20200101_to_20240101.json"
}
```

### 2. Python ML сервис (ml_service)

Сервис машинного обучения, который:
- Обучает модели на основе исторических данных
- Делает предсказания на основе текущих данных

#### Основные компоненты:

1. **ModelTrainer** (`train_models.py`)
   - Загрузка и подготовка данных
   - Обучение моделей для разных типов магазинов
   - Сохранение обученных моделей

2. **PredictionService** (`predict.py`)
   - Загрузка обученных моделей
   - Расчет признаков
   - Предсказание коммерческой активности

## Установка и запуск

### Требования

- Go 1.21+
- Python 3.9+
- PostgreSQL 14+
- Docker и Docker Compose

### Установка

1. Клонировать репозиторий:
```bash
git clone https://github.com/yourusername/commercial-activity-prediction.git
cd commercial-activity-prediction
```

2. Установить зависимости Go:
```bash
cd osm_service
go mod download
```

3. Установить зависимости Python:
```bash
cd ml_service
pip install -r requirements.txt
```

### Запуск

1. Запустить PostgreSQL:
```bash
docker-compose up -d postgres
```

2. Запустить Go сервис:
```bash
cd osm_service
go run cmd/main.go
```

3. Запустить ML сервис:
```bash
cd ml_service
python predict.py
```

## Конфигурация

### Go сервис

Переменные окружения:
- `POSTGRES_URL` - URL для подключения к PostgreSQL
- `OVERPASS_URL` - URL Overpass API
- `ML_SERVICE_URL` - URL ML сервиса

### ML сервис

Переменные окружения:
- `MODEL_PATH` - путь к сохраненным моделям
- `DATASET_PATH` - путь к датасетам

## Структура проекта

```
.
├── osm_service/                 # Go сервис
│   ├── cmd/                     # Точка входа
│   │   ├── internal/                # Внутренний код
│   │   │   ├── api/                # HTTP обработчики
│   │   │   ├── core/               # Бизнес-логика
│   │   │   ├── domain/             # Доменные модели
│   │   │   └── infrastructure/     # Инфраструктурный код
│   │   └── go.mod
│   └── docker-compose.yml          # Конфигурация Docker
```

## Разработка

### Добавление новых типов магазинов

1. Добавить новый тип в `osm_service/internal/domain/model/models.go`
2. Создать датасет через `/api/training`
3. Обучить модель через `ml_service/train_models.py`

### Расширение функциональности

1. Добавить новые признаки в `osm_service/internal/core/temporal.go` и `spatial.go`
2. Обновить модели в ML сервисе
3. Обновить API документацию

## Лицензия

MIT

# ML Service API Documentation

## Endpoints

### 1. Get Available Models
```http
GET /models
```
Returns a list of available trained models with their training periods.

Response:
```json
[
    {
        "name": "model_2020_2024",
        "train_year": "2020_2024"
    }
]
```

### 2. Make Prediction
```http
POST /predict
```
Makes a prediction using the specified model.

Request body:
```json
{
    "model_name": "model_2020_2024",
    "features": {
        "avg_area": 150.5,          // Средняя площадь объектов
        "avg_dist_to_primary": 200.0, // Среднее расстояние до основных дорог
        "avg_dist_to_subway": 500.0,  // Среднее расстояние до метро
        "closure_rate": 0.1,         // Процент закрытых объектов
        "new_object_rate": 0.2,      // Процент новых объектов
        "object_density": 0.5,       // Плотность объектов
        "total_objects": 100         // Общее количество объектов
    }
}
```

Response:
```json
{
    "activity_level": 0.75,  // Уровень активности (0-1)
    "trend": 0.5            // Тренд (-1 до 1, где -1 - падение, 1 - рост)
}
```

### 3. Health Check
```http
GET /health
```
Returns the health status of the service.

Response:
```json
{
    "status": "healthy"
}
```

## Error Responses

All endpoints may return the following error responses:

### 400 Bad Request
```json
{
    "error": "Missing required fields"
}
```

### 404 Not Found
```json
{
    "error": "Model not found"
}
```

### 500 Internal Server Error
```json
{
    "error": "Internal server error"
}
```

## Features Description

The prediction service uses the following features:

1. `avg_area` - Average area of commercial objects in the cluster
2. `avg_dist_to_primary` - Average distance to primary roads
3. `avg_dist_to_subway` - Average distance to subway stations
4. `closure_rate` - Rate of object closures (0-1)
5. `new_object_rate` - Rate of new objects (0-1)
6. `object_density` - Density of objects in the cluster
7. `total_objects` - Total number of objects in the cluster

## Model Training

Models are trained using historical data and saved with year suffixes (e.g., `model_2020_2024.joblib`). Each model is trained on data from a specific time period and can be used to make predictions for similar periods.

## Response Interpretation

- `activity_level` (0-1):
  - 0: Low activity
  - 1: High activity

- `trend` (-1 to 1):
  - -1: Strong negative trend
  - 0: Stable
  - 1: Strong positive trend
