package main

import (
	"context"
	"net/http"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/example/fuses3redispostgres/internal/api"
	"github.com/example/fuses3redispostgres/internal/config"
	"github.com/example/fuses3redispostgres/internal/logging"
	"github.com/example/fuses3redispostgres/internal/metadata"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := config.Load("")
	if err != nil {
		panic(err)
	}
	log, _ := logging.New(cfg.LogLevel)
	ctx := context.Background()
	pg, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		panic(err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword, DB: cfg.RedisDB})
	awsCfg, _ := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.S3Region))
	s3c := s3.NewFromConfig(awsCfg)
	repo := metadata.NewRepository(pg)
	resolver := metadata.NewResolver(repo, rdb, 10000, 10*time.Minute)
	srv := api.New(cfg, log, repo, resolver, s3c, rdb)
	log.Info("ingest-api listening")
	if err := http.ListenAndServe(cfg.HTTPAddr, srv.Router()); err != nil {
		panic(err)
	}
}
