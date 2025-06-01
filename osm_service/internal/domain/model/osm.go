package model

type OSMElement struct {
	ID     int64
	Type   string
	Lat    float64
	Lon    float64
	Tags   map[string]string
	Bounds Bounds
}

type Bounds struct {
	MinLat float64
	MinLon float64
	MaxLat float64
	MaxLon float64
}

type PredictionRequest struct {
	BBox     string
	Years    int
	ShopType string
}

type PredictionResult struct {
	ActivityLevel float64
	TrendSlope    float64
	TrendStrength float64
	Hotspots      []Hotspot
}

type Hotspot struct {
	Lat   float64
	Lon   float64
	Score float64
}

type HistoricalData struct {
	Period        string
	TotalObjects  int
	NewObjects    int
	ClosedObjects int
}

type TemporalFeatures struct {
	YearsAnalyzed int
	ObjectDensity float64
	NewObjectRate float64
	TrendSlope    float64
}

type SpatialFeatures struct {
	TotalObjects     int
	AvgArea          float64
	SubwayStations   int
	AvgDistToSubway  float64
	AvgDistToPrimary float64
}

type FeatureSet struct {
	Spatial  SpatialFeatures
	Temporal TemporalFeatures
}
