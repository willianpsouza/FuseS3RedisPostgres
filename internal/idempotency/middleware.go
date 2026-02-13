package idempotency

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func Middleware(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		idk := c.GetHeader("Idempotency-Key")
		if idk == "" {
			c.Next()
			return
		}
		ok, _ := rdb.SetNX(c, "idem:"+idk, "1", 10*time.Minute).Result()
		if !ok {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "duplicated idempotency key"})
			return
		}
		c.Next()
	}
}
