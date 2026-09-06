package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL        string
	JWTSecret          string
	Port               string
	SuperAdminEmail    string
	SuperAdminPassword string

	// Redis (caching)
	RedisAddr string

	// PgBouncer (connection pooling)
	PgBouncerURL string

	// Email (SendGrid)
	SendGridAPIKey string
	EmailFrom      string
	EmailFromName  string
	AppURL         string // public URL the welcome email links to

	// Image hosting (Cloudinary)
	CloudinaryCloudName string
	CloudinaryAPIKey    string
	CloudinaryAPISecret string
}

func Load() Config {
	_ = godotenv.Load()
	pgBouncerURL := getenv("PGBOUNCER_URL", "")
	return Config{
		DatabaseURL:         getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/postgres?sslmode=disable"),
		JWTSecret:           getenv("JWT_SECRET", "dev-secret-change-me"),
		Port:                getenv("PORT", "8080"),
		SuperAdminEmail:     getenv("SUPER_ADMIN_EMAIL", "super@hms.local"),
		SuperAdminPassword:  getenv("SUPER_ADMIN_PASSWORD", "SuperAdmin123!"),
		RedisAddr:           getenv("REDIS_ADDR", "localhost:6379"),
		PgBouncerURL:        pgBouncerURL,
		SendGridAPIKey:      getenv("SENDGRID_API_KEY", ""),
		EmailFrom:           getenv("EMAIL_FROM", "no-reply@hms.local"),
		EmailFromName:       getenv("EMAIL_FROM_NAME", "HMS Platform"),
		AppURL:              getenv("APP_URL", "http://localhost:5173"),
		CloudinaryCloudName: getenv("CLOUDINARY_CLOUD_NAME", ""),
		CloudinaryAPIKey:    getenv("CLOUDINARY_API_KEY", ""),
		CloudinaryAPISecret: getenv("CLOUDINARY_API_SECRET", ""),
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
