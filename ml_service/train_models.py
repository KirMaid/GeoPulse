import os
import json
import numpy as np
import pandas as pd
from sklearn.ensemble import RandomForestRegressor
from sklearn.preprocessing import StandardScaler
from sklearn.metrics import mean_squared_error, r2_score
import joblib
import logging
import glob
import re
from typing import Tuple
from sklearn.model_selection import train_test_split

# Настройка логирования
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Определяем колонки признаков
FEATURE_COLUMNS = [
    'avg_area',
    'avg_dist_to_primary',
    'avg_dist_to_subway',
    'closure_rate',
    'new_object_rate',
    'object_density',
    'total_objects'
]

class ModelTrainer:
    def __init__(self):
        # Переходим в корневую директорию проекта
        root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), '..'))
        os.chdir(root_dir)
        
        # Создаем директорию models в корне проекта
        self.models_dir = "models"
        os.makedirs(self.models_dir, exist_ok=True)
        self.metrics = {}
        
    def calculate_activity_level(self, data: dict) -> float:
        """Рассчитывает уровень активности на основе признаков"""
        # Нормализуем object_density, деля на максимальное значение в датасете (примерно 10000)
        normalized_density = data['object_density'] / 10000
        
        # Рассчитываем уровень активности как взвешенную сумму показателей
        activity = (
            data['new_object_rate'] * 0.4 +  # 40% веса для новых объектов
            (1 - data['closure_rate']) * 0.3 +  # 30% веса для стабильности
            normalized_density * 0.3  # 30% веса для плотности
        )
        
        return float(activity)
        
    def load_dataset(self, dataset_path: str) -> Tuple[pd.DataFrame, str, str, str]:
        """Загрузка датасета и извлечение метаданных"""
        try:
            # Извлекаем тип заведения и период из имени файла
            # Формат: dataset_shop_type_YYYYMMDD_to_YYYYMMDD.json
            filename = os.path.basename(dataset_path)
            parts = filename.replace('.json', '').split('_')
            
            if len(parts) < 4:
                raise ValueError(f"Invalid dataset filename format: {filename}")
            
            shop_type = parts[1]
            start_date = parts[2]
            end_date = parts[4]
            
            # Преобразуем даты в годы
            start_year = start_date[:4]
            end_year = end_date[:4]
            year_suffix = f"{start_year}-{end_year}"
            
            # Загружаем данные
            with open(dataset_path, 'r') as f:
                data = json.load(f)
            
            # Преобразуем в DataFrame
            records = []
            for cluster in data['clusters']:
                bbox = cluster['bbox']
                for year_data in cluster['data']:
                    year = year_data['year']
                    for feature_data in year_data['data']:
                        # Рассчитываем уровень активности
                        activity_level = self.calculate_activity_level(feature_data)
                        
                        record = {
                            'year': year,
                            'bbox': bbox,
                            'activity_level': activity_level,
                            **feature_data
                        }
                        records.append(record)
            
            df = pd.DataFrame(records)
            
            # Извлекаем координаты из bbox для формирования суффикса
            bbox_parts = bbox.split(',')
            if len(bbox_parts) == 4:
                bbox_suffix = f"{bbox_parts[0]}_{bbox_parts[1]}_{bbox_parts[2]}_{bbox_parts[3]}"
            else:
                bbox_suffix = bbox.replace(',', '_')
            
            return df, shop_type, year_suffix, bbox_suffix
            
        except Exception as e:
            logger.error(f"Error loading dataset {dataset_path}: {str(e)}")
            raise

    def train_model(self, df: pd.DataFrame, shop_type: str, year_suffix: str, bbox_suffix: str) -> None:
        """Обучение модели для конкретного типа заведения и периода"""
        try:
            logger.info(f"Training model for {shop_type} ({year_suffix}) in area {bbox_suffix}")
            
            # Подготовка данных
            X = df[FEATURE_COLUMNS].values
            y = df['activity_level'].values
            
            # Разделение на обучающую и тестовую выборки
            X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.2, random_state=42)
            
            # Нормализация признаков
            scaler = StandardScaler()
            X_train_scaled = scaler.fit_transform(X_train)
            X_test_scaled = scaler.transform(X_test)
            
            # Обучение модели
            model = RandomForestRegressor(
                n_estimators=100,
                max_depth=10,
                min_samples_split=5,
                min_samples_leaf=2,
                random_state=42
            )
            model.fit(X_train_scaled, y_train)
            
            # Оценка модели
            y_pred = model.predict(X_test_scaled)
            mse = mean_squared_error(y_test, y_pred)
            rmse = np.sqrt(mse)
            r2 = r2_score(y_test, y_pred)
            
            # Сохранение модели и скейлера
            model_name = f"model_{shop_type}_{year_suffix}_{bbox_suffix}"
            model_path = os.path.join(self.models_dir, f"{model_name}.joblib")
            scaler_path = os.path.join(self.models_dir, f"scaler_{model_name}.joblib")
            
            joblib.dump(model, model_path)
            joblib.dump(scaler, scaler_path)
            
            # Сохранение метрик
            self.metrics[model_name] = {
                'mse': float(mse),
                'rmse': float(rmse),
                'r2': float(r2)
            }
            
            logger.info(f"Model {model_name} trained successfully")
            logger.info(f"Metrics: MSE={mse:.4f}, RMSE={rmse:.4f}, R²={r2:.4f}")
            
        except Exception as e:
            logger.error(f"Error training model for {shop_type}: {str(e)}")
            raise

    def train_all_models(self) -> None:
        """Обучение моделей для всех датасетов"""
        try:
            # Создаем директорию для моделей, если её нет
            os.makedirs(self.models_dir, exist_ok=True)
            
            # Ищем все датасеты
            datasets_dir = "datasets"
            dataset_files = glob.glob(os.path.join(datasets_dir, "dataset_*.json"))
            
            if not dataset_files:
                logger.warning("No dataset files found")
                return
            
            for dataset_path in dataset_files:
                try:
                    # Загружаем датасет и получаем метаданные
                    df, shop_type, year_suffix, bbox_suffix = self.load_dataset(dataset_path)
                    
                    # Обучаем модель
                    self.train_model(df, shop_type, year_suffix, bbox_suffix)
                    
                except Exception as e:
                    logger.error(f"Error processing dataset {dataset_path}: {str(e)}")
                    continue
            
            # Сохраняем метрики
            metrics_path = os.path.join(self.models_dir, "training_metrics.json")
            with open(metrics_path, 'w') as f:
                json.dump(self.metrics, f, indent=2)
            
            logger.info("All models trained successfully")
            
        except Exception as e:
            logger.error(f"Error in train_all_models: {str(e)}")
            raise

if __name__ == "__main__":
    trainer = ModelTrainer()
    trainer.train_all_models()