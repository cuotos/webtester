package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version string = "unset"
	port    int    = 5117
)

func main() {

	value, ok := os.LookupEnv("PORT")
	if ok {
		i, _ := strconv.Atoi(value)
		port = i
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	r := chi.NewMux()
	r.Use(promMiddleware, headerMiddleware)

	r.Handle("/healthz", healthzHandler())
	r.Handle("/", indexHandler())
	r.Handle("/metrics", promhttp.Handler())
	r.Handle("/versionz", versionzHandler())

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	log.Printf("server listening on port: %d", port)

	<-done

	log.Print("Server Stopped")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown failed: %s", err)
	}
}

func versionzHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(version))
	}
}

func indexHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		hostname, _ := os.Hostname()

		resp := fmt.Sprintf("%s\n%s", hostname, os.Getenv("TEXT"))
		w.Write([]byte(resp))
	}
}

func healthzHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}
}

var httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "webtester_http_duration_seconds",
	Help: "Duration of HTTP requests.",
}, []string{"path"})

func promMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timer := prometheus.NewTimer(httpDuration.WithLabelValues(r.URL.Path))
		next.ServeHTTP(w, r)
		timer.ObserveDuration()
	})
}

func headerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Cuotos-Webtester", "true")
		next.ServeHTTP(w, r)
	})
}
