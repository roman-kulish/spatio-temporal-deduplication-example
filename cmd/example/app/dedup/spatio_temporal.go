package dedup

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/golang/geo/s1"
	"github.com/golang/geo/s2"
)

const (
	earthRadiusMeters = 6371010.0
	locationsTTL      = 24 * time.Hour
)

// SpatioTemporalFilter implements spatio-temporal deduplication filter.
type SpatioTemporalFilter struct {
	db       *badger.DB
	distance s1.ChordAngle
	interval time.Duration
	level    int

	mu        sync.RWMutex
	watermark time.Time
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
func (f *SpatioTemporalFilter) Distance() float64 {
	return float64(f.distance.Angle() * earthRadiusMeters)
}

// Interval returns time tolerance.
func (f *SpatioTemporalFilter) Interval() time.Duration {
	return f.interval
}

// Level returns filter cell level.
func (f *SpatioTemporalFilter) Level() int {
	return f.level
}

// IndexedLocations iterates over indexed locations and calls fn with
// latitude and longitude.
func (f *SpatioTemporalFilter) IndexedLocations(fn func(lat, lng float64) error) error {
	return f.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte{SpatioTemporalKey}
		opts.PrefetchValues = false
		iter := txn.NewIterator(opts)
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			cellID, t := decodeKey(append([]byte(nil), iter.Item().Key()...))

			// check if location has expired.
			f.mu.RLock()
			if f.watermark.Add(-f.interval).After(t) {
				f.mu.RUnlock()
				continue
			}
			f.mu.RUnlock()

			ll := cellID.LatLng()
			if err := fn(ll.Lat.Degrees(), ll.Lng.Degrees()); err != nil {
				return err
			}
		}
		return nil
	})
}

func (f *SpatioTemporalFilter) Filter(ev Event) (isUnique bool, err error) {
	err = f.db.Update(func(txn *badger.Txn) error {
		ll := s2.LatLngFromDegrees(ev.Lat, ev.Lng)
		if !ll.IsValid() {
			return fmt.Errorf("filter: invalid coordinates [%v, %v]", ev.Lat, ev.Lng)
		}

		// watermark holds the time of the most recent event.
		f.mu.Lock()
		if ev.Time.After(f.watermark) {
			f.watermark = ev.Time
		}
		f.mu.Unlock()

		// first pass, is the scan for any earlier events.
		pt := s2.PointFromLatLng(ll)
		for _, id := range f.Cells(ll) {
			if hasMatch := f.match(txn, id, pt); hasMatch {
				return nil // found match
			}
		}
		isUnique = true

		// second pass, is storing given event in the database index, if no
		// earlier events found. Entry is created with TTL to satisfy temporal
		// requirement.
		key := encodeKey(s2.CellIDFromLatLng(ll), ev.Time)
		entry := badger.NewEntry(key, nil)
		return txn.SetEntry(entry.WithTTL(locationsTTL))
	})
	return
}

// Cells returns s2.CellUnion of cells to search for earlier indexed locations.
func (f *SpatioTemporalFilter) Cells(ll s2.LatLng) s2.CellUnion {
	// Cell 0 is where the current event LatLng belongs to. Cell edge length is
	// approximately equals to the distance tolerance. If event's LatLng is
	// close to the cell edge, then earlier event's coordinates can be in the
	// cell 0 or one of the neighbour Cells. Hence, all 9 Cells must be checked
	// for earlier events. CellID is used as a key prefix.
	//
	// +---+---+---+
	// | 1 | 2 | 3 |
	// +---+---+---+
	// | 4 | 0 | 5 |
	// +---+---+---+
	// | 6 | 7 | 8 |
	// +---+---+---+

	cellID := s2.CellIDFromLatLng(ll).Parent(f.level)
	cells := make([]s2.CellID, 9)
	cells[0] = cellID
	for i, cellID := range cellID.AllNeighbors(f.level) {
		cells[i+1] = cellID
	}
	return cells
}

// match iterates over records with the prefix from cellID and compares distance
// between given s2.LatLng and coordinates on the index key. If distance is
// within distance (argument) is returns true.
func (f *SpatioTemporalFilter) match(txn *badger.Txn, cellID s2.CellID, pt s2.Point) bool {
	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	opts.Prefix = []byte{SpatioTemporalKey}
	iter := txn.NewIterator(opts)
	defer iter.Close()

	minRange := encodePrefix(cellID.RangeMin())
	maxRange := encodePrefix(cellID.RangeMax())

	for iter.Seek(minRange); iter.Valid() && bytes.Compare(iter.Item().Key(), maxRange) <= 0; iter.Next() {
		key := iter.Item().Key()
		cellID, t := decodeKey(key)

		// check if location has expired.
		f.mu.RLock()
		if f.watermark.Add(-f.interval).After(t) {
			f.mu.RUnlock()
			_ = txn.Delete(key) // delete expired location.
			continue
		}
		f.mu.RUnlock()

		if s2.CompareDistance(pt, cellID.Point(), f.distance) <= 0 {
			return true
		}
	}
	return false
}

const (
	s2CellIDLen  = 8
	timestampLen = 8
)

// encodeKey takes latitude and longitude and encodes them into a key, which is
// used in the database index.
// Key format is:
// - 1 byte, key type;
// - 8 bytes, s2.CellID, always indexed at the maximum level;
// - 8 bytes, UNIX timestamp.
func encodeKey(id s2.CellID, t time.Time) []byte {
	buf := make([]byte, keyLen+s2CellIDLen+timestampLen)
	buf[0] = SpatioTemporalKey
	binary.BigEndian.PutUint64(buf[keyLen:], uint64(id))
	binary.BigEndian.PutUint64(buf[keyLen+s2CellIDLen:], uint64(t.Unix()))
	return buf
}

// encodeKey takes latitude and longitude and encodes them into a key, which is
// used in the database index.
// Key format is:
// - 1 byte, key type;
// - 8 bytes, s2.CellID, always indexed at the maximum level.
func encodePrefix(id s2.CellID) []byte {
	buf := make([]byte, keyLen+s2CellIDLen+timestampLen)
	buf[0] = SpatioTemporalKey
	binary.BigEndian.PutUint64(buf[keyLen:], uint64(id))
	return buf
}

// decodeKey decodes given slice of bytes (database index key) into s2.LatLng.
func decodeKey(p []byte) (s2.CellID, time.Time) {
	id := binary.BigEndian.Uint64(p[keyLen:])
	ts := binary.BigEndian.Uint64(p[keyLen+s2CellIDLen:])
	return s2.CellID(id), time.Unix(int64(ts), 0)
}
