// Package config loads all runtime settings from the environment. Nothing in
// the application reads os.Getenv directly, so every deployment target is
// configured the same way and secrets never need to be committed.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	App struct {
		Name        string
		Env         string
		URL         string // public URL of this API
		FrontendURL string // public URL of the SPA, used in emails and share links
	}
	Server struct {
		Port            string
		ReadTimeoutSec  int
		WriteTimeoutSec int
	}
	Database struct {
		Host, Port, User, Password, Name string
		AutoMigrate                      bool
		Seed                             bool
	}
	Redis struct {
		Addr, Password string
		DB             int
	}
	JWT struct {
		Secret     string
		AccessTTL  int // minutes
		RefreshTTL int // days
		Issuer     string
	}
	OTP struct {
		TTL         int // seconds
		MaxAttempts int
	}
	CORS struct {
		Origins []string
	}
	Storage struct {
		Driver   string // local | s3
		LocalDir string
		BaseURL  string
		S3       struct {
			Bucket, Region, Endpoint, AccessKey, SecretKey, PublicBaseURL string
		}
	}
	Mail struct {
		Driver   string // log | smtp
		Host     string
		Port     string
		Username string
		Password string
		From     string
		FromName string
	}
	SMS struct {
		Driver string // log | (vendor implementations plug in here)
	}
	WhatsApp struct {
		Driver        string
		DefaultNumber string
		CountryCode   string
	}
	Rental struct {
		// ListingDays is how long a new rental stays live before the owner must
		// renew it. Sale listings never expire.
		ListingDays int
		// SweepIntervalHours is how often stale rentals are retired and owners
		// warned. Zero switches the background sweep off, leaving the manual
		// administrator trigger as the only way to run it.
		SweepIntervalHours int
	}
	RateLimit struct {
		AuthPerMinute   int
		OTPPerHour      int
		PublicPerMinute int
	}
	Seed struct {
		AdminEmail    string
		AdminPassword string
		AdminMobile   string
		DemoData      bool // sample builders and projects, for development
	}
}

