package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-redis/redis"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	shutdownGrace = time.Duration(15 * time.Second)
	requestGrace  = time.Duration(1 * time.Second)
)

type statusHandler struct {
	shutdown *int32
}

func (h *statusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status := atomic.LoadInt32(h.shutdown)
	w.WriteHeader(int(status))

	// We'll be kind and do a JSON blob here, as the python
	// version does.
	resp := struct {
		Service string `json:"service"`
		Status  int32  `json:"status"`
	}{
		Service: "graceful-go",
		Status:  status,
	}

	bs, _ := json.Marshal(resp)
	buf := bytes.NewBuffer(bs)

	io.Copy(w, buf)
}

func (h *statusHandler) Shutdown() {
	atomic.StoreInt32(h.shutdown, 500)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{
		"hello": "world",
	}

	bs, _ := json.Marshal(resp)
	buf := bytes.NewBuffer(bs)

	io.Copy(w, buf)
}

func redisHandler(rcli *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()

		log.Printf("redis ping: %#v", rcli.Incr("mykey"))
		dur := time.Since(now)
		log.Printf("duration: %s", dur)

		resp := map[string]string{
			"hello": "world",
			"dur":   fmt.Sprintf("%#v", dur),
		}

		bs, _ := json.Marshal(resp)
		buf := bytes.NewBuffer(bs)

		io.Copy(w, buf)
	}
}

func main() {
	shutdownInt := int32(200)
	sh := &statusHandler{
		shutdown: &shutdownInt,
	}

	duration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "A histogram of latencies for requests.",
			Buckets: []float64{.001, .005, 0.010, 0.05, 0.5, 1.0},
		},
		[]string{},
	)
	prometheus.MustRegister(duration)

	mux := http.NewServeMux()
	mux.HandleFunc("/",
		promhttp.InstrumentHandlerDuration(
			duration,
			http.HandlerFunc(rootHandler)))

	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/status", sh)

	redisdb := redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName:    "mymaster",
		SentinelAddrs: []string{"rfs-shared-redis.default:26379"},
	})

	mux.Handle("/redis", redisHandler(redisdb))

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	signal.Notify(quit, syscall.SIGTERM)

	go func() {
		<-quit

		log.Println("setting status to shutdown")
		sh.Shutdown()

		log.Printf("sleeping for %s before shutdown", shutdownGrace)
		time.Sleep(shutdownGrace)

		ctx, cancel := context.WithTimeout(context.Background(), requestGrace)
		defer cancel()

		log.Printf("shutting down the server")
		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		log.Printf("server shutdown")

		close(done)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on %s: %v\n", ":8080", err)
	}
}
