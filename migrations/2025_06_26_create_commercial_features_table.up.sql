-- Создание таблицы для хранения исторических данных коммерческой активности
CREATE TABLE commercial_features (
    id SERIAL PRIMARY KEY,
    period VARCHAR(10) NOT NULL,          -- Формат: Год-Квартал (2024-Q1)
    shop_type VARCHAR(50) NOT NULL,       -- Тип магазина (supermarket, restaurant и т.д.)
    bbox GEOMETRY(Polygon, 4326) NOT NULL,-- Геометрия bounding box
    total_objects INTEGER NOT NULL,       -- Общее количество объектов
    new_objects INTEGER NOT NULL,         -- Количество новых объектов
    closed_objects INTEGER NOT NULL,      -- Количество закрытых объектов
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_commercial_features_period ON commercial_features (period);
CREATE INDEX idx_commercial_features_shop_type ON commercial_features (shop_type);
CREATE INDEX idx_commercial_features_bbox ON commercial_features USING GIST (bbox);

COMMENT ON TABLE commercial_features IS 'Исторические данные коммерческой активности по регионам и периодам';
COMMENT ON COLUMN commercial_features.period IS 'Временной период в формате Год-Квартал (2024-Q1)';
COMMENT ON COLUMN commercial_features.bbox IS 'Геометрическая область bounding box в SRID 4326';
COMMENT ON COLUMN commercial_features.total_objects IS 'Общее количество коммерческих объектов в периоде';
COMMENT ON COLUMN commercial_features.new_objects IS 'Количество новых объектов за период';
COMMENT ON COLUMN commercial_features.closed_objects IS 'Количество закрытых объектов за период';