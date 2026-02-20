package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// FSyncMode controls how often the AOF file is flushed to disk.
// Mirrors Redis's appendfsync config option.
type FSyncMode string

const (
	// Always flushes the AOF file every time a write command is executed.
	// Safest but slowest - guarantees only one command lost on db crash.
	Always FSyncMode = "always"

	// EverySec flushes the AOF buffer every second. Good balance between
	// safety and performance - at most only a second of commands may be
	// lost on db crash.
	EverySec FSyncMode = "everysec"

	// NoFSync leaves flushing entirely to the OS. Fastest but least safe
	// — OS decides when to flush, potentially losing several seconds of
	// writes on db crash.
	NoFSync FSyncMode = "nofsync"
)

// Eviction controls which keys are evicted when maxmemory is reached.
// Mirrors Redis's maxmemory-policy config option.'
type Eviction string

const (
	// NoEviction returns an error on write commands when maxmemory is reached.
	// No keys are evicted.
	NoEviction Eviction = "noeviction"

	// AllKeysRandom evicts a random key from the entire keyspace.
	AllKeysRandom Eviction = "allkeys-random"

	// AllKeysLRU evicts the least recently used key from the entire keyspace.
	AllKeysLRU Eviction = "allkeys-lru"

	// AllKeysLFU evicts the least frequently used key from the entire
	// keyspace.
	AllKeysLFU Eviction = "allkeys-lfu"

	// VolatileRandom evicts a random key that has an expiry set.
	VolatileRandom Eviction = "volatile-random"

	// VolatileLRU evicts the least recently used key that has an expiry set.
	VolatileLRU Eviction = "volatile-lru"

	// VolatileTTL evicts the key with the shortest TTL among keys with an
	// expiry set.
	VolatileTTL Eviction = "volatile-ttl"

	// VolatileLFU evicts the least frequently used key that has an expiry set.
	VolatileLFU Eviction = "volatile-lfu"
)

// RDbSnapshot defines a condition under which an RDB snapshot is triggered.
// A snapshot is taken when at least KeysChanged keys have been modified
// within the last Secs seconds.
type RDbSnapshot struct {
	Secs        int
	KeysChanged int
}

// Config holds all server configuration parsed from the config file. Zero values
// are safe defaults — a Config with no persistence, no auth, no memory limit,
// and no eviction.
type Config struct {
	// configFP is the path to the config file, used for INFO output.
	configFP string

	// dir is the working directory for RDB and AOF files.
	dir string

	// rdb holds the list of RDB snapshot policies. Empty means RDB persistence is
	// disabled.
	rdb []RDbSnapshot

	// rdbFn is the filename for the RDB snapshot file.
	rdbFn string

	// aofEnabled controls whether AOF persistence is active.
	aofEnabled bool

	// aofFn is the filename for the AOF file.
	aofFn string

	// aofFsync controls the AOF flush frequency.
	aofFsync FSyncMode

	// requirepass controls whether AUTH is required before commands are accepted.
	requirepass bool

	// password is the required AUTH password. Only meaningful if requirepass is
	// true.
	password string

	// maxmem is the maximum memory in bytes before eviction is triggered.
	// 0 means no limit.
	maxmem uint64

	// eviction is the eviction policy applied when maxmem is reached.
	eviction Eviction

	// memSamples is the number of keys sampled during eviction candidate selection.
	// Higher values give more accurate eviction at the cost of CPU. Defaults to 5
	// if not set, matching Redis's default.
	memSamples int
}

// readConfig parses the Redis compatible config file at fpath and returns the
// resulting Config. If the file cannot be opened, a default Config is returned,
// and a message is printed - this allows the server to run without a provided
// config.
func readConfig(fpath string) *Config {
	conf := &Config{memSamples: 5}

	cf, err := os.Open(fpath)
	if err != nil {
		log.Printf("cannot read config file at: %q - using defaults", fpath)
		return conf
	}
	defer func() { _ = cf.Close() }()

	scanner := bufio.NewScanner(cf)
	for scanner.Scan() {
		parseLines(scanner.Text(), conf)
	}
	if err := scanner.Err(); err != nil {
		log.Printf("error scanning config file: %s", err)
		return conf
	}
	if strings.TrimSpace(conf.dir) != "" {
		if err = os.MkdirAll(conf.dir, 0o755); err != nil {
			log.Printf("cannot create dir: %q: %v", conf.dir, err)
		}
	}
	return conf
}

