package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/dgraph-io/badger/v2"

	"github.com/roman-kulish/spatio-temporal-deduplication-example/cmd/example/app/config"
	"github.com/roman-kulish/spatio-temporal-deduplication-example/cmd/example/app/dedup"
	"github.com/roman-kulish/spatio-temporal-deduplication-example/cmd/example/app/server"
)

// Run is the second "main", which initialises and wires services and starts
// the application. Services which must be finalised on termination can
// register their finalise methods via defer. Run function does not panic or
// terminates via os.Exit() and so it guarantees graceful services shutdown.
func Run() (err error) {
	defer errorHandler(&err)

	cfg, err := config.NewFromEnv()
	if err != nil {
		return err
	}

	db, err := badger.Open(badger.DefaultOptions(cfg.DBPath).WithLogger(nil))
	if err != nil {
		return err
	}

	filter, err := dedup.NewSpatioTemporalFilter(db, cfg.Tolerance.Distance, cfg.Tolerance.Interval)
	if err != nil {
		return err
	}

	f, ok := filter.(*dedup.SpatioTemporalFilter)
	if !ok {
		return fmt.Errorf("filter must be an instance of %T, got %T", (*dedup.SpatioTemporalFilter)(nil), filter)
	}

	srv, err := server.New(&cfg.Server, f)
	if err != nil {
		return err
	}

	done := make(chan error, 1)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

		<-quit

		ctx, cancel := context.Background(), func() {}
		if cfg.Server.ShutdownTimeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, cfg.Server.ShutdownTimeout)
			defer cancel()
		}

		done <- srv.Shutdown(ctx)
	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return <-done
}

func errorHandler(err *error) {
	if r := recover(); r != nil {
		switch v := r.(type) {
		case error:
			*err = v
		default:
			*err = fmt.Errorf("panic: %v", v)
		}
	}
}
