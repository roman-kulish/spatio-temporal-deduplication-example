package handler

import (
	"github.com/golang/geo/s2"

	"github.com/roman-kulish/spatio-temporal-deduplication-example/cmd/example/app/server/handler/s2geojson"
)

func makePoint(lat, lng float64, props map[string]interface{}) s2geojson.Feature {
	pt := s2geojson.NewPoint(lat, lng)
	ft := s2geojson.NewFeature(pt)
	if len(props) > 0 {
		for k := range props {
			ft.Properties[k] = props[k]
		}
	}
	return ft
}

func makeGrid(cu s2.CellUnion) *s2geojson.FeatureCollection {
	fc := s2geojson.NewFeatureCollection()
	for _, cell := range cu {
		c := s2.CellFromCellID(cell)
		pl := s2geojson.Polygon{Loop: s2.LoopFromCell(c)}
		ft := s2geojson.NewFeature(pl)
		fc.Push(ft)
	}
	return fc
}
