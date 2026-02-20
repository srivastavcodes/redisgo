package main

import "time"

// RDbStats tracks redis's persistence activity.
type RDbStats struct {
	lastSaveTs int64
	saves      int
}

// AofStats tracks AOF's persistence activity.
type AofStats struct {
	rewrites int
}

// GeneralStats tracks server-wide command and connection activity.
type GeneralStats struct {
	totalConnections int
	expiredKeys      int
	evictedKeys      int
	totalCommands    int
}

// RedisGo is the single shared state for the server. One instance exists per
// running server and is passed to every handler. Fields are not individually
// synchronized — callers are responsible for holding db.mu where needed.
type RedisGo struct {
	redisDb *RedisDb
	conf    *Config
	// aof  *Aof

	// monitors []*Client
	startedAt     time.Time
	clientCount   int
	peakMem       uint64
	inCompaction  bool // true if the server is currently running Aof compaction.
	inRdbSnapshot bool // true if the server is currently snapshotting Rdb.
	dbCopy        map[string]*Item

	rbdState RDbStats
	aofStats AofStats
	genStats GeneralStats
}

// NewRedisGo initializes a new RedisGo server from conf. If Aof is enabled,
// the Aof file is opened and EverySec fsync goroutine is started if configured.
func NewRedisGo(conf *Config) *RedisGo {
	server := &RedisGo{
		redisDb:   NewRdb(),
		conf:      conf,
		startedAt: time.Now(),
	}
	if conf.aofEnabled {
		// todo: create a new aof, and sync EverySec in a goroutine.
	}
	return server
}