// Load reads the environment, applying development-friendly defaults.
func Load() *Config {
	c := &Config{}
	c.App.Name = getenv("APP_NAME", "RERA Property Platform")
	c.App.Env = getenv("APP_ENV", "development")
	c.App.URL = getenv("APP_URL", "http://localhost:8080")
	c.App.FrontendURL = getenv("FRONTEND_URL", "http://localhost:5173")

	c.Server.Port = getenv("PORT", "8080")
	c.Server.ReadTimeoutSec = getenvInt("SERVER_READ_TIMEOUT", 30)
	c.Server.WriteTimeoutSec = getenvInt("SERVER_WRITE_TIMEOUT", 60)

	c.Database.Host = getenv("DB_HOST", "localhost")
	c.Database.Port = getenv("DB_PORT", "3306")
	c.Database.User = getenv("DB_USER", "root")
	c.Database.Password = getenv("DB_PASSWORD", "")
	c.Database.Name = getenv("DB_NAME", "rera_properties")
	c.Database.AutoMigrate = getenvBool("DB_AUTO_MIGRATE", true)
	c.Database.Seed = getenvBool("DB_SEED", true)

	c.Redis.Addr = getenv("REDIS_ADDR", "localhost:6379")
	c.Redis.Password = getenv("REDIS_PASSWORD", "")
	c.Redis.DB = getenvInt("REDIS_DB", 0)

	c.JWT.Secret = getenv("JWT_SECRET", "change-me-in-production-min-32-chars")
	c.JWT.AccessTTL = getenvInt("JWT_ACCESS_TTL", 15)
	c.JWT.RefreshTTL = getenvInt("JWT_REFRESH_TTL", 7)
	c.JWT.Issuer = getenv("JWT_ISSUER", "rera-property-platform")

	c.OTP.TTL = getenvInt("OTP_TTL", 300)
	c.OTP.MaxAttempts = getenvInt("OTP_MAX_ATTEMPTS", 5)

	c.CORS.Origins = splitAndTrim(getenv("CORS_ORIGINS", "http://localhost:5173,http://localhost:3000"))

	c.Storage.Driver = getenv("STORAGE_DRIVER", "local")
	c.Storage.LocalDir = getenv("STORAGE_LOCAL_DIR", "./uploads")
	c.Storage.BaseURL = getenv("STORAGE_BASE_URL", c.App.URL+"/uploads")
	c.Storage.S3.Bucket = getenv("S3_BUCKET", "")
	c.Storage.S3.Region = getenv("S3_REGION", "")
	c.Storage.S3.Endpoint = getenv("S3_ENDPOINT", "")
	c.Storage.S3.AccessKey = getenv("S3_ACCESS_KEY", "")
	c.Storage.S3.SecretKey = getenv("S3_SECRET_KEY", "")
	c.Storage.S3.PublicBaseURL = getenv("S3_PUBLIC_BASE_URL", "")

	c.Mail.Driver = getenv("MAIL_DRIVER", "log")
	c.Mail.Host = getenv("MAIL_HOST", "")
	c.Mail.Port = getenv("MAIL_PORT", "587")
	c.Mail.Username = getenv("MAIL_USERNAME", "")
	c.Mail.Password = getenv("MAIL_PASSWORD", "")
	c.Mail.From = getenv("MAIL_FROM", "no-reply@example.com")
	c.Mail.FromName = getenv("MAIL_FROM_NAME", c.App.Name)

	c.SMS.Driver = getenv("SMS_DRIVER", "log")

	c.WhatsApp.Driver = getenv("WHATSAPP_DRIVER", "log")
	c.WhatsApp.DefaultNumber = getenv("WHATSAPP_NUMBER", "")
	c.WhatsApp.CountryCode = getenv("WHATSAPP_COUNTRY_CODE", "91")

	c.RateLimit.AuthPerMinute = getenvInt("RATE_LIMIT_AUTH", 10)
	c.RateLimit.OTPPerHour = getenvInt("RATE_LIMIT_OTP", 20)
	c.RateLimit.PublicPerMinute = getenvInt("RATE_LIMIT_PUBLIC", 120)

	c.Rental.ListingDays = getenvInt("RENTAL_LISTING_DAYS", 60)
	c.Rental.SweepIntervalHours = getenvInt("RENTAL_SWEEP_INTERVAL_HOURS", 6)

	c.Seed.AdminEmail = getenv("ADMIN_EMAIL", "admin@reraproperties.local")
	c.Seed.AdminPassword = getenv("ADMIN_PASSWORD", "Admin@12345")
	c.Seed.AdminMobile = getenv("ADMIN_MOBILE", "9000000001")
	c.Seed.DemoData = getenvBool("SEED_DEMO_DATA", true)

	return c
}

// Validate refuses to start a production deployment with development defaults.
func (c *Config) Validate() error {
	if !c.IsProduction() {
		return nil
	}
	if strings.HasPrefix(c.JWT.Secret, "change-me") || len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be set to a value of at least 32 characters in production")
	}
	if c.Database.Password == "" {
		return fmt.Errorf("DB_PASSWORD must be set in production")
	}
	for _, o := range c.CORS.Origins {
		if o == "*" {
			return fmt.Errorf("CORS_ORIGINS must not be a wildcard in production")
		}
	}
	if c.Seed.AdminPassword == "Admin@12345" {
		return fmt.Errorf("ADMIN_PASSWORD must be changed from the development default in production")
	}
	if c.Seed.DemoData {
		return fmt.Errorf("SEED_DEMO_DATA must be false in production")
	}
	return nil
}

func (c *Config) IsProduction() bool {
	env := strings.ToLower(c.App.Env)
	return env == "production" || env == "prod"
}

// DSN builds the MySQL connection string.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.Database.User, c.Database.Password,
		c.Database.Host, c.Database.Port, c.Database.Name,
	)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getenvBool(k string, def bool) bool {
	if v := os.Getenv(k); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
