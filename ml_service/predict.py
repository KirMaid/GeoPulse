import os
import json
import numpy as np
import joblib
import logging
import glob
from typing import Dict, List, Optional
from flask import Flask, request, jsonify

# Настройка логирования
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

app = Flask(__name__)

class PredictionService:
    def __init__(self):
        # Переходим в корневую директорию проекта
        root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), '..'))
        os.chdir(root_dir)
        
        # Путь к директории с моделями
        self.models_dir = "models"
        
    def find_best_model(self, shop_type: str, bbox: str) -> str:
        """Находит наиболее подходящую модель для заданных параметров"""
        try:
            # Парсим bbox запроса
            req_min_lat, req_min_lon, req_max_lat, req_max_lon = map(float, bbox.split(','))
            
            # Ищем все модели для данного типа магазина
            all_models = glob.glob(os.path.join(self.models_dir, f"model_{shop_type}_*.joblib"))
            if not all_models:
                raise ValueError(f"No models found for shop type: {shop_type}")
            
            # Проверяем каждую модель
            for model_path in all_models:
                model_name = os.path.basename(model_path).replace('.joblib', '')
                # Извлекаем координаты из имени модели
                # Формат: model_shop_type_YYYYMMDD_to_YYYYMMDD_lat_lon_lat_lon
                parts = model_name.split('_')
                if len(parts) >= 8:
                    model_bbox = f"{parts[-4]}_{parts[-3]}_{parts[-2]}_{parts[-1]}"
                    model_min_lat, model_min_lon, model_max_lat, model_max_lon = map(float, model_bbox.split('_'))
                    
                    # Проверяем, находится ли запрошенная область внутри области модели
                    if (model_min_lat <= req_min_lat and model_max_lat >= req_max_lat and
                        model_min_lon <= req_min_lon and model_max_lon >= req_max_lon):
                        logger.info(f"Found matching model: {model_name}")
                        return model_name
            
            # Если точного совпадения нет, берем самую последнюю модель
            logger.warning("No exact area match found, using the latest model")
            return os.path.basename(all_models[-1]).replace('.joblib', '')
            
        except Exception as e:
            logger.error(f"Error finding best model: {str(e)}")
            raise
    
    def predict(self, features: Dict[str, float], shop_type: str, bbox: str) -> Dict[str, float]:
        """Получение предсказания"""
        try:
            # Находим подходящую модель
            model_name = self.find_best_model(shop_type, bbox)
            logger.info(f"Using model: {model_name}")
            
            # Загружаем модель и скейлер
            model_path = os.path.join(self.models_dir, f"{model_name}.joblib")
            scaler_path = os.path.join(self.models_dir, f"scaler_{model_name}.joblib")
            
            if not os.path.exists(model_path) or not os.path.exists(scaler_path):
                raise ValueError(f"Model or scaler not found: {model_name}")
            
            model = joblib.load(model_path)
            scaler = joblib.load(scaler_path)
            
            # Подготавливаем признаки
            feature_values = [
                features['avg_area'],
                features['avg_dist_to_primary'],
                features['avg_dist_to_subway'],
                features['closure_rate'],
                features['new_object_rate'],
                features['object_density'],
                features['total_objects']
            ]
            
            # Нормализуем признаки
            X = scaler.transform([feature_values])
            
            # Получаем предсказание
            activity_level = float(model.predict(X)[0])
            
            # Рассчитываем тренд на основе признаков
            trend = (
                features['new_object_rate'] * 0.6 -
                features['closure_rate'] * 0.4
            )
            
            # Рассчитываем предсказанное количество новых и закрытых объектов
            total_objects = features['total_objects']
            predicted_new = int(total_objects * features['new_object_rate'])
            predicted_closed = int(total_objects * features['closure_rate'])
            
            return {
                "activity_level": activity_level,
                "trend": trend,
                "predicted_new": predicted_new,
                "predicted_closed": predicted_closed,
                "total_objects": total_objects,
                "model_used": model_name
            }
            
        except Exception as e:
            logger.error(f"Error making prediction: {str(e)}")
            raise
    
    def get_available_models(self) -> List[Dict[str, str]]:
        """Получение списка доступных моделей"""
        models = []
        metrics_file = os.path.join(self.models_dir, "training_metrics.json")
        
        if os.path.exists(metrics_file):
            with open(metrics_file, 'r') as f:
                metrics = json.load(f)
            
            for model_name in metrics:
                # Извлекаем информацию из имени модели
                # Формат: model_shop_type_YYYY_YYYY_lat_lon_lat_lon
                parts = model_name.split('_')
                if len(parts) >= 4:
                    shop_type = parts[1]
                    train_year = f"{parts[2]}_{parts[3]}"
                    bbox = f"{parts[4]}_{parts[5]}_{parts[6]}_{parts[7]}"
                    
                    models.append({
                        "name": model_name,
                        "shop_type": shop_type,
                        "train_year": train_year,
                        "bbox": bbox,
                        "metrics": metrics[model_name]
                    })
        
        return models

