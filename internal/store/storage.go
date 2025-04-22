package store

import (
	"database/sql"
)

type Storage struct {
	Chat interface {
	}
}

func NewStorage(db *sql.DB) Storage {
	return Storage{
		Chat: &ChatStore{db},
	}
}
