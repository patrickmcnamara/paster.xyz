package main

import (
	"crypto/rand"
	"database/sql/driver"
	"encoding/base64"
	"time"
)

type id []byte

type nullTime struct {
	Time  time.Time
	Valid bool
}

func (n *nullTime) Scan(value interface{}) error {
	n.Time, n.Valid = value.(time.Time)
	return nil
}

func (n nullTime) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Time, nil
}

func (ID id) String() string {
	return base64.RawURLEncoding.EncodeToString(ID)
}

func generateID() id {
	ID := make([]byte, 8)
	rand.Read(ID)
	return ID
}

type paste struct {
	ID     id
	Value  string
	Time   time.Time
	Expiry nullTime
	User   id
	List   bool
}
