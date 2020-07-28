package dedup

import (
	"bytes"
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
	distance s1.ChordAngle
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
		distance: s1.ChordAngleFromAngle(s1.Angle(rad)),
		interval: interval,
		level:    s2.MinEdgeMetric.ClosestLevel(rad),
	}
	return &f, nil
}

// Distance returns distance tolerance in meters.
func (f SpatioTemporalFilter) Distance() float64 {
	return float64(f.distance.Angle() * earthRadiusMeters)
}

// Level returns filter cell level.
func (f SpatioTemporalFilter) Level() int {
	return f.level
}

func (f SpatioTemporalFilter) Grid(ll s2.LatLng) s2.CellUnion {

	/*
		+---+---+---+ Cell 0 is where the current event LatLng belongs to.
		| 1 | 2 | 3 | Cell edge length is approximately equals to the distance
		+---+---+---+ tolerance. If event's LatLng is close to the cell edge,
		| 4 | 0 | 5 | then earlier event's coordinates can be in the cell 0 or
		+---+---+---+ one of the neighbour cells. Hence, all 9 cells must be
		| 6 | 7 | 8 | checked for earlier events. CellID is used as a key prefix.
		+---+---+---+
	*/

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
			cellID := decodeKey(append([]byte(nil), iter.Item().Key()...))
			ll := cellID.LatLng()
			if err := fn(ll.Lat.Degrees(), ll.Lng.Degrees()); err != nil {
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
		pt := s2.PointFromLatLng(ll)
		for _, id := range f.Grid(ll) {
			if isUnique = !f.match(txn, id, pt); !isUnique {
				return nil // found match
			}
		}

		// second pass, is storing given event in the database index, if no
		// earlier events found. Entry is created with TTL to satisfy temporal
		// requirement.
		entry := badger.NewEntry(encodeKey(s2.CellIDFromLatLng(ll)), nil)
		return txn.SetEntry(entry.WithTTL(f.interval))
	})
	return
}

// match iterates over records with the prefix from cellID and compares distance
// between given s2.LatLng and coordinates on the index key. If distance is
// within distance (argument) is returns true.
func (f SpatioTemporalFilter) match(txn *badger.Txn, cellID s2.CellID, pt s2.Point) bool {
	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	iter := txn.NewIterator(opts)
	defer iter.Close()

	minRange := encodeKey(cellID.RangeMin())
	maxRange := encodeKey(cellID.RangeMax())

	for iter.Seek(minRange); iter.Valid() && bytes.Compare(minRange, maxRange) <= 0; iter.Next() {
		cellID := decodeKey(iter.Item().Key())
		if s2.CompareDistance(pt, cellID.Point(), f.distance) <= 0 {
			return true
		}
	}
	return false
}

const s2CellIDLen = 8

// encodeKey takes latitude and longitude and encodes them into a key, which is
// used in the database index.
// Key format is:
// - 1 byte, key type;
// - 8 bytes, s2.CellID, always indexed at the maximum level;
func encodeKey(id s2.CellID) []byte {
	buf := make([]byte, keyLen+s2CellIDLen)
	buf[0] = SpatioTemporalKey
	binary.BigEndian.PutUint64(buf[keyLen:], uint64(id))
	return buf
}

// decodeKey decodes given slice of bytes (database index key) into s2.LatLng.
func decodeKey(p []byte) s2.CellID {
	id := binary.BigEndian.Uint64(p[keyLen:])
	return s2.CellID(id)
}
