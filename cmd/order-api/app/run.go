package app

import (
	"context"
	"database/sql"
	"time"

	"github.com/aq2208/gorder-api/configs"
	"github.com/aq2208/gorder-api/internal/adapter/cache"
	"github.com/aq2208/gorder-api/internal/adapter/http"
	"github.com/aq2208/gorder-api/internal/adapter/http/middleware"
	"github.com/aq2208/gorder-api/internal/adapter/observ"
	"github.com/aq2208/gorder-api/internal/adapter/repo"
	"github.com/aq2208/gorder-api/internal/usecase"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

type App struct {
	Router *gin.Engine
}

type ginEngine interface {
	Run(addr ...string) error
}

func InitWithConfig(cfg configs.Config) (*App, func(), error) {
	// init logger
	logger, _ := observ.NewLogger()
	defer logger.Sync()

	// init database
	db, err := sql.Open("mysql", cfg.MySQL.DSN)
	if err != nil {
		return nil, nil, err
	}
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetMaxOpenConns(16)
	db.SetMaxIdleConns(16)

	// init context
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	if err := db.PingContext(ctx); err != nil {
		cancel()
		return nil, nil, err
	}
	cancel()

	// init redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       0,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, nil, err
	}

	orderRepo := repo.NewMySQLOrderRepo(db)
	outboxRepo := repo.NewMySQLOutboxRepo(db)
	idem := cache.NewRedisIdempotencyStore(rdb, cfg.Idempotency.TTL)

	createUC := usecase.NewCreateOrder(orderRepo, idem, outboxRepo)
	h := http.NewOrderHandler(createUC, orderRepo)
	th := http.NewTokenHandler(cfg)
	authz := middleware.NewAuthz(cfg)
	router := http.NewRouter(h, th, authz)

	cleanup := func() {
		_ = db.Close()
		_ = rdb.Close()
	}

	return &App{Router: router}, cleanup, nil
}
