package storage

import "errors"

var (
	ErrURLNotFound = errors.New("url not found")
	ErrUrlExists   = errors.New("url exists")
)
