from flask import Flask, request, jsonify
import joblib
import numpy as np
import pandas as pd
from sklearn.ensemble import RandomForestRegressor, GradientBoostingRegressor
from sklearn.linear_model import LinearRegression
from sklearn.svm import SVR
from sklearn.model_selection import train_test_split, GridSearchCV
from sklearn.preprocessing import StandardScaler
from sklearn.cluster import DBSCAN
from typing import Dict, List
import os
import threading
from sqlalchemy import create_engine

app = Flask(__name__)

MODEL_DIR = "models"
os.makedirs(MODEL_DIR, exist_ok=True)
SHOP_TYPES = ['supermarket', 'restaurant', 'clothing']
DB_URL = os.getenv("DB_URL", "postgresql://user:pass@localhost/osm")

models: Dict[str, any] = {}
scalers: Dict[str, any] = {}
engine = create_engine(DB_URL)


class TemporalAnalyzer:
    def analyze_trend(self, historical_data: List[Dict], years: int) -> Dict:
        """Анализ временных трендов"""
        if not historical_data:
            return {'trend_slope': 0.0, 'trend_stability': 0.0}

        df = pd.DataFrame(historical_data)

        # Преобразование периода в числовой формат (год + квартал)
        df['period_numeric'] = df['period'].apply(
            lambda p: float(p.split('-')[0]) + (float(p.split('-')[1][1:]) - 1) / 4.0
        )

        # Подготовка данных
        X = df[['period_numeric']]
        y = df['total_objects']

        # Обучение простой линейной модели
        if len(df) < 2:
            return {'trend_slope': 0.0, 'trend_stability': 0.0}

        try:
            model = LinearRegression()
            model.fit(X, y)
            r2 = model.score(X, y)
            return {
                'trend_slope': float(model.coef_[0]),
                'trend_stability': float(r2)
            }
        except Exception:
            return {'trend_slope': 0.0, 'trend_stability': 0.0}


