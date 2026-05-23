// Command server runs Arabica and Oolong in one process while keeping each
// app's listener, metrics port, data directory, and SQLite database isolated.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	arabicaapp "tangled.org/arabica.social/arabica/internal/arabica/app"
	coffeehandlers "tangled.org/arabica.social/arabica/internal/arabica/handlers"
	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/atplatform/server"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/logging"
	oolongapp "tangled.org/arabica.social/arabica/internal/oolong/app"
	teahandlers "tangled.org/arabica.social/arabica/internal/oolong/handlers"
	"tangled.org/arabica.social/arabica/internal/routing"

	"github.com/rs/zerolog/log"
)

type appRun struct {
	app                *domain.App
	defaultPort        string
	defaultMetricsPort string
	appRoutes          routing.AppRoutes
	staticPages        handlers.StaticPageRenderers
}

func main() {
	knownDIDsFile := flag.String("known-dids", "", "Path to file containing DIDs to backfill on startup (one per line)")
	flag.Parse()

	logging.ConfigureFromEnv(os.Stdout)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Info().Msg("Signal received, shutting down")
		cancel()
	}()

	runs := []appRun{
		{app: arabicaapp.New(), defaultPort: "18910", defaultMetricsPort: "9101", appRoutes: coffeehandlers.Routes{}},
		{app: oolongapp.New(), defaultPort: "18920", defaultMetricsPort: "9102", appRoutes: teahandlers.Routes{}, staticPages: teahandlers.StaticPages()},
	}

	if err := runApps(ctx, *knownDIDsFile, runs); err != nil {
		log.Fatal().Err(err).Msg("Server exited with error")
	}
	log.Info().Msg("Stopped")
}

func runApps(ctx context.Context, knownDIDsFile string, runs []appRun) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, len(runs))
	var wg sync.WaitGroup
	for _, run := range runs {
		wg.Go(func() {
			log.Info().Str("app", run.app.Name).Msg("Starting app")
			err := server.Run(ctx, run.app, server.Options{
				KnownDIDsPath:      knownDIDsFile,
				DefaultPort:        run.defaultPort,
				DefaultMetricsPort: run.defaultMetricsPort,
				AppRoutes:          run.appRoutes,
				StaticPages:        run.staticPages,
			})
			if err != nil {
				errCh <- fmt.Errorf("%s: %w", run.app.Name, err)
				cancel()
			}
		})
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		select {
		case err := <-errCh:
			return err
		default:
			return nil
		}
	case err := <-errCh:
		cancel()
		<-done
		return err
	case <-ctx.Done():
		<-done
		return nil
	}
}
