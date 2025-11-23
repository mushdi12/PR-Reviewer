package db

import (
	"log/slog"

	"github.com/jmoiron/sqlx"
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB

	Team *TeamRepository
	User *UserRepository
	PR   *PRRepository
}

func New(log *slog.Logger, address string) (*DB, error) {
	conn, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, err
	}

	db := &DB{
		log:  log,
		conn: conn,
	}

	db.Team = NewTeamRepository(db)
	db.User = NewUserRepository(db)
	db.PR = NewPRRepository(db)

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

// для тестов
func (db *DB) Conn() *sqlx.DB {
	return db.conn
}
