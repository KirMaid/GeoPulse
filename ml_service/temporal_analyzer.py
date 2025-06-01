import pandas as pd
import joblib
from sklearn.linear_model import LinearRegression
from typing import List, Dict

class TemporalAnalyzer:
    def __init__(self):
        # Use LinearRegression as a default model (replace with actual model if needed)
        self.trend_model = joblib.load('models/trend_model.pkl') if joblib else LinearRegression()

    def analyze_trend(self, historical_data: List[Dict], years: int) -> Dict:
        """Analyze trends over the specified period."""
        if not historical_data:
            return {'trend_slope': 0.0, 'trend_stability': 0.0}

        df = pd.DataFrame(historical_data)
        df['period'] = pd.to_datetime(df['period'], format='%Y-Q%m')

        # Prepare features: years since start
        df['years_since_start'] = (df['period'].dt.year - df['period'].dt.year.min()) + \
                                 (df['period'].dt.month - 1) / 12.0

        # Fit model to predict total_objects based on time
        X = df[['years_since_start']]
        y = df['total_objects']
        self.trend_model.fit(X, y)

        # Compute trend slope and stability (R^2 score)
        trend_slope = float(self.trend_model.coef_[0])
        trend_stability = float(self.trend_model.score(X, y))

        return {
            'trend_slope': trend_slope,
            'trend_stability': trend_stability
        }