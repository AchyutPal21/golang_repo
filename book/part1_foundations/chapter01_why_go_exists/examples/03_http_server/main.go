// FILE: book/part1_foundations/chapter01_why_go_exists/examples/03_http_server/main.go
// CHAPTER: 01 — Why Go Exists
// TOPIC: A real (tiny) HTTP server in standard-library Go.
//
// Run (from the chapter folder):
//   go run ./examples/03_http_server
// Then in another terminal:
//   curl http://localhost:8080/
//   curl http://localhost:8080/healthz
// Stop with Ctrl-C; you'll see graceful-shutdown messages.
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   To show, in 80 lines, that the things other languages need a framework
//   for — routing, structured logging, graceful shutdown, signal handling —
//   are built into Go's standard library. You'll understand every line of
//   this by Chapter 56. For now, run it and feel the shape.
//
//   This is not a toy: the same patterns scale to a production service.
//   The only things missing for "real" production are TLS (Chapter 55),
//   metrics (Chapter 92), and tracing (Chapter 93). Everything else here
//   you would keep verbatim.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// ─── start time, captured once at program start ─────────────────────────────
//
// Package-level variables are initialized before main runs. We use
// `start` so the / handler can show how long the server has been up.
// Don't do this for anything expensive — initialization order is
// covered in Chapter 8.
var start = time.Now()

func main() {
	// ─── Logger ─────────────────────────────────────────────────────────
	//
	// log/slog is Go's structured logger, added in Go 1.21. The handler
	// formats records (here as JSON) and writes them to stderr. In
	// production you would feed this into Loki, Elastic, or your
	// centralised log pipeline. Chapter 63 covers it in depth.
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// ─── Routes ─────────────────────────────────────────────────────────
	//
	// http.NewServeMux returns the standard-library router. Since Go 1.22
	// it supports method-prefixed patterns ("GET /") and path parameters
	// ("/users/{id}"). Chapter 58 evaluates when this is enough and when
	// you'd reach for chi or gin.
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		// Plain text response. The Content-Type is inferred for short
		// responses, but for anything non-trivial set it explicitly.
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("hello from Go " +
			"— uptime " + time.Since(start).Round(time.Millisecond).String() + "\n"))
	})

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		// A real health endpoint would check downstream dependencies
		// (DB, Redis, etc.). For Chapter 1, "we're alive" is enough.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	// ─── A logging middleware ───────────────────────────────────────────
	//
	// http.Handler is an interface with a single method, ServeHTTP. We
	// wrap our mux in a function that logs each request and then delegates.
	// This is the standard "decorator" pattern in Go HTTP — Chapter 59
	// covers it in depth.
	logged := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		mux.ServeHTTP(w, r)
		logger.Info("request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Duration("latency", time.Since(t)),
			slog.String("remote", r.RemoteAddr),
		)
	})

	// ─── Server with sane timeouts ──────────────────────────────────────
	//
	// http.Server's zero value is *not* safe to expose to the public
	// internet — it has no timeouts, which means a malicious client can
	// hold a connection open forever. Always set ReadHeaderTimeout at
	// minimum. Chapter 56 walks through every timeout.
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           logged,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// ─── Run the server in a background goroutine ───────────────────────
	//
	// We do this so the main goroutine is free to wait on a signal.
	// The server runs until it receives ErrServerClosed (from Shutdown).
	go func() {
		logger.Info("listening", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			logger.Error("listen failed", slog.Any("err", err))
			os.Exit(1)
		}
	}()

	// ─── Wait for SIGINT or SIGTERM ─────────────────────────────────────
	//
	// signal.Notify hooks the named OS signals into our `quit` channel.
	// `<-quit` blocks until the first signal arrives. Ctrl-C in your
	// terminal sends SIGINT.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	sig := <-quit
	logger.Info("shutting down", slog.String("signal", sig.String()))

	// ─── Graceful shutdown ──────────────────────────────────────────────
	//
	// Give in-flight requests up to 10 seconds to finish. After that,
	// drop them. This is the standard production pattern; Chapter 56
	// explains why every timeout matters.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown failed", slog.Any("err", err))
		os.Exit(1)
	}
	logger.Info("bye")
}
