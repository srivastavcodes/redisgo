package main

import (
	"time"
)

// unixTSEpoch is used as a sentinel value to indicate that an item has no expiry set.
// We use the Unix timestamp epoch rather than Go's zero time (year 0001) because
// time.Time{} is not the same as time.Unix(0,0). This keeps expiry checks consistent
// when serializing/deserializing to Unix timestamps.
const unixTSEpoch int64 = -62135596800

// Item represents a value stored in the database along with its metadata.
// All fields are exported to support gob encoding for RDB persistence.
type Item struct {
	// Value is the string value stored for this key.
	Value string

	// Expiration is the expiry time for this item. If Exp.Unix() == unixTSEpoch,
	// the item has no expiry set and will never expire passively.
	Expiration time.Time

	// LastAccess records the last time this item was read.
	// Used by the LRU eviction policy to determine least recently used keys.
	LastAccess time.Time

	// AccessCount counts how many times this item has been read.
	// Used by the LFU eviction policy to determine least frequently used keys.
	AccessCount int
}

// hasExpired reports whether this item has an expiry set and that expiry has
// passed. An item with no expiry set (Exp.Unix() == unixTSEpoch) never expires.
func (i *Item) hasExpired() bool {
	return i.Expiration.Unix() != unixTSEpoch && time.Until(i.Expiration) <= 0
}

// approxMemUsage returns an approximate memory usage in bytes for this item,
// including its key. These estimates are based on
// Go runtime internals (could change in future go versions):
//   - string header (pointer + length): 16 bytes
//   - time.Time: 24 bytes
//   - map entry overhead: ~32 bytes
//
// Used by the eviction policy to track total memory usage. Precision is not
// required since eviction decisions tolerate some inaccuracy.
func (i *Item) approxMemUsage(key string) int64 {
	var total int64
	const (
		stringHeader = 16
		timeSize     = 24
		mapEntry     = 32
	)
	total += timeSize + mapEntry
	total += int64(stringHeader + len(key))
	total += int64(stringHeader + len(i.Value))
	return total
}
