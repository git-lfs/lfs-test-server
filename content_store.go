package main

import (
	"io"
	"os"
	"path/filepath"
)

// ContentStore provides a simple file system based storage.
type ContentStore struct {
	basePath string
}

// NewContentStore creates a ContentStore at the base directory.
func NewContentStore(base string) (*ContentStore, error) {
	if err := os.MkdirAll(base, 0750); err != nil {
		return nil, err
	}

	return &ContentStore{base}, nil
}

// Get takes a Meta object and retreives the content from the store, returning
// it as an io.Reader.
func (s *ContentStore) Get(meta *Meta) (io.Reader, error) {
	path := filepath.Join(s.basePath, transformKey(meta.Oid))

	return os.Open(path)
}

// Put takes a Meta object and an io.Reader and writes the content to the store.
func (s *ContentStore) Put(meta *Meta, r io.Reader) error {
	path := filepath.Join(s.basePath, transformKey(meta.Oid))
	tmpPath := path + ".tmp"

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0640)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath)

	if _, err := io.Copy(file, r); err != nil {
		return err
	}
	file.Close()

	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	return nil
}

func transformKey(key string) string {
	if len(key) < 5 {
		return key
	}

	return filepath.Join(key[0:2], key[2:4], key[4:len(key)])
}
