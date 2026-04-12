package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	OAuth       OAuthConfig
	S3          S3Config
	Mail        MailConfig
	Search      SearchConfig
	CORS        CORSConfig
	GalgameWiki GalgameWikiConfig
}

type GalgameWikiConfig struct {
	BaseURL string
}

type ServerConfig struct {
	Port string
	Mode string // "dev" or "prod"
}

type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int // seconds
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type OAuthConfig struct {
	ServerURL    string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	JWTSecret    string
}

type S3Config struct {
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
}

type MailConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

type SearchConfig struct {
	MeilisearchURL string
	MeilisearchKey string
}

type CORSConfig struct {
	AllowOrigins string
}

func Load() (*Config, error) {
	dbURL, err := requireEnv("KUN_DATABASE_URL")
	if err != nil {
		return nil, err
	}
	oauthServerURL, err := requireEnv("OAUTH_SERVER_URL")
	if err != nil {
		return nil, err
	}
	oauthClientID, err := requireEnv("OAUTH_CLIENT_ID")
	if err != nil {
		return nil, err
	}
	oauthClientSecret, err := requireEnv("OAUTH_CLIENT_SECRET")
	if err != nil {
		return nil, err
	}
	oauthRedirectURI, err := requireEnv("OAUTH_REDIRECT_URI")
	if err != nil {
		return nil, err
	}

	return &Config{
		Server: ServerConfig{
			Port: envOrDefault("SERVER_PORT", "2334"),
			Mode: envOrDefault("SERVER_MODE", "dev"),
		},
		Database: DatabaseConfig{
			URL:             dbURL,
			MaxOpenConns:    envOrDefaultInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    envOrDefaultInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: envOrDefaultInt("DB_CONN_MAX_LIFETIME", 300),
		},
		Redis: RedisConfig{
			Host:     envOrDefault("REDIS_HOST", "127.0.0.1"),
			Port:     envOrDefault("REDIS_PORT", "6379"),
			Password: envOrDefault("REDIS_PASSWORD", ""),
			DB:       envOrDefaultInt("REDIS_DB", 0),
		},
		OAuth: OAuthConfig{
			ServerURL:    oauthServerURL,
			ClientID:     oauthClientID,
			ClientSecret: oauthClientSecret,
			RedirectURI:  oauthRedirectURI,
			JWTSecret:    envOrDefault("JWT_SECRET", ""),
		},
		S3: S3Config{
			Endpoint:  envOrDefault("S3_ENDPOINT", ""),
			Region:    envOrDefault("S3_REGION", ""),
			Bucket:    envOrDefault("S3_BUCKET", ""),
			AccessKey: envOrDefault("S3_ACCESS_KEY", ""),
			SecretKey: envOrDefault("S3_SECRET_KEY", ""),
		},
		Mail: MailConfig{
			Host:     envOrDefault("MAIL_HOST", ""),
			Port:     envOrDefaultInt("MAIL_PORT", 587),
			User:     envOrDefault("MAIL_USER", ""),
			Password: envOrDefault("MAIL_PASSWORD", ""),
			From:     envOrDefault("MAIL_FROM", ""),
		},
		Search: SearchConfig{
			MeilisearchURL: envOrDefault("MEILISEARCH_URL", "http://127.0.0.1:7700"),
			MeilisearchKey: envOrDefault("MEILISEARCH_KEY", ""),
		},
		CORS: CORSConfig{
			AllowOrigins: envOrDefault(
				"CORS_ALLOW_ORIGINS",
				"http://127.0.0.1:2333,https://www.kungal.com",
			),
		},
		GalgameWiki: GalgameWikiConfig{
			BaseURL: envOrDefault("GALGAME_WIKI_BASE_URL", "http://127.0.0.1:9280/api"),
		},
	}, nil
}

func requireEnv(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("环境变量 %s 未设置", key)
	}
	return val, nil
}

func envOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func envOrDefaultInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return fallback
}
