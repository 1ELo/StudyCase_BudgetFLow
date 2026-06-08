package main

import (
	"log"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	dsn := buildDSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	_ = db // akan digunakan saat repo tersedia

	port := getEnv("APP_PORT", "8080")
	log.Printf("Server would start on port %s (router not yet wired)", port)
	// TODO: wire router saat handler tersedia
}

// SKELETON: Cukup return string kosong dulu agar tidak error
func buildDSN() string {
	// TODO: Implementasi DSN builder
	return ""
}

// SKELETON: Cukup return string kosong atau fallback dulu
func getEnv(key, fallback string) string {
	// TODO: Implementasi pembacaan env
	return fallback
}
