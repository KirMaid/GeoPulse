import pandas as pd
from sqlalchemy import create_engine
import os


def update_activity_levels():
    # Конфигурация БД
    DB_URL = os.getenv("DB_URL", "postgresql://user:pass@localhost/osm")
    engine = create_engine(DB_URL)

    # Загрузка внешних данных (пример)
    external_data = pd.read_csv('external_activity.csv')

    # Обновление записей в БД
    for _, row in external_data.iterrows():
        update_query = f"""
            UPDATE training_data
            SET activity_level = {row['activity_level']}
            WHERE 
                shop_type = '{row['shop_type']}' AND
                bbox = '{row['bbox']}' AND
                recorded_at::date = '{row['date']}'
        """
        try:
            engine.execute(update_query)
            print(f"Updated record for {row['shop_type']} in {row['bbox']}")
        except Exception as e:
            print(f"Update failed: {str(e)}")


if __name__ == '__main__':
    update_activity_levels()