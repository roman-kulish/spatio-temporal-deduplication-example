package dedup

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/golang/geo/s1"
	"github.com/golang/geo/s2"
)

const earthRadiusMeters = 6371010.0

// SpatioTemporalFilter implements spatio-temporal deduplication filter.
type SpatioTemporalFilter struct {
	db       *badger.DB
	distance s1.Angle
	interval time.Duration
	level    int
}

// NewSpatioTemporalFilter creates and returns an instance of the deduplication Filter.
func NewSpatioTemporalFilter(db *badger.DB, distance float64, interval time.Duration) (Filter, error) {
	switch {
	case distance <= 0:
		return nil, errors.New("filter: distance tolerance between events must be greater than zero")
	case interval <= 0:
		return nil, errors.New("filter: time tolerance between events must be greater than zero")
	}
	rad := distance / earthRadiusMeters
	f := SpatioTemporalFilter{
		db:       db,
		distance: s1.Angle(rad),
		interval: interval,
		level:    s2.AvgEdgeMetric.MinLevel(rad),
	}
	return &f, nil
}

// Distance returns distance tolerance as s1.Angle (radians).
func (f SpatioTemporalFilter) Distance() s1.Angle {
	return f.distance
}

// DistanceMeters returns distance tolerance in meters.
func (f SpatioTemporalFilter) DistanceMeters() float64 {
	return float64(f.distance * earthRadiusMeters)
}

// Interval returns time tolerance.
func (f SpatioTemporalFilter) Interval() time.Duration {
	return f.interval
}

// Level returns filter cell level.
func (f SpatioTemporalFilter) Level() int {
	return f.level
}

func (f SpatioTemporalFilter) Grid(ll s2.LatLng) s2.CellUnion {
	cellID := s2.CellIDFromLatLng(ll).Parent(f.level)
	cells := make([]s2.CellID, 9)
	cells[0] = cellID
	for i, cellID := range cellID.AllNeighbors(f.level) {
		cells[i+1] = cellID
	}
	return cells
}

// IndexedLocations iterates over indexed locations and calls fn with
// latitude and longitude.
func (f SpatioTemporalFilter) IndexedLocations(fn func(lat, lng float64) error) error {
	return f.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte{SpatioTemporalKey}
		opts.PrefetchValues = false
		iter := txn.NewIterator(opts)
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			lat, lng := decodeKey(append([]byte(nil), iter.Item().Key()...))
			if err := fn(lat, lng); err != nil {
				return err
			}
		}
		return nil
	})
}

func (f SpatioTemporalFilter) Filter(ev Event) (isUnique bool, err error) {
	err = f.db.Update(func(txn *badger.Txn) error {
		ll := s2.LatLngFromDegrees(ev.Lat, ev.Lng)
		if !ll.IsValid() {
			return fmt.Errorf("filter: invalid coordinates [%v, %v]", ev.Lat, ev.Lng)
		}

		// first pass, is the scan for any earlier events.
		prefixes := makePrefixes(f.Grid(ll))
		for i := range prefixes {
			if isUnique = !match(txn, prefixes[i], f.distance, ll); !isUnique {
				return nil // found match
			}
		}

		// second pass, is storing given event in the database index, if no
		// earlier events found. Entry is created with TTL to satisfy temporal
		// requirement.
		entry := badger.NewEntry(encodeLatLng(ll), nil)
		return txn.SetEntry(entry.WithTTL(f.interval))
	})
	return
}

const (
	s2CellIDLen = 16
	coordsLen   = 8
)

// encodeLatLng takes latitude and longitude and encodes them into a key, which is
// used in the database index.
// Key format is:
// - 1 byte, key type;
// - 8 bytes, s2.CellID, always indexed at the maximum level;
// - 4 bytes, latitude, in MicroDegrees format (E7);
// - 4 bytes, longitude, in MicroDegrees format (E7).
func encodeLatLng(ll s2.LatLng) []byte {
	cellID := s2.CellIDFromLatLng(ll)
	buf := make([]byte, keyLen+s2CellIDLen+(coordsLen*2))

	buf[0] = SpatioTemporalKey
	copy(buf[keyLen:], cellID.ToToken())
	binary.BigEndian.PutUint32(buf[keyLen+s2CellIDLen:], uint32(ll.Lat.Degrees()*1e7))
	binary.BigEndian.PutUint32(buf[keyLen+s2CellIDLen+coordsLen:], uint32(ll.Lng.Degrees()*1e7))
	return buf
}

// encodeCellID takes s2.CellID and encodes it into a key, which is then can
// be used for prefix search.
// Key format is:
// - 1 byte, key type;
// - N bytes, s2.CellID, trailing zeros stripped
func encodeCellID(cellID s2.CellID) []byte {
	return append([]byte{SpatioTemporalKey}, cellID.ToToken()...)
}

// decodeKey decodes given slice of bytes (database index key) into s2.LatLng.
func decodeKey(p []byte) (lat, lng float64) {
	latE6 := binary.BigEndian.Uint32(p[keyLen+s2CellIDLen:])
	lngE6 := binary.BigEndian.Uint32(p[keyLen+s2CellIDLen+coordsLen:])
	return float64(int32(latE6)) / 1e7, float64(int32(lngE6)) / 1e7
}

// makePrefixes generates a list of prefixes to scan for earlier events, using
// given coordinates and S2 Cell level.
func makePrefixes(c []s2.CellID) [][]byte {
	/*
		+---+---+---+ Cell 0 is where the current event LatLng belongs to.
		| 1 | 2 | 3 | Cell edge length is approximately equals to the distance
		+---+---+---+ tolerance. If event's LatLng is close to the cell edge,
		| 4 | 0 | 5 | then earlier event's coordinates can be in the cell 0 or
		+---+---+---+ one of the neighbour cells. Hence, all 9 cells must be
		| 6 | 7 | 8 | checked for earlier events. CellID is used as a key prefix.
		+---+---+---+
	*/
	cells := make([][]byte, len(c))
	for i := range c {
		cells[i] = encodeCellID(c[i])
	}
	return cells
}

// match iterates over records with the given prefix and compares distance
// between given s2.LatLng and coordinates on the index key. If distance is
// within distance (argument) is returns true.
func match(txn *badger.Txn, prefix []byte, distance s1.Angle, loc s2.LatLng) bool {
	opts := badger.DefaultIteratorOptions
	opts.Prefix = prefix
	opts.PrefetchValues = false
	iter := txn.NewIterator(opts)
	defer iter.Close()

	for iter.Rewind(); iter.Valid(); iter.Next() {
		lat, lng := decodeKey(append([]byte(nil), iter.Item().Key()...))
		ll := s2.LatLngFromDegrees(lat, lng)
		if ll.Distance(loc) <= distance {
			return true
		}
	}
	return false
}
