package config

import "time"

type storageConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	HealthCheck     time.Duration
}

var Storage = &storageConfig{
	Host:            "localhost",
	Port:            5432,
	User:            "main",
	Password:        "",
	Database:        "perpspread",
	SSLMode:         "disable",
	MaxConns:        10,
	MinConns:        2,
	MaxConnLifetime: time.Minute * 30,
	MaxConnIdleTime: time.Minute * 5,
	HealthCheck:     time.Minute,
}
