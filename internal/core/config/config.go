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

	// Browser origins allowed to call the API, comma-separated. The frontend is
	// same-origin in development and under docker-compose; a deploy that serves
	// it from another host has to be named here.
	CORSOrigins string

	// PgBouncer (connection pooling)
	PgBouncerURL string

	// Email (SendGrid)
	SendGridAPIKey string
	EmailFrom      string
	EmailFromName  string
	AppURL         string // public URL the welcome email links to

	// Patient messaging (no provider integrated yet — see pkg/sms, pkg/whatsapp)
	SMSAPIKey      string
	SMSFrom        string
	WhatsAppAPIKey string
	WhatsAppFrom   string

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
		CORSOrigins:         getenv("CORS_ORIGINS", "http://localhost:5173,http://localhost:5174"),
		PgBouncerURL:        pgBouncerURL,
		SendGridAPIKey:      getenv("SENDGRID_API_KEY", ""),
		EmailFrom:           getenv("EMAIL_FROM", "no-reply@hms.local"),
		EmailFromName:       getenv("EMAIL_FROM_NAME", "HMS Platform"),
		AppURL:              getenv("APP_URL", "http://localhost:5173"),
		SMSAPIKey:           getenv("SMS_API_KEY", ""),
		SMSFrom:             getenv("SMS_FROM", "HMS"),
		WhatsAppAPIKey:      getenv("WHATSAPP_API_KEY", ""),
		WhatsAppFrom:        getenv("WHATSAPP_FROM", ""),
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
