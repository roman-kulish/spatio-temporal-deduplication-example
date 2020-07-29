package model

// Info contains distance and time tolerance information.
type Info struct {
	Distance string `json:"distance"`
	TTL      string `json:"ttl"`
}

// LatLng contains latitude and longitude pair.
type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// BBox is a bounding box.
type BBox struct {
	Hi LatLng `json:"hi"`
	Lo LatLng `json:"lo"`
}
