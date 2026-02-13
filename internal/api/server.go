package api

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/example/fuses3redispostgres/internal/auth"
	"github.com/example/fuses3redispostgres/internal/config"
	"github.com/example/fuses3redispostgres/internal/idempotency"
	"github.com/example/fuses3redispostgres/internal/metadata"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var dateRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

type Server struct {
	cfg      config.App
	log      *zap.Logger
	repo     *metadata.Repository
	resolver *metadata.Resolver
	uploader *manager.Uploader
	redis    *redis.Client
}

func New(cfg config.App, log *zap.Logger, repo *metadata.Repository, resolver *metadata.Resolver, s3c *s3.Client, rdb *redis.Client) *Server {
	return &Server{cfg: cfg, log: log, repo: repo, resolver: resolver, uploader: manager.NewUploader(s3c), redis: rdb}
}

func (s *Server) Router() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), auth.APIKey(s.cfg.APIKey), auth.RateLimit(s.redis, s.cfg.RateLimitRPS), idempotency.Middleware(s.redis))
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.POST("/v1/upload", s.upload)
	r.GET("/v1/resolve", s.resolve)
	return r
}

func (s *Server) resolve(c *gin.Context) {
	filename := c.Query("filename")
	obj, err := s.resolver.Resolve(c.Request.Context(), filename)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, obj)
}

func (s *Server) upload(c *gin.Context) {
	dateRaw, filename := c.Query("date"), c.Query("filename")
	if !dateRE.MatchString(dateRaw) || filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date or filename"})
		return
	}
	dateVal, _ := time.Parse("2006-01-02", dateRaw)
	bucket, key := decideBucketKey(dateVal, filename)
	file, closeFn, err := extractReader(c, filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer closeFn()
	md5h := md5.New()
	sha := sha256.New()
	tee := io.TeeReader(file, io.MultiWriter(md5h, sha))

	cr := &countingReader{r: tee}
	upOut, err := s.uploader.Upload(context.Background(), &s3.PutObjectInput{Bucket: &bucket, Key: &key, Body: cr})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "s3 upload failed"})
		return
	}
	obj := metadata.Object{Filename: filename, Bucket: bucket, Key: key, Size: cr.n, ETag: ptrStr(upOut.ETag), LastModified: time.Now().UTC(), ChecksumMD5: ptr(hex.EncodeToString(md5h.Sum(nil))), ChecksumSHA: ptr(hex.EncodeToString(sha.Sum(nil)))}
	if err := s.repo.UpsertObject(c.Request.Context(), obj, dateVal, "active"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "metadata upsert failed"})
		return
	}
	s.redis.Publish(c.Request.Context(), "object_ingested", fmt.Sprintf("%s|%s|%s", filename, bucket, key))
	c.JSON(http.StatusOK, gin.H{"bucket": bucket, "key": key, "size": obj.Size, "etag": obj.ETag, "checksums": gin.H{"md5": obj.ChecksumMD5, "sha256": obj.ChecksumSHA}})
}

func extractReader(c *gin.Context, fallbackName string) (io.Reader, func(), error) {
	if c.ContentType() == "application/octet-stream" {
		return c.Request.Body, func() { _ = c.Request.Body.Close() }, nil
	}
	f, _, err := c.Request.FormFile("file")
	if err != nil {
		return nil, func() {}, fmt.Errorf("missing form file: %w", err)
	}
	return f, func() { _ = f.Close() }, nil
}

func decideBucketKey(date time.Time, filename string) (string, string) {
	bucket := fmt.Sprintf("data-%d", date.Year())
	h := sha256.Sum256([]byte(filename))
	prefix := hex.EncodeToString(h[:])[:4]
	key := path.Join(date.Format("2006/01/02"), prefix, filename)
	return bucket, key
}

func ptr(s string) *string { return &s }
func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func closeMultipart(f multipart.File) { _ = f.Close() }

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}
