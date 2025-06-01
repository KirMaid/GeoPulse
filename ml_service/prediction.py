from typing import Dict, List
import joblib
from flask import Flask, request, jsonify
from temporal_analyzer import TemporalAnalyzer

app = Flask(__name__)

class PredictionEngine:
    def __init__(self):
        self.models = {
            'supermarket': self.load_model('supermarket'),
            'restaurant': self.load_model('restaurant'),
            'clothing': self.load_model('clothing'),  # Added clothing type
        }
        self.temporal_analyzer = TemporalAnalyzer()

    def predict(self, features: Dict, shop_type: str, historical_data: List[Dict] = None, years: int = 5) -> Dict:
        model = self.models.get(shop_type)
        if not model:
            raise ValueError(f"Unsupported shop type: {shop_type}")

        # Prepare input data for activity level prediction
        input_data = self.prepare_input(features)

        # Predict activity level
        prediction = model.predict(input_data)
        confidence = float(model.predict_proba(input_data).max())

        # Analyze trends if historical data is provided
        trend_slope = 0.0
        trend_strength = 0.0
        if historical_data:
            trend_result = self.temporal_analyzer.analyze_trend(historical_data, years)
            trend_slope = trend_result['trend_slope']
            trend_strength = trend_result['trend_stability']

        # Generate hotspots (placeholder: based on spatial features)
        hotspots = self.generate_hotspots(features.get('spatial', {}))

        return {
            'activity_level': float(prediction[0]),
            'trend_slope': trend_slope,
            'trend_strength': trend_strength,
            'hotspots': hotspots
        }

    def load_model(self, shop_type: str):
        path = f"models/{shop_type}_model.pkl"
        return joblib.load(path)

    def prepare_input(self, features: Dict) -> List:
        # Example: Convert features to a list for model input
        spatial = features.get('spatial', {})
        temporal = features.get('temporal', {})
        return [[
            spatial.get('total_objects', 0),
            spatial.get('avg_area', 0.0),
            spatial.get('subway_stations', 0),
            spatial.get('avg_dist_to_subway', 0.0),
            spatial.get('avg_dist_to_primary', 0.0),
            temporal.get('object_density', 0.0),
            temporal.get('new_object_rate', 0.0),
            temporal.get('trend_slope', 0.0)
        ]]

    def generate_hotspots(self, spatial_features: Dict) -> List[Dict]:
        # Placeholder: Generate dummy hotspots based on spatial features
        # In a real implementation, use clustering (e.g., DBSCAN) on spatial data
        return [
            {'lat': 55.75, 'lon': 37.62, 'score': 0.9},
            {'lat': 55.76, 'lon': 37.63, 'score': 0.8}
        ]

@app.route('/predict', methods=['POST'])
def predict():
    try:
        data = request.get_json()
        features = data['features']
        shop_type = data['shop_type']
        historical_data = data.get('historical_data', [])
        years = data.get('years', 5)

        engine = PredictionEngine()
        result = engine.predict(features, shop_type, historical_data, years)
        return jsonify(result)
    except Exception as e:
        return jsonify({'error': str(e)}), 400

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000)