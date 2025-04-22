package store

import (
	"database/sql"
)

type ChatStore struct {
	db *sql.DB
}
