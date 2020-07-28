package dedup

const (
	SpatioTemporalKey byte = 0x01

	keyLen = 1
)

// Filter interface is implemented by event deduplication filters.
type Filter interface {
	// Filter processes event and returns true, if it is unique.
	Filter(Event) (isUnique bool, err error)
}

// Event is a demo event type.
type Event struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}
