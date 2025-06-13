-- Создание таблицы для хранения данных обучения ML-моделей
CREATE TABLE training_data (
    id SERIAL PRIMARY KEY,
    shop_type VARCHAR(50) NOT NULL,       -- Тип магазина
    bbox TEXT NOT NULL,                   -- Bounding box в текстовом формате
    total_objects INTEGER NOT NULL,       -- Общее количество объектов
    avg_area FLOAT NOT NULL,              -- Средняя площадь объектов
    subway_stations INTEGER NOT NULL,     -- Количество станций метро
    avg_dist_to_subway FLOAT NOT NULL,    -- Среднее расстояние до метро (км)
    avg_dist_to_primary FLOAT NOT NULL,   -- Среднее расстояние до основных дорог (км)
    object_density FLOAT NOT NULL,        -- Плотность объектов (объектов/год)
    new_object_rate FLOAT NOT NULL,       -- Доля новых объектов
    trend_slope FLOAT NOT NULL,           -- Наклон тренда
    elements JSONB NOT NULL,              -- Данные элементов в формате JSON
    activity_level FLOAT NOT NULL,        -- Уровень коммерческой активности (целевая переменная)
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание индексов
CREATE INDEX idx_training_data_shop_type ON training_data (shop_type);
CREATE INDEX idx_training_data_activity_level ON training_data (activity_level);
CREATE INDEX idx_training_data_recorded_at ON training_data (recorded_at);

-- Комментарии к таблице и колонкам
COMMENT ON TABLE training_data IS 'Данные для обучения ML-моделей прогнозирования коммерческой активности';
COMMENT ON COLUMN training_data.bbox IS 'Bounding box в формате "minLat,minLon,maxLat,maxLon"';
COMMENT ON COLUMN training_data.avg_area IS 'Средняя площадь коммерческих объектов в км²';
COMMENT ON COLUMN training_data.avg_dist_to_subway IS 'Среднее расстояние до ближайшего метро в км';
COMMENT ON COLUMN training_data.avg_dist_to_primary IS 'Среднее расстояние до основных дорог в км';
COMMENT ON COLUMN training_data.object_density IS 'Плотность объектов (объектов/год)';
COMMENT ON COLUMN training_data.new_object_rate IS 'Доля новых объектов от общего количества';
COMMENT ON COLUMN training_data.trend_slope IS 'Наклон тренда изменения количества объектов';
COMMENT ON COLUMN training_data.elements IS 'Данные OSM-элементов в формате JSON';
COMMENT ON COLUMN training_data.activity_level IS 'Уровень коммерческой активности (целевая переменная для модели)';