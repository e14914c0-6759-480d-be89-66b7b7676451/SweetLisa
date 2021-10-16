package db

import (
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/config"
	"log"
	"path"
	"sync"
)

var (
	ErrKeyNotFound = fmt.Errorf("key not found")

	once sync.Once
	db   *bolt.DB
)

func initDB() {
	confPath := config.GetConfig().Config
	var err error
	db, err = bolt.Open(path.Join(confPath, "bolt.db"), 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func DB() *bolt.DB {
	once.Do(initDB)
	return db
}
