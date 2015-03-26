package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"time"

	"github.com/boltdb/bolt"
)

type MetaStore struct {
	db *bolt.DB
}

var (
	errNoBucket       = errors.New("Bucket not found")
	errObjectNotFound = errors.New("Object not found")
)

var (
	usersBucket   = []byte("users")
	objectsBucket = []byte("objects")
)

func NewMetaStore(dbFile string) (*MetaStore, error) {
	db, err := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(usersBucket); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists(objectsBucket); err != nil {
			return err
		}

		return nil
	})

	return &MetaStore{db: db}, nil
}

func (s *MetaStore) Get(v *RequestVars) (*Meta, error) {
	if !s.authenticate(v.User, v.Password) {
		return nil, NewAuthError()
	}

	var meta Meta
	var value []byte

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(objectsBucket)
		if bucket == nil {
			return errNoBucket
		}

		value = bucket.Get([]byte(v.Oid))
		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(value) == 0 {
		return nil, errObjectNotFound
	}

	dec := gob.NewDecoder(bytes.NewBuffer(value))
	err = dec.Decode(&meta)
	if err != nil {
		return nil, err
	}

	return &meta, err
}

func (s *MetaStore) Put(v *RequestVars) (*Meta, error) {
	if !s.authenticate(v.User, v.Password) {
		return nil, NewAuthError()
	}

	// Check if it exists first
	if meta, err := s.Get(v); err == nil {
		meta.existing = true
		return meta, nil
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	meta := Meta{Oid: v.Oid, Size: v.Size}
	err := enc.Encode(meta)
	if err != nil {
		return nil, err
	}

	err = s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(objectsBucket)
		if bucket == nil {
			return errNoBucket
		}

		err = bucket.Put([]byte(v.Oid), buf.Bytes())
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &meta, nil
}

func (s *MetaStore) Shutdown() {
	s.db.Close()
}

func (s *MetaStore) AddUser(user, pass string) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(usersBucket)
		if bucket == nil {
			return errNoBucket
		}

		err := bucket.Put([]byte(user), []byte(pass))
		if err != nil {
			return err
		}
		return nil
	})

	return err
}

func (s *MetaStore) authenticate(user, password string) bool {
	value := ""

	s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(usersBucket)
		if bucket == nil {
			return errNoBucket
		}

		value = string(bucket.Get([]byte(user)))
		return nil
	})

	if value != "" && value == password {
		return true
	}
	return false
}

type authError struct {
	error
}

func (e authError) AuthError() bool {
	return true
}

func NewAuthError() authError {
	return authError{errors.New("Forbidden")}
}
