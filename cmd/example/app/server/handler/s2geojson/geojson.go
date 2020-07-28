package s2geojson

import (
	"encoding/json"

	"github.com/golang/geo/s2"
)

const (
	TypePoint             Type = "Point"
	TypePolygon           Type = "Polygon"
	TypeFeature           Type = "Feature"
	TypeFeatureCollection Type = "FeatureCollection"
)

type Type string

type geometryObject struct {
	Type        Type        `json:"type"`
	Coordinates interface{} `json:"coordinates"`
}

// Point represents GeoJSON Point.
type Point [2]float64

func (p Point) MarshalJSON() ([]byte, error) {
	return json.Marshal(geometryObject{
		Type:        TypePoint,
		Coordinates: [2]float64(p),
	})
}

// NewPoint returns GeoJSON Point instance from the provided latitude and
// longitude.
func NewPoint(lat, lng float64) Point {
	return [2]float64{lng, lat}
}

// Polygon represents GeoJSON Polygon.
type Polygon struct {
	*s2.Loop
}

func (p Polygon) MarshalJSON() ([]byte, error) {
	v := p.Vertices()
	coords := make([][]float64, 0, len(v)+1)
	for _, pt := range v {
		ll := s2.LatLngFromPoint(pt)
		coords = append(coords, []float64{ll.Lng.Degrees(), ll.Lat.Degrees()})
	}
	coords = append(coords, coords[0])
	return json.Marshal(geometryObject{
		Type:        TypePolygon,
		Coordinates: [][][]float64{coords},
	})
}

// Feature represents GeoJSON feature.
type Feature struct {
	Type       Type                   `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	ID         string                 `json:"id,omitempty"`
	Geometry   interface{}            `json:"geometry,omitempty"`
}

// NewFeature returns GeoJSON Feature instance from the provided geometry.
func NewFeature(geometry interface{}) Feature {
	return Feature{
		Type:       TypeFeature,
		Properties: make(map[string]interface{}),
		Geometry:   geometry,
	}
}

// FeatureCollection represents a collection of GeoJSON features.
type FeatureCollection struct {
	Type     Type      `json:"type"`
	Features []Feature `json:"features,omitempty"`
}

// NewFeatureCollection returns GeoJSON FeatureCollection instance.
func NewFeatureCollection() *FeatureCollection {
	return &FeatureCollection{
		Type:     TypeFeatureCollection,
		Features: make([]Feature, 0),
	}
}

// Push adds Feature to the collection.
func (fc *FeatureCollection) Push(f Feature) *FeatureCollection {
	fc.Features = append(fc.Features, f)
	return fc
}
