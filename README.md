# Анализ и прогнозирование развития городской инфраструктуры

## Описание проекта
Проект представляет собой систему для анализа и прогнозирования развития городской инфраструктуры на основе данных OpenStreetMap. Система включает в себя сервисы для сбора данных, их обработки, обучения моделей машинного обучения и предоставления прогнозов.

## Структура проекта
```
.
├── ml_service/                # Сервис машинного обучения
│   ├── train_models.py       # Скрипт для обучения моделей
│   └── predict.py            # API для получения прогнозов
├── osm_service/              # Сервис работы с OSM данными
│   ├── cmd/                  # Точка входа в приложение
│   ├── internal/            # Внутренняя логика сервиса
│   └── pkg/                 # Публичные пакеты
├── datasets/                # Директория с датасетами
│   └── dataset_*.json      # JSON файлы с данными
├── models/                  # Директория с обученными моделями
│   ├── model_*.joblib      # Файлы моделей
│   ├── scaler_*.joblib     # Файлы скейлеров
│   └── training_metrics.json # Метрики обучения
├── Dockerfile              # Конфигурация Docker
└── README.md              # Документация проекта
```

## Установка и запуск

### Требования
- Python 3.8+
- Go 1.21+
- Docker

### Запуск сервиса машинного обучения
1. Установите зависимости Python:
```bash
pip install -r requirements.txt
```

2. Запустите сервис:
```bash
python ml_service/predict.py
```

Сервис будет доступен по адресу `http://localhost:5000`

### Запуск сервиса OSM
1. Соберите Go приложение:
```bash
cd osm_service
go build -o osm_service cmd/main.go
```

2. Запустите сервис:
```bash
./osm_service
```

## API Endpoints

### ML Service

#### GET /health
Проверка работоспособности сервиса.

#### GET /models
Получение списка доступных моделей.

#### POST /predict
Получение прогноза для указанной области.

Пример запроса:
```json
{
    "bbox": "55.78,37.58,55.80,37.60",
    "shop_type": "restaurant",
    "prediction_year": "2024",
    "train_period": "20190101_to_20231231"
}
```

Параметры:
- `bbox`: координаты области в формате "min_lat,min_lon,max_lat,max_lon"
- `shop_type`: тип заведения (например, "restaurant")
- `prediction_year`: год для прогноза
- `train_period`: период обучения в формате "YYYYMMDD_to_YYYYMMDD"

Пример ответа:
```json
{
    "activity_level": 0.75,
    "trend": 0.2,
    "predicted_new": 5,
    "predicted_closed": 2,
    "total_objects": 20,
    "model_used": "model_restaurant_2019-2023_55.783333_37.575000_55.800000_37.600000",
    "bbox": "55.78,37.58,55.80,37.60",
    "shop_type": "restaurant",
    "prediction_year": "2024",
    "train_period": "20190101_to_20231231"
}
```

### OSM Service

#### GET /api/health
Проверка работоспособности сервиса.

#### GET /api/models
Получение списка доступных моделей.

#### POST /api/predict
Получение прогноза для указанной области.

## Обучение моделей

Для обучения новых моделей используйте скрипт `train_models.py`:

```bash
python ml_service/train_models.py
```

Скрипт:
1. Загружает датасеты из директории `datasets/`
2. Обучает модели для каждого типа заведения и периода
3. Сохраняет модели и скейлеры в директорию `models/`
4. Генерирует файл с метриками обучения `training_metrics.json`

## Формат датасетов

Датасеты хранятся в формате JSON в директории `datasets/` с именами вида:
```
dataset_{shop_type}_{start_date}_to_{end_date}.json
```

Например:
```
dataset_restaurant_20190101_to_20231231.json
```

Пример структуры датасета:
```json
{
    "clusters": [
        {
            "bbox": "55.783333,37.575000,55.800000,37.600000",
            "data": [
                {
                    "year": "2023",
                    "data": [
                        {
                            "avg_area": 150.5,
                            "avg_dist_to_primary": 200.0,
                            "avg_dist_to_subway": 500.0,
                            "closure_rate": 0.1,
                            "new_object_rate": 0.2,
                            "object_density": 0.5,
                            "total_objects": 100
                        }
                    ]
                },
                {
                    "year": "2022",
                    "data": [
                        {
                            "avg_area": 145.0,
                            "avg_dist_to_primary": 195.0,
                            "avg_dist_to_subway": 490.0,
                            "closure_rate": 0.08,
                            "new_object_rate": 0.15,
                            "object_density": 0.45,
                            "total_objects": 95
                        }
                    ]
                }
            ]
        }
    ]
}
```

## Формат моделей

Модели сохраняются в директории `models/` с именами вида:
```
model_{shop_type}_{train_period}_{min_lat}_{min_lon}_{max_lat}_{max_lon}.joblib
scaler_{shop_type}_{train_period}_{min_lat}_{min_lon}_{max_lat}_{max_lon}.joblib
```

Например:
```
model_restaurant_2019-2023_55.783333_37.575000_55.800000_37.600000.joblib
scaler_restaurant_2019-2023_55.783333_37.575000_55.800000_37.600000.joblib
```