# Инициализация сервиса предсказаний
prediction_service = PredictionService()

@app.route('/predict', methods=['POST'])
def predict_endpoint():
    try:
        data = request.get_json()
        
        # Валидация входных данных
        required_fields = ['bbox', 'shop_type', 'prediction_year', 'train_period']
        for field in required_fields:
            if field not in data:
                return jsonify({'error': f'Missing required field: {field}'}), 400
        
        # Получаем признаки из датасета
        datasets_dir = "datasets"
        
        # Преобразуем формат периода из YYYY-YYYY в YYYYMMDD_to_YYYYMMDD
        start_year, end_year = data['train_period'].split('-')
        dataset_pattern = f"dataset_{data['shop_type']}_{start_year}0101_to_{end_year}1231.json"
        dataset_files = glob.glob(os.path.join(datasets_dir, dataset_pattern))
        
        if not dataset_files:
            return jsonify({'error': f'Dataset for shop type {data["shop_type"]} and period {data["train_period"]} not found'}), 404
        
        dataset_path = dataset_files[0]
        with open(dataset_path, 'r') as f:
            dataset = json.load(f)
        
        # Находим кластер по координатам
        target_bbox = data['bbox']
        cluster_features = None
        
        for cluster in dataset['clusters']:
            if cluster['bbox'] == target_bbox:
                # Берем последний год из данных кластера
                last_year_data = cluster['data'][-1]['data'][0]
                cluster_features = {
                    'avg_area': last_year_data['avg_area'],
                    'avg_dist_to_primary': last_year_data['avg_dist_to_primary'],
                    'avg_dist_to_subway': last_year_data['avg_dist_to_subway'],
                    'closure_rate': last_year_data['closure_rate'],
                    'new_object_rate': last_year_data['new_object_rate'],
                    'object_density': last_year_data['object_density'],
                    'total_objects': last_year_data['total_objects']
                }
                break
        
        if cluster_features is None:
            return jsonify({'error': f'Cluster with bbox {target_bbox} not found in dataset'}), 404
        
        # Получаем предсказание
        result = prediction_service.predict(cluster_features, data['shop_type'], data['bbox'])
        
        # Добавляем информацию о запросе
        result.update({
            'bbox': data['bbox'],
            'shop_type': data['shop_type'],
            'prediction_year': data['prediction_year'],
            'train_period': data['train_period']
        })
        
        return jsonify(result), 200
        
    except ValueError as e:
        return jsonify({'error': str(e)}), 400
    except Exception as e:
        logger.error(f"Prediction error: {str(e)}")
        return jsonify({'error': 'Internal server error'}), 500

@app.route('/health', methods=['GET'])
def health_check():
    return jsonify({'status': 'healthy'}), 200

@app.route('/models', methods=['GET'])
def get_models():
    try:
        models = prediction_service.get_available_models()
        return jsonify(models), 200
    except Exception as e:
        logger.error(f"Error getting available models: {str(e)}")
        return jsonify({'error': 'Internal server error'}), 500

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000) 