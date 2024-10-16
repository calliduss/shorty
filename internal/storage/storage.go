package storage

import "errors"

var (
	ErrURLNotFound      = errors.New("url not found")
	ErrURLAlreadyExists = errors.New("url already exists")
)
