module github.com/example/fuses3redispostgres

go 1.22

require (
	github.com/aws/aws-sdk-go-v2/config v1.27.27
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.17.16
	github.com/aws/aws-sdk-go-v2/service/s3 v1.58.3
	github.com/gin-gonic/gin v1.10.0
	github.com/hanwen/go-fuse/v2 v2.5.1
	github.com/jackc/pgx/v5 v5.6.0
	github.com/prometheus/client_golang v1.20.1
	github.com/redis/go-redis/v9 v9.6.1
	github.com/spf13/viper v1.19.0
	go.etcd.io/bbolt v1.3.10
	go.uber.org/zap v1.27.0
	golang.org/x/sync v0.8.0
	golang.org/x/time v0.6.0
)
