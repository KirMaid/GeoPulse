package repository

import (
	"context"
	"fmt"
	"github.com/serjvanilla/go-overpass"
	"net/http"
	"osm_service/internal/domain/model"
	"time"
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

	result, err := r.executeQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute commercial data query: %w", err)
	}

	return convertToOSMElements(result), nil
}

func (r *OverpassRepository) GetSubwayData(ctx context.Context) ([]model.OSMElement, error) {
	query := fmt.Sprintf(`
		[out:json];
		(
			node["railway"="station"]["station"="subway"];
			way["railway"="station"]["station"="subway"];
		);
		out body;
		>;
		out skel qt;
	`)

	result, err := r.executeQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute subway data query: %w", err)
	}

	return convertToOSMElements(result), nil
}

func (r *OverpassRepository) GetRoadData(ctx context.Context) ([]model.OSMElement, error) {
	query := fmt.Sprintf(`
		[out:json];
		(
			way["highway"~"primary|secondary|trunk"];
		);
		out body;
		>;
		out skel qt;
	`)

	result, err := r.executeQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute road data query: %w", err)
	}

	return convertToOSMElements(result), nil
}

func (r *OverpassRepository) executeQuery(ctx context.Context, query string) (*overpass.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	result, err := r.client.Query(query)
	if err != nil {
		return nil, fmt.Errorf("overpass query failed: %w", err)
	}

	return &result, nil
}

func convertToOSMElements(result *overpass.Result) []model.OSMElement {
	var elements []model.OSMElement

	for _, node := range result.Nodes {
		elements = append(elements, model.OSMElement{
			ID:     node.ID,
			Type:   string(overpass.ElementTypeNode),
			Lat:    node.Lat,
			Lon:    node.Lon,
			Tags:   node.Tags,
			Bounds: model.Bounds{}, // Nodes don't have bounds in your model
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
