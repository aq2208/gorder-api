package app

import (
	"context"
	"database/sql"
	"time"

	"github.com/aq2208/gorder-api/configs"
	"github.com/aq2208/gorder-api/internal/adapter/cache"
	"github.com/aq2208/gorder-api/internal/adapter/grpc"
	"github.com/aq2208/gorder-api/internal/adapter/http"
	"github.com/aq2208/gorder-api/internal/adapter/http/middleware"
	"github.com/aq2208/gorder-api/internal/adapter/kafka"
	"github.com/aq2208/gorder-api/internal/adapter/observ"
	"github.com/aq2208/gorder-api/internal/adapter/queue"
	"github.com/aq2208/gorder-api/internal/adapter/repo"
	"github.com/aq2208/gorder-api/internal/logging"
	"github.com/aq2208/gorder-api/internal/security"
	"github.com/aq2208/gorder-api/internal/usecase"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/rabbitmq/amqp091-go"
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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	if err := db.PingContext(ctx); err != nil {
		cancel()
		return nil, nil, err
	}
	cancel()

	logging.FromCtx(ctx).Info("order-api: Starting up...")

	// init redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       0,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, nil, err
	}

	// init rabbitmq + register [queue-handler]
	conn, _ := amqp091.Dial("amqp://guest:guest@localhost:5672/")
	ch, _ := conn.Channel()

	// load crypto keys
	cm, _ := security.NewCryptoMaterial(cfg)
	cs, _ := security.NewCryptoService(cm)

	// gRPC: connect to order-gw
	grpcConn, closeGRPC, err := InitOrderGWConn(context.Background(), cfg)
	if err != nil {
		return nil, nil, err
	}
	gw := grpc.NewOrderGWClientFromConn(grpcConn, 8*time.Second, "go-order-api/worker")

	// infra
	orderRepo := repo.NewMySQLOrderRepo(db)
	idem := cache.NewRedisIdempotencyStore(rdb, cfg.Idempotency.TTL)
	redisCache := cache.NewRedisCache(rdb, cfg.Cache.TTL)
	producer, err := queue.NewRabbitProducer(ch)
	if err != nil {
		return nil, nil, err
	}

	// register queue-handler
	setupQueue(ch, gw)

	// register kafka-listener
	setupKafkaListener(cfg, orderRepo, redisCache)

	// init handlers + routers + middleware
	createUC := usecase.NewCreateOrder(orderRepo, redisCache, idem, producer)
	h := http.NewOrderHandler(createUC, orderRepo)
	th := http.NewTokenHandler(cfg)
	auth := middleware.NewAuthz(cfg)
	cv := middleware.NewCryptoVerify(cs)
	router := http.NewRouter(h, th, auth, cv)

	cleanup := func() {
		_ = db.Close()
		_ = rdb.Close()
		closeGRPC()
	}

	return &App{Router: router}, cleanup, nil
}

func setupQueue(ch *amqp091.Channel, gw *grpc.OrderGWClient) {
	h := queue.NewOrderCreatedHandler(gw)

	router := queue.NewRouter(ch, queue.WithPrefetch(50))
	router.Register("order.created.q", queue.JSONHandler[usecase.CreatedMsg]{HandleFunc: h.HandleCreate})

	if err := router.Start(); err != nil {
		panic(err)
	}
}

func setupKafkaListener(cfg configs.Config, repo *repo.MySQLOrderRepo, redisCache *cache.RedisCache) {
	grp, err := kafka.NewGroup(cfg.KafkaBroker.KafkaBrokers, cfg.KafkaBroker.KafkaGroupID)
	if err != nil {
		panic(err)
	}

	h := kafka.NewOrderStatusChangedHandler(repo, redisCache)
	consumer := kafka.NewConsumer(grp, []string{cfg.KafkaBroker.KafkaTopic}, h.Handle)

	// Run in background (respect app context if you have one)
	go func() {
		if err := consumer.Start(context.Background()); err != nil {
			panic(err)
		}
	}()
}
