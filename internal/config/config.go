package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type App struct {
	ServiceName      string
	LogLevel         string
	HTTPAddr         string
	MetricsAddr      string
	PostgresDSN      string
	RedisAddr        string
	RedisPassword    string
	RedisDB          int
	S3Region         string
	S3Endpoint       string
	CacheDir         string
	CacheSizeBytes   int64
	BlockSizeBytes   int64
	PrefetchSizeByte int64
	GlobalS3Limit    int64
	PerBucketS3Limit int64
	Timeout          time.Duration
	APIKey           string
	RateLimitRPS     int
	FuseMountPoint   string
	ScanDirs         []string
}

func Load(path string) (App, error) {
	v := viper.New()
	v.SetConfigType("env")
	if path != "" {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			return App{}, fmt.Errorf("read config: %w", err)
		}
	}
	v.AutomaticEnv()
	v.SetDefault("SERVICE_NAME", "virtualfs")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("HTTP_ADDR", ":8080")
	v.SetDefault("METRICS_ADDR", ":9090")
	v.SetDefault("CACHE_DIR", "/var/cache/virtualfs")
	v.SetDefault("CACHE_SIZE_BYTES", int64(10*1024*1024*1024))
	v.SetDefault("BLOCK_SIZE_BYTES", int64(8*1024*1024))
	v.SetDefault("PREFETCH_SIZE_BYTES", int64(32*1024*1024))
	v.SetDefault("GLOBAL_S3_LIMIT", int64(200))
	v.SetDefault("PER_BUCKET_S3_LIMIT", int64(20))
	v.SetDefault("TIMEOUT", "30s")
	v.SetDefault("RATE_LIMIT_RPS", 50)
	v.SetDefault("FUSE_MOUNT_POINT", "/mnt/virtualfs")

	timeout, err := time.ParseDuration(v.GetString("TIMEOUT"))
	if err != nil {
		return App{}, fmt.Errorf("parse TIMEOUT: %w", err)
	}
	return App{
		ServiceName:      v.GetString("SERVICE_NAME"),
		LogLevel:         v.GetString("LOG_LEVEL"),
		HTTPAddr:         v.GetString("HTTP_ADDR"),
		MetricsAddr:      v.GetString("METRICS_ADDR"),
		PostgresDSN:      v.GetString("POSTGRES_DSN"),
		RedisAddr:        v.GetString("REDIS_ADDR"),
		RedisPassword:    v.GetString("REDIS_PASSWORD"),
		RedisDB:          v.GetInt("REDIS_DB"),
		S3Region:         v.GetString("S3_REGION"),
		S3Endpoint:       v.GetString("S3_ENDPOINT"),
		CacheDir:         v.GetString("CACHE_DIR"),
		CacheSizeBytes:   v.GetInt64("CACHE_SIZE_BYTES"),
		BlockSizeBytes:   v.GetInt64("BLOCK_SIZE_BYTES"),
		PrefetchSizeByte: v.GetInt64("PREFETCH_SIZE_BYTES"),
		GlobalS3Limit:    v.GetInt64("GLOBAL_S3_LIMIT"),
		PerBucketS3Limit: v.GetInt64("PER_BUCKET_S3_LIMIT"),
		Timeout:          timeout,
		APIKey:           v.GetString("API_KEY"),
		RateLimitRPS:     v.GetInt("RATE_LIMIT_RPS"),
		FuseMountPoint:   v.GetString("FUSE_MOUNT_POINT"),
		ScanDirs:         splitCSV(v.GetString("SCAN_DIRS")),
	}, nil
}

func splitCSV(in string) []string {
	if in == "" {
		return nil
	}
	parts := strings.Split(in, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
