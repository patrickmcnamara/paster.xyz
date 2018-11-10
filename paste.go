package main

import (
	"crypto/rand"
	"encoding/base64"
	"time"
)

type id []byte

func (ID id) String() string {
	return base64.RawURLEncoding.EncodeToString(ID)
}

func generateID() id {
	ID := make([]byte, 8)
	rand.Read(ID)
	return ID
}

type paste struct {
	ID    id
	Value string
	Time  time.Time
	User  id
}
