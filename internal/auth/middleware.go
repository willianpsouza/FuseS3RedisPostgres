package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func APIKey(expected string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if expected == "" || c.GetHeader("X-API-Key") == expected {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	}
}

func RateLimit(rdb *redis.Client, rps int) gin.HandlerFunc {
	window := time.Second
	return func(c *gin.Context) {
		k := "ratelimit:" + c.ClientIP() + ":" + c.GetHeader("X-API-Key")
		count, _ := rdb.Incr(c, k).Result()
		if count == 1 {
			rdb.Expire(c, k, window)
		}
		if count > int64(rps) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
