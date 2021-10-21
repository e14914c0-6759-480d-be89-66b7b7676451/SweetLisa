package db

import (
	"fmt"
	"github.com/boltdb/bolt"
	"log"
	"path"
)

var (
	ErrKeyNotFound = fmt.Errorf("key not found")

	db   *bolt.DB
)

func InitDB(confDir string) {
	var err error
	db, err = bolt.Open(path.Join(confDir, "bolt.db"), 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func DB() *bolt.DB {
	return db
}
