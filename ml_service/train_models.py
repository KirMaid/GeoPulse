import time
import schedule
from prediction_engine import PredictionEngine

def job():
    print("Starting scheduled model retraining...")
    engine = PredictionEngine()
    for shop_type in ['supermarket', 'restaurant', 'clothing']:
        engine.retrain_model(shop_type)
    print("Model retraining completed!")

# Ежедневное переобучение в 2:00
schedule.every().day.at("02:00").do(job)

if __name__ == '__main__':
    while True:
        schedule.run_pending()
        time.sleep(60)