package main

import (
	"context"
	"net/http"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/example/fuses3redispostgres/internal/config"
	"github.com/example/fuses3redispostgres/internal/fusefs"
	"github.com/example/fuses3redispostgres/internal/logging"
	"github.com/example/fuses3redispostgres/internal/metadata"
	"github.com/example/fuses3redispostgres/internal/s3io"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := config.Load("")
	if err != nil {
		panic(err)
	}
	log, _ := logging.New(cfg.LogLevel)
	ctx := context.Background()
	pg, _ := pgxpool.New(ctx, cfg.PostgresDSN)
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword, DB: cfg.RedisDB})
	awsCfg, _ := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.S3Region))
	s3c := s3.NewFromConfig(awsCfg)
	repo := metadata.NewRepository(pg)
	resolver := metadata.NewResolver(repo, rdb, 50000, 30*time.Minute)
	reader := s3io.NewReader(s3c, cfg.GlobalS3Limit, cfg.PerBucketS3Limit)
	root := fusefs.NewRoot(resolver, reader, cfg.BlockSizeBytes, cfg.PrefetchSizeByte)
	server, err := fs.Mount(cfg.FuseMountPoint, root, &fs.Options{MountOptions: fuse.MountOptions{FsName: "virtualfs", Name: "virtualfs", ReadOnly: true}})
	if err != nil {
		panic(err)
	}
	go http.ListenAndServe(cfg.MetricsAddr, promhttp.Handler())
	log.Info("fuse mounted")
	server.Wait()
}
