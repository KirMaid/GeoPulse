package model

type OSMElement struct {
	ID            int64             `json:"id"`
	Type          string            `json:"type"`
	Lat           float64           `json:"lat"`
	Lon           float64           `json:"lon"`
	Tags          map[string]string `json:"tags"`
	Bounds        Bounds            `json:"bounds"`
	Area          float64           `json:"area"`            // Площадь объекта в квадратных метрах
	DistToPrimary float64           `json:"dist_to_primary"` // Расстояние до главной дороги в метрах
	DistToSubway  float64           `json:"dist_to_subway"`  // Расстояние до метро в метрах
}

type Tags struct {
	Name            string `json:"name"`
	Shop            string `json:"shop"`
	Building        string `json:"building"`
	Highway         string `json:"highway"`
	PublicTransport string `json:"public_transport"`
}
