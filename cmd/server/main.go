package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/djdietrick/radio/internal/api"
	"github.com/djdietrick/radio/internal/auth"
	"github.com/djdietrick/radio/internal/catalog"
	"github.com/djdietrick/radio/internal/catalog/art"
	"github.com/djdietrick/radio/internal/catalog/metadata"
	"github.com/djdietrick/radio/internal/catalog/metadata/external"
	"github.com/djdietrick/radio/internal/catalog/probe"
	"github.com/djdietrick/radio/internal/catalog/scanner"
	"github.com/djdietrick/radio/internal/config"
	"github.com/djdietrick/radio/internal/db"
	"github.com/djdietrick/radio/internal/playlist"
	"github.com/djdietrick/radio/internal/radio"
	"github.com/djdietrick/radio/internal/stream"
	"github.com/djdietrick/radio/internal/users"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Root context cancelled on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Database + migrations.
	database, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer database.Close()
	if err := database.Migrate(ctx); err != nil {
		return err
	}
	log.Info("database ready")

	if err := os.MkdirAll(cfg.ArtCacheDir, 0o755); err != nil {
		log.Warn("could not create art cache dir", "dir", cfg.ArtCacheDir, "err", err)
	}

	// Subsystems.
	catalogStore := catalog.NewStore(database)
	var extractor *metadata.Extractor
	if cfg.ExternalMetadataEnabled {
		extractor = metadata.NewExtractorWithExternal(external.New())
		log.Info("external metadata enabled (MusicBrainz + Cover Art Archive)")
	} else {
		extractor = metadata.NewExtractor(false)
	}
	prober := probe.New()
	artCache := art.New(cfg.ArtCacheDir)
	scn := scanner.New(cfg.MusicDirs, catalogStore, extractor, prober, artCache, log)
	playlistStore := playlist.NewStore(database)
	resolver := playlist.NewResolver(catalogStore)
	radioEngine := radio.NewEngine(database, playlistStore, resolver)
	streamHandler := stream.New(catalogStore)

	// Auth + users.
	authenticator := auth.NewAuthenticator(cfg.JWTSecret, cfg.TokenTTL)
	userStore := users.NewStore(database)
	if applied, err := userStore.BootstrapAdmin(ctx, cfg.DefaultUserID, cfg.AdminUsername, cfg.AdminPassword); err != nil {
		return err
	} else if applied {
		log.Info("bootstrapped admin account", "username", cfg.AdminUsername)
	} else if cfg.AdminPassword == "" {
		log.Warn("no admin password set (RADIO_ADMIN_PASSWORD); no account can log in until one is created")
	}

	// Initial scan (async so the server comes up immediately).
	if cfg.ScanOnStartup {
		go func() {
			if err := scn.ScanAll(ctx); err != nil {
				log.Error("startup scan failed", "err", err)
			}
		}()
	}

	// Filesystem watcher.
	if cfg.WatchEnabled {
		go func() {
			if err := scn.Watch(ctx); err != nil {
				log.Error("watcher stopped", "err", err)
			}
		}()
	}

	// HTTP server.
	handler := api.NewRouter(api.Deps{
		Catalog:   catalogStore,
		Playlists: playlistStore,
		Resolver:  resolver,
		Radio:     radioEngine,
		Stream:    streamHandler,
		Scanner:   scn,
		Art:       artCache,
		Users:     userStore,
		Auth:      authenticator,
		Log:       log,
	})

	srv := &http.Server{Addr: cfg.ListenAddr, Handler: handler}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
