package main

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// RedisDb represents a Redis database, an in-memory key-value store; instance
// must not be copied after first use because sync.Mutex must not be copied.
// Methods on RedisDb are thread-safe for now.
type RedisDb struct {
	store   map[string]*Item
	rwm     sync.RWMutex
	memUsed atomic.Uint64 // memUsed is approximate memory usage of the database in bytes across
}

// NewRedisDb returns an initialized empty database.
func NewRedisDb() *RedisDb {
	return &RedisDb{
		store: make(map[string]*Item),
	}
}

// MemUsed returns the approximate memory usage of the database in bytes.
// It is an atomic operation, hence can be read from multiple goroutines.
func (rdb *RedisDb) MemUsed() uint64 {
	return rdb.memUsed.Load()
}

// Set stores a key-value pair in the database. If the key already existed in
// the database, its memory usage is subtracted before overwriting. Set is
// thread-safe.
func (rdb *RedisDb) Set(key, val string) {
	rdb.rwm.Lock()
	defer rdb.rwm.Unlock()

	if old, ok := rdb.store[key]; ok {
		prev := rdb.memUsed.Load()
		curr := prev - old.approxMemUsage(key)
		rdb.memUsed.Store(curr)
	}
	item := &Item{Value: val}
	imem := item.approxMemUsage(key)

	rdb.memUsed.Add(imem)
	rdb.store[key] = item
	log.Printf("set key=%q, memory usage=%d bytes", key, rdb.memUsed.Load())
}

// Get returns the (val, true) for the given key, or (nil, false) if the key
// does not exist. Get updates LastAccessed and AccessCount on the Item for
// LRU/LFU tracking. Get is thread-safe.
func (rdb *RedisDb) Get(key string) (*Item, bool) {
	rdb.rwm.Lock()
	defer rdb.rwm.Unlock()

	item, ok := rdb.store[key]
	if !ok {
		return nil, false
	}
	item.AccessCount++
	item.LastUsedAt = time.Now()
	log.Printf(
		"key=%q accessed %d times, last used at=%v",
		key, item.AccessCount, item.LastUsedAt,
	)
	return item, true
}

// Delete removes the key from the underlying store and updates the memory
// usage, returns early if key not exists. Delete is thread-safe.
func (rdb *RedisDb) Delete(key string) {
	rdb.rwm.Lock()
	defer rdb.rwm.Unlock()

	item, ok := rdb.store[key]
	if !ok {
		return
	}
	prev := rdb.memUsed.Load()
	curr := prev - item.approxMemUsage(key)
	rdb.memUsed.Store(curr)
	delete(rdb.store, key)
	log.Printf(
		"deleted key=%q, memory usage=%d bytes",
		key, rdb.memUsed.Load(),
	)
}
