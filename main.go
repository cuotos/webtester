package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version string = "unset"
	port    int    = 5117
)

func init() {
	info, ok := debug.ReadBuildInfo()
	if ok {
		version = info.Main.Version
	}
}
func main() {

	value, ok := os.LookupEnv("PORT")
	if ok {
		i, _ := strconv.Atoi(value)
		port = i
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	mux := http.DefaultServeMux

	mux.Handle("/healthz", healthzHandler())
	mux.Handle("/", indexHandler())
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/versionz", versionzHandler())
	mux.Handle("/status/{status_code}", statusHandler())

	handler := &MiddlewareHandler{
		Handler: mux,
		Middleware: []func(http.Handler) http.Handler{
			headerMiddleware,
			promMiddleware,
			injectMiddleware,
			middleware.Logger,
		},
	}
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler,
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

func statusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providedStatusCodeString := r.PathValue("status_code")

		providedStatusCode, err := strconv.Atoi(providedStatusCodeString)
		if err != nil || http.StatusText(providedStatusCode) == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("status code provided is not valid: %s", providedStatusCodeString)))
			return
		}

		w.WriteHeader(providedStatusCode)
		w.Write([]byte(http.StatusText(providedStatusCode)))
	}
}

func versionzHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(version))
	}
}

func indexHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/plain")
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

func injectMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cw := &CustomWriter{
			ResponseWriter: w,
		}
		next.ServeHTTP(cw, r)

		hostname, _ := os.Hostname()

		resp := fmt.Sprintf("%s\n%s", hostname, os.Getenv("TEXT"))

		fmt.Fprintf(cw, "\n\n%s", resp)
	})
}

type CustomWriter struct {
	http.ResponseWriter
}

type MiddlewareHandler struct {
	Handler    http.Handler
	Middleware []func(http.Handler) http.Handler
}

func (m *MiddlewareHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler := m.Handler
	for _, middleware := range m.Middleware {
		handler = middleware(handler)
	}
	handler.ServeHTTP(w, r)
}
