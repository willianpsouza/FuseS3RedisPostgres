package main

import (
	"net/http"
	"time"

	"github.com/example/fuses3redispostgres/internal/config"
	"github.com/example/fuses3redispostgres/internal/logging"
	"github.com/example/fuses3redispostgres/internal/scanner"
	"go.etcd.io/bbolt"
	"golang.org/x/time/rate"
)

func main() {
	cfg, err := config.Load("")
	if err != nil {
		panic(err)
	}
	log, _ := logging.New(cfg.LogLevel)
	db, err := bbolt.Open("scanner-state.db", 0600, nil)
	if err != nil {
		panic(err)
	}
	agent := &scanner.Agent{Dirs: cfg.ScanDirs, APIBaseURL: "http://localhost:8080", APIKey: cfg.APIKey, DB: db, Log: log, Limit: rate.NewLimiter(rate.Limit(10), 20), Workers: 4}
	go http.ListenAndServe(":18080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte("ok")) }))
	for {
		if err := agent.RunOnce(); err != nil {
			log.Error("scan loop")
		}
		time.Sleep(30 * time.Second)
	}
}
