package db

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PostgresInitializer initializes DBInitializer for Postgresql.
type PostgresInitializer struct {
	host     string
	port     int
	user     string
	password string
	sslMode  string
}

// NewPostgresInitializer creates a new PostgresInitializer.
func NewPostgresInitializer(host string, port int, user string, password string, sslMode string) *PostgresInitializer {
	return &PostgresInitializer{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		sslMode:  sslMode,
	}
}

func (p *PostgresInitializer) Connect(dbName string) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=%s dbname=%s ", p.host, p.port, p.user, p.password, p.sslMode, dbName)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

func (p *PostgresInitializer) CreateDatabaseIfNotExists(db *gorm.DB, dbName string) error {
	pool, err := db.DB()
	if err != nil {
		return err
	}
	defer pool.Close()

	rows, err := pool.Query(fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname = '%s'", dbName))
	if err != nil {
		return err
	}

	if !rows.Next() {
		// create database
		_, err = pool.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	}
	return err
}

func (p *PostgresInitializer) Setup(db *gorm.DB) error {
	pool, err := db.DB()
	if err != nil {
		return err
	}

	// check if uuid-ossp extension exists
	rows, err := pool.Query("SELECT 1 FROM pg_extension WHERE extname = 'uuid-ossp'")
	if err != nil {
		return err
	}

	if !rows.Next() {
		// create the uuid-ossp extension
		_, err = pool.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")
		if err != nil {
			return err
		}
	}

	pool.SetConnMaxIdleTime(time.Minute * 1)
	return nil
}
