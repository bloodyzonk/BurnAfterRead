package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"io"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Message struct {
	ID         string
	Ciphertext string
	Nonce      string
	Filename   string
	ExpiresAt  time.Time
}

func InitStore(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    ciphertext TEXT,
    nonce TEXT,
    expires_at DATETIME
)`)
	return db, err
}

func NewID() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *Server) dbInsertMessage(id, ciphertext, nonce string, expiresAt time.Time) error {
	_, err := s.db.Exec(`INSERT INTO messages (id, ciphertext, nonce, expires_at) VALUES (?, ?, ?, ?)`,
		id, ciphertext, nonce, expiresAt)
	return err
}

func (s *Server) dbGetMessage(id string) (*Message, error) {
	var m Message
	err := s.db.QueryRow(`SELECT ciphertext, nonce, expires_at FROM messages WHERE id = ?`, id).
		Scan(&m.Ciphertext, &m.Nonce, &m.ExpiresAt)
	if err != nil {
		return nil, err
	}
	m.ID = id
	return &m, nil
}

func (s *Server) dbDeleteMessage(id string) error {
	_, err := s.db.Exec(`DELETE FROM messages WHERE id = ?`, id)
	return err
}

func (s *Server) dbCleanupExpired() {
	_, _ = s.db.Exec(`DELETE FROM messages WHERE expires_at <= CURRENT_TIMESTAMP`)
}
