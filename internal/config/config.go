package config

import (
	"crypto/rand"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime configuration, populated from environment variables.
type Config struct {
	// HTTP
	ListenAddr string

	// Database
	DatabaseURL string

	// Library
	// MusicDirs is the set of root directories scanned for audio files.
	MusicDirs []string
	// ArtCacheDir is where extracted/fetched album art is stored.
	ArtCacheDir string

	// Scanning
	ScanOnStartup bool
	WatchEnabled  bool

	// Metadata
	// ExternalMetadataEnabled toggles MusicBrainz/Cover Art Archive fallback.
	ExternalMetadataEnabled bool

	// DefaultUserID is the id of the seeded user that owns any pre-auth data.
	// On startup it is bootstrapped into the admin account using the
	// AdminUsername / AdminPassword credentials below.
	DefaultUserID string

	// Auth
	// JWTSecret signs and verifies bearer tokens. Must be set in production;
	// a random per-process secret is generated when unset (tokens won't survive
	// a restart in that case).
	JWTSecret []byte
	// TokenTTL is how long an issued JWT remains valid.
	TokenTTL time.Duration
	// AdminUsername / AdminPassword bootstrap the default user as an admin on
	// startup. When AdminPassword is empty, no bootstrap occurs (an already-set
	// password is never overwritten).
	AdminUsername string
	AdminPassword string

	// ShutdownTimeout bounds graceful shutdown.
	ShutdownTimeout time.Duration
}

// Load reads configuration from the environment, applying sensible defaults.
func Load() (*Config, error) {
	c := &Config{
		ListenAddr:              env("RADIO_LISTEN_ADDR", ":8080"),
		DatabaseURL:             env("RADIO_DATABASE_URL", "postgres://radio:radio@localhost:5432/radio?sslmode=disable"),
		MusicDirs:               splitNonEmpty(env("RADIO_MUSIC_DIRS", "/music")),
		ArtCacheDir:             env("RADIO_ART_CACHE_DIR", "/data/art"),
		ScanOnStartup:           envBool("RADIO_SCAN_ON_STARTUP", true),
		WatchEnabled:            envBool("RADIO_WATCH_ENABLED", true),
		ExternalMetadataEnabled: envBool("RADIO_EXTERNAL_METADATA", false),
		DefaultUserID:           env("RADIO_DEFAULT_USER_ID", "00000000-0000-0000-0000-000000000001"),
		TokenTTL:                envDuration("RADIO_TOKEN_TTL", 24*time.Hour),
		AdminUsername:           env("RADIO_ADMIN_USERNAME", "admin"),
		AdminPassword:           env("RADIO_ADMIN_PASSWORD", ""),
		ShutdownTimeout:         envDuration("RADIO_SHUTDOWN_TIMEOUT", 15*time.Second),
	}

	if secret := env("RADIO_JWT_SECRET", ""); secret != "" {
		c.JWTSecret = []byte(secret)
	} else {
		// Generate an ephemeral secret so the server still runs out of the box;
		// tokens are invalidated on restart. Production must set RADIO_JWT_SECRET.
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			return nil, fmt.Errorf("config: generate ephemeral JWT secret: %w", err)
		}
		c.JWTSecret = buf
	}

	if len(c.MusicDirs) == 0 {
		return nil, fmt.Errorf("config: at least one music directory is required (RADIO_MUSIC_DIRS)")
	}
	return c, nil
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func envDuration(key string, def time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func splitNonEmpty(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