class PredictionEngine:
    def __init__(self):
        self.temporal_analyzer = TemporalAnalyzer()
        self.load_models()

    def load_models(self):
        """Загрузка предобученных моделей"""
        for shop_type in SHOP_TYPES:
            model_path = os.path.join(MODEL_DIR, f'{shop_type}_model.pkl')
            scaler_path = os.path.join(MODEL_DIR, f'{shop_type}_scaler.pkl')

            if os.path.exists(model_path) and os.path.exists(scaler_path):
                try:
                    models[shop_type] = joblib.load(model_path)
                    scalers[shop_type] = joblib.load(scaler_path)
                    print(f"Loaded model for {shop_type}")
                except Exception as e:
                    print(f"Error loading model for {shop_type}: {str(e)}")
                    self.retrain_model(shop_type)
            else:
                print(f"No model found for {shop_type}, training new model")
                self.retrain_model(shop_type)

    def retrain_model(self, shop_type: str):
        """Переобучение модели для конкретного типа магазина"""
        print(f"Training model for {shop_type}")

        try:
            # Загрузка данных из БД
            query = f"""
                SELECT 
                    total_objects, avg_area, subway_stations,
                    avg_dist_to_subway, avg_dist_to_primary,
                    object_density, new_object_rate, trend_slope,
                    activity_level
                FROM training_data
                WHERE shop_type = '{shop_type}' AND activity_level > 0
            """
            df = pd.read_sql_query(query, engine)

            if df.empty:
                print(f"No training data for {shop_type}")
                return

            # Подготовка данных
            X = df.drop('activity_level', axis=1)
            y = df['activity_level']

            # Разделение данных
            X_train, X_test, y_train, y_test = train_test_split(
                X, y, test_size=0.2, random_state=42
            )

            # Масштабирование признаков
            scaler = StandardScaler()
            X_train_scaled = scaler.fit_transform(X_train)
            X_test_scaled = scaler.transform(X_test)

            # Кандидаты моделей
            model_candidates = [
                (RandomForestRegressor(random_state=42),
                 {'n_estimators': [100, 200], 'max_depth': [None, 10, 20]}
                 ),
                (GradientBoostingRegressor(random_state=42),
                 {'n_estimators': [100, 200], 'learning_rate': [0.01, 0.1]}
                 ),
                (SVR(), {'C': [0.1, 1, 10], 'kernel': ['linear', 'rbf']})
            ]

            best_score = -float('inf')
            best_model = None

            # Поиск лучшей модели
            for model, params in model_candidates:
                grid = GridSearchCV(
                    estimator=model,
                    param_grid=params,
                    cv=5,
                    scoring='r2',
                    n_jobs=-1
                )
                grid.fit(X_train_scaled, y_train)

                # Оценка на тестовых данных
                score = grid.score(X_test_scaled, y_test)
                if score > best_score:
                    best_score = score
                    best_model = grid.best_estimator_
                    best_scaler = scaler

            # Сохранение лучшей модели
            if best_model:
                models[shop_type] = best_model
                scalers[shop_type] = best_scaler
                joblib.dump(best_model, os.path.join(MODEL_DIR, f'{shop_type}_model.pkl'))
                joblib.dump(scaler, os.path.join(MODEL_DIR, f'{shop_type}_scaler.pkl'))
                print(f"Model for {shop_type} retrained. R²: {best_score:.4f}")

        except Exception as e:
            print(f"Error training model for {shop_type}: {str(e)}")

    def predict(self, request_data: Dict) -> Dict:
        """Основной метод предсказания"""
        features = request_data['features']
        shop_type = request_data['shop_type']
        historical_data = request_data.get('historical_data', [])
        years = request_data.get('years', 5)
        elements = features.get('elements', [])

        # Проверка наличия модели
        if shop_type not in models or shop_type not in scalers:
            self.retrain_model(shop_type)
            if shop_type not in models:
                raise ValueError(f"No model available for {shop_type}")

        # Подготовка входных данных
        input_data = self.prepare_input(features)
        input_scaled = scalers[shop_type].transform([input_data])

        # Предсказание активности
        activity_level = float(models[shop_type].predict(input_scaled)[0])

        # Анализ тренда
        trend_result = self.temporal_analyzer.analyze_trend(historical_data, years)

        # Генерация хотспотов
        hotspots = self.generate_hotspots(elements)

        return {
            'activity_level': activity_level,
            'trend_slope': trend_result['trend_slope'],
            'trend_strength': trend_result['trend_stability'],
            'hotspots': hotspots
        }

    def prepare_input(self, features: Dict) -> List:
        """Подготовка входных данных для модели"""
        spatial = features.get('spatial', {})
        temporal = features.get('temporal', {})
        return [
            spatial.get('total_objects', 0),
            spatial.get('avg_area', 0.0),
            spatial.get('subway_stations', 0),
            spatial.get('avg_dist_to_subway', 0.0),
            spatial.get('avg_dist_to_primary', 0.0),
            temporal.get('object_density', 0.0),
            temporal.get('new_object_rate', 0.0),
            temporal.get('trend_slope', 0.0)
        ]

    def generate_hotspots(self, elements: List[Dict]) -> List[Dict]:
        """Генерация хотспотов через кластеризацию DBSCAN"""
        if not elements:
            return []

        # Извлечение координат
        coords = []
        for element in elements:
            if 'lat' in element and 'lon' in element:
                coords.append([element['lat'], element['lon']])

        if len(coords) < 3:
            return []

        # Кластеризация DBSCAN
        coords_array = np.array(coords)
        dbscan = DBSCAN(eps=0.01, min_samples=3, metric='euclidean')
        labels = dbscan.fit_predict(coords_array)

        # Сбор результатов кластеризации
        hotspots = []
        unique_labels = set(labels)

        for label in unique_labels:
            if label == -1:  # Пропуск шума
                continue

            cluster_mask = (labels == label)
            cluster_points = coords_array[cluster_mask]

            # Расчет центра кластера
            centroid = np.mean(cluster_points, axis=0)

            # Расчет score (нормированный размер кластера)
            cluster_size = len(cluster_points)
            score = min(1.0, cluster_size / 20.0)  # Макс. 20 точек = score 1.0

            hotspots.append({
                'lat': float(centroid[0]),
                'lon': float(centroid[1]),
                'score': float(score)
            })

        # Сортировка по score
        hotspots.sort(key=lambda x: x['score'], reverse=True)
        return hotspots[:10]  # Возвращаем топ-10 хотспотов


# Инициализация движка
prediction_engine = PredictionEngine()


@app.route('/predict', methods=['POST'])
def predict_endpoint():
    try:
        data = request.get_json()
        result = prediction_engine.predict(data)
        return jsonify(result), 200
    except Exception as e:
        return jsonify({'error': str(e)}), 400


@app.route('/retrain', methods=['POST'])
def retrain_endpoint():
    def background_retrain():
        for shop_type in SHOP_TYPES:
            prediction_engine.retrain_model(shop_type)

    # Запуск в фоновом потоке
    threading.Thread(target=background_retrain).start()
    return jsonify({'status': 'retraining started'}), 202


if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000, debug=True)