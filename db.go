package main

import (
	"sync"
)

type Database struct {
	rwm    sync.RWMutex
	memory int64
	store  map[string]*Item
}

func NewDb() *Database {
	return &Database{
		store: make(map[string]*Item),
	}
}
