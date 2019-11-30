package main

import (
	"crypto/rand"
	"time"
)

type paste struct {
	ID    []byte
	Value string
	Time  time.Time
}

func generateID() (id []byte) {
	id = make([]byte, 8)
	rand.Read(id)
	return id
}
