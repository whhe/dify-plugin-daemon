package db

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// MySQLInitializer initializes DBInitializer for MySQL.
type MySQLInitializer struct {
	host     string
	port     int
	user     string
	password string
	sslMode  string
}

// NewMySQLInitializer creates a new MySQLInitializer.
func NewMySQLInitializer(host string, port int, user string, password string, sslMode string) *MySQLInitializer {
	return &MySQLInitializer{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		sslMode:  sslMode,
	}
}

func (m *MySQLInitializer) Connect(dbName string) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&tls=%v", m.user, m.password, m.host, m.port, dbName, m.sslMode == "require")
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}
func (m *MySQLInitializer) CreateDatabaseIfNotExists(db *gorm.DB, dbName string) error {
	pool, err := db.DB()
	if err != nil {
		return err
	}
	defer pool.Close()

	rows, err := pool.Query(fmt.Sprintf("SELECT 1 FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = '%s'", dbName))
	if err != nil {
		return err
	}

	if !rows.Next() {
		// create database
		_, err = pool.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	}
	return err
}

func (m *MySQLInitializer) Setup(db *gorm.DB) error {
	return nil
}
