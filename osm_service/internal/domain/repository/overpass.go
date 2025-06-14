package repository

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"osm_service/internal/domain/model"
	"time"

	"github.com/serjvanilla/go-overpass"
)

type OverpassRepository struct {
	client  *overpass.Client
	timeout time.Duration
}

func NewOverpassRepository(endpoint string, timeout time.Duration) *OverpassRepository {
	httpClient := &http.Client{
		Timeout: timeout,
	}
	client := overpass.NewWithSettings(endpoint, 2, httpClient)
	return &OverpassRepository{
		client:  &client,
		timeout: timeout,
	}
}

func (r *OverpassRepository) GetCommercialData(ctx context.Context, bbox string, shopType string) ([]model.OSMElement, error) {
	query := fmt.Sprintf(`
		[out:json];
		(
			node["shop"="%s"](%s);
			way["shop"="%s"](%s);
			node["amenity"~"%s"](%s);
			way["amenity"~"%s"](%s);
		);
		out body;
		>;
		out skel qt;
	`,
		shopType, bbox,
		shopType, bbox,
		getAmenityFilter(shopType), bbox,
		getAmenityFilter(shopType), bbox)

	log.Printf("Executing commercial data query for bbox=%s, shop_type=%s:\n%s", bbox, shopType, query)
	result, err := r.executeQuery(ctx, query)
	if err != nil {
		log.Printf("Failed to execute commercial data query: %v", err)
		return nil, fmt.Errorf("failed to execute commercial data query: %w", err)
	}

	elements := convertToOSMElements(result)
	log.Printf("Retrieved %d commercial elements", len(elements))
	return elements, nil
}

func (r *OverpassRepository) GetSubwayData(ctx context.Context, bbox string, date string) ([]model.OSMElement, error) {
	query := fmt.Sprintf(`
		[out:json][date:"%s"];
		(
			node["railway"="station"]["station"="subway"](%s);
			way["railway"="station"]["station"="subway"](%s);
		);
		out body;
		>;
		out skel qt;
	`, date, bbox, bbox)

	log.Printf("Executing subway data query for bbox=%s, date=%s:\n%s", bbox, date, query)
	result, err := r.executeQuery(ctx, query)
	if err != nil {
		log.Printf("Failed to execute subway data query: %v", err)
		return nil, fmt.Errorf("failed to execute subway data query: %w", err)
	}

	elements := convertToOSMElements(result)
	log.Printf("Retrieved %d subway elements", len(elements))
	return elements, nil
}

func (r *OverpassRepository) GetRoadData(ctx context.Context, bbox string, date string) ([]model.OSMElement, error) {
	query := fmt.Sprintf(`
		[out:json][date:"%s"];
		(
			way["highway"~"primary|secondary|trunk"](%s);
		);
		out body;
		>;
		out skel qt;
	`, date, bbox)

	log.Printf("Executing road data query for bbox=%s, date=%s:\n%s", bbox, date, query)
	result, err := r.executeQuery(ctx, query)
	if err != nil {
		log.Printf("Failed to execute road data query: %v", err)
		return nil, fmt.Errorf("failed to execute road data query: %w", err)
	}

	elements := convertToOSMElements(result)
	log.Printf("Retrieved %d road elements", len(elements))
	return elements, nil
}

func (r *OverpassRepository) executeQuery(ctx context.Context, query string) (*overpass.Result, error) {
	startTime := time.Now()
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	log.Printf("Starting Overpass API query execution...")
	result, err := r.client.Query(query)
	if err != nil {
		log.Printf("Overpass API query failed after %v: %v", time.Since(startTime), err)
		return nil, fmt.Errorf("overpass query failed: %w", err)
	}

	log.Printf("Overpass API query completed successfully in %v", time.Since(startTime))
	return &result, nil
}

func convertToOSMElements(result *overpass.Result) []model.OSMElement {
	var elements []model.OSMElement

	// Convert nodes
	for _, node := range result.Nodes {
		// Для узлов создаем небольшую область вокруг точки (например, 50 метров)
		const nodeRadius = 0.00045 // примерно 50 метров в градусах
		elements = append(elements, model.OSMElement{
			ID:   node.ID,
			Type: string(overpass.ElementTypeNode),
			Lat:  node.Lat,
			Lon:  node.Lon,
			Tags: node.Tags,
			Bounds: model.Bounds{
				MinLat: node.Lat - nodeRadius,
				MinLon: node.Lon - nodeRadius,
				MaxLat: node.Lat + nodeRadius,
				MaxLon: node.Lon + nodeRadius,
			},
		})
	}

	// Convert ways
	for _, way := range result.Ways {
		var lat, lon float64
		count := len(way.Nodes)
		if count > 0 {
			for _, node := range way.Nodes {
				lat += node.Lat
				lon += node.Lon
			}
			lat /= float64(count)
			lon /= float64(count)
		}

		var bounds model.Bounds
		if way.Bounds != nil {
			bounds = model.Bounds{
				MinLat: way.Bounds.Min.Lat,
				MinLon: way.Bounds.Min.Lon,
				MaxLat: way.Bounds.Max.Lat,
				MaxLon: way.Bounds.Max.Lon,
			}
		}

		elements = append(elements, model.OSMElement{
			ID:     way.ID,
			Type:   string(overpass.ElementTypeWay),
			Lat:    lat,
			Lon:    lon,
			Tags:   way.Tags,
			Bounds: bounds,
		})
	}

	// Note: Relations are not used in your model, so they are ignored
	return elements
}

func getAmenityFilter(shopType string) string {
	switch shopType {
	case "restaurant":
		return "restaurant|cafe|fast_food"
	case "supermarket":
		return "supermarket|convenience"
	case "clothing":
		return "clothing_store"
	default:
		return ".*"
	}
}

func (r *OverpassRepository) GetCommercialDataByDate(ctx context.Context, bbox, shopType, date string) ([]model.OSMElement, error) {
	query := fmt.Sprintf(`
        [out:json][date:"%s"];
        (
            node["shop"="%s"](%s);
            way["shop"="%s"](%s);
            node["amenity"~"%s"](%s);
            way["amenity"~"%s"](%s);
        );
        out body;
        >;
        out skel qt;
    `, date,
		shopType, bbox,
		shopType, bbox,
		getAmenityFilter(shopType), bbox,
		getAmenityFilter(shopType), bbox)

	log.Printf("Executing commercial data query for bbox=%s, shop_type=%s, date=%s:\n%s", bbox, shopType, date, query)
	result, err := r.executeQuery(ctx, query)
	if err != nil {
		log.Printf("Failed to execute commercial data query for date %s: %v", date, err)
		return nil, fmt.Errorf("failed to execute commercial data query for date %s: %w", date, err)
	}

	elements := convertToOSMElements(result)
	log.Printf("Retrieved %d commercial elements for date %s", len(elements), date)
	return elements, nil
}