// parseLines parses a single line of a config file into the provided Config.
// Lines starting with # are ignored and empty lines are ignored.
// Malformed lines are logged and skipped.
func parseLines(line string, conf *Config) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return
	}
	args := strings.Fields(line)
	if len(args) == 0 {
		return
	}
	cmd := strings.ToLower(args[0])
	switch cmd {
	case "save":
		if len(args) < 3 {
			log.Printf("save requires 2 args, got %d: %s", len(args), line)
			return
		}
		secs, err := strconv.Atoi(args[1])
		if err != nil {
			log.Printf("invalid save secs: %q, err=%s", args[1], err)
			return
		}
		keysChanged, err := strconv.Atoi(args[2])
		if err != nil {
			log.Printf("invalid save keysChanged: %q, err=%s", args[2], err)
			return
		}
		rdb := RDbSnapshot{Secs: secs, KeysChanged: keysChanged}
		conf.rdb = append(conf.rdb, rdb)
	case "dbfilename":
		if len(args) < 2 {
			log.Println("dbfilename requires a value")
			return
		}
		conf.rdbFn = args[1]
	case "appendfilename":
		if len(args) < 2 {
			log.Println("appendfilename requires a value")
			return
		}
		conf.aofFn = args[1]
	case "appendfsync":
		if len(args) < 2 {
			log.Println("appendfsync requires a value")
			return
		}
		conf.aofFsync = FSyncMode(args[1])
	case "dir":
		if len(args) < 2 {
			log.Println("dir requires a value")
			return
		}
		conf.dir = args[1]
	case "appendonly":
		if len(args) < 2 {
			log.Println("appendonly requires a value")
			return
		}
		conf.aofEnabled = strings.ToLower(args[1]) == "yes"
	case "requirepass":
		if len(args) < 2 {
			log.Println("requirepass requires a value")
			return
		}
		conf.requirepass = true
		conf.password = args[1]
	case "maxmemory":
		if len(args) < 2 {
			log.Println("maxmemory requires a value")
			return
		}
		maxmem, err := parseMem(args[1])
		if err != nil {
			log.Printf("cannot parse maxmemory %q, defaulting to 0: %v", args[1], err)
			return
		}
		conf.maxmem = maxmem
	case "maxmemory-policy":
		if len(args) < 2 {
			log.Println("maxmemory-policy requires a value")
			return
		}
		conf.eviction = Eviction(args[1])
	case "maxmemory-samples":
		if len(args) < 2 {
			log.Println("maxmemory-samples requires a value")
			return
		}
		n, err := strconv.Atoi(args[1])
		if err != nil {
			log.Printf("cannot parse maxmemory-samples %q, defaulting to 5: %v", args[1], err)
			return
		}
		conf.memSamples = n
	default:
		log.Printf("unknown directive %q", cmd)
	}
}

// parseMem parses a memory string with optional unit suffix into bytes.
// Supports kb, mb, gb suffixes (case-insensitive); lack of suffix means
// bytes. Ex: 100mb, 1gb, 32kb, 1024.
func parseMem(str string) (uint64, error) {
	str = strings.TrimSpace(strings.ToLower(str))

	var multiplier uint64 = 1
	switch {
	case strings.HasSuffix(str, "kb"):
		multiplier = 1024
		str = strings.TrimSuffix(str, "kb")
	case strings.HasSuffix(str, "mb"):
		multiplier = 1024 * 1024
		str = strings.TrimSuffix(str, "mb")
	case strings.HasSuffix(str, "gb"):
		multiplier = 1024 * 1024 * 1024
		str = strings.TrimSuffix(str, "gb")
	}
	mem, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid memory value %q: %w", str, err)
	}
	return mem * multiplier, nil
}
