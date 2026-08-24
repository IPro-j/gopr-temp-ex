package main

import (
	"advanced-blog-management-system/internal/handler"
	"advanced-blog-management-system/internal/middleware"
	"advanced-blog-management-system/internal/repository"
	"advanced-blog-management-system/internal/service"
	"advanced-blog-management-system/pkg/auth"
	"advanced-blog-management-system/pkg/database"
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {

	// в Docker лучше level Info, в dev — Debug
	logrus.SetLevel(logrus.DebugLevel)

	// Создаём экземпляр  middleware
	loggingMW := middleware.NewLoggingMiddleware(logrus.StandardLogger())

	if err := godotenv.Load(); err != nil {
		//logger.Printf("Warning: .env file not found, using environment variables directly")
		logrus.WithError(err).
			WithField("component", "config").
			Warn("`.env` file not found, using environment variables directly")

	}

	cfg := loadConfig()

	dbConfig := database.Config{
		Host:         cfg.DBHost,
		Port:         cfg.DBPort,
		User:         cfg.DBUser,
		Password:     cfg.DBPassword,
		DBName:       cfg.DBName,
		SSLMode:      cfg.DBSSLMode,
		MaxOpenConns: cfg.DBMaxOpenConns,
		MaxIdleConns: cfg.DBMaxOpenConns,
	}

	db, err := database.NewPostgresDB(dbConfig)
	if err != nil {
		logrus.WithError(err).WithField("component", "database").Fatalf("failed to connect to database")
	}

	logrus.WithField("component", "database").Info("mDatabase connected successfully")

	migrations := []string{
		"./migrations/001_init_schema.sql",
		"./migrations/002_add_indexes.sql",
	}

	if err := database.RunMigrations(db, migrations); err != nil {
		logrus.WithError(err).
			WithField("component", "migrations").
			Fatal("migrations failed")
	}

	logrus.WithField("component", "migrations").Info("migrations completed successfully")

	jwtManager, err := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiryHours)
	if err != nil {
		logrus.WithError(err).
			WithField("component", "jwt").
			Fatalf("failed to create JWT manager: %v", err)
	}

	postRepo := repository.NewPostRepo(db)
	commentRepo := repository.NewCommentRepo(db)
	userRepo := repository.NewUserRepo(db)

	userService := service.NewUserService(userRepo, jwtManager)
	postService := service.NewPostService(postRepo, userRepo)
	commentService := service.NewCommentService(commentRepo, postRepo)

	authHandler := handler.NewAuthHandler(userService, jwtManager)
	postHandler := handler.NewPostHandler(postService)
	commentHandler := handler.NewCommentHandler(commentService)

	authMW := middleware.NewAuthMiddleware(jwtManager)

	router := chi.NewRouter()

	router.Use(loggingMW.Chain())

	router.Route("/api", func(r chi.Router) {

		//проверка состояния сервиса (GET /api/health)
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok","service":"blog-api"}`))
		})

		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Get("/posts", postHandler.GetAll)
		r.Get("/posts/{id}", postHandler.GetByID)
		r.Get("/posts/{id}/comments", commentHandler.GetByPost)

		r.Group(func(r chi.Router) {
			r.Use(authMW.AuthMiddlewareForChi()) // авторизация только тут
			r.Post("/posts", postHandler.Create)
			r.Patch("/posts/{id}", postHandler.Update)
			r.Delete("/posts/{id}", postHandler.Delete)
			r.Post("/posts/{id}/comments", commentHandler.Create)
			r.Put("/comments/{id}", commentHandler.Update)
			r.Delete("/comments/{id}", commentHandler.Delete)
		})

	})

	srv := &http.Server{
		Addr:         cfg.ServerHost + ":" + strconv.Itoa(cfg.ServerPort),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.HttpReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.HttpWriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.HttpIdleTimeout) * time.Second,
	}

	logrus.
		WithField("component", "http_server").
		WithField("addr", srv.Addr).
		WithField("read_timeout_seconds", srv.ReadTimeout).
		WithField("write_timeout_seconds", srv.WriteTimeout).
		WithField("idle_timeout_seconds", srv.IdleTimeout).
		Info("starting HTTP server")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatal("server failed to start")
			//logger.Fatalf("Server failed to start: %v", err)
		}
	}()

	<-stop
	logrus.Info("shutdown signal received")
	//	logger.Println("Shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.HttpShutdownGracePeriod))
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logrus.WithError(err).Error("graceful shutdown failed")
	} else {
		logrus.Info("server stopped gracefully")
	}

}

type Config struct {
	ServerHost              string
	ServerPort              int
	DBHost                  string
	DBPort                  int
	DBUser                  string
	DBPassword              string
	DBName                  string
	DBSSLMode               string
	DBMaxOpenConns          int
	DBMaxIdleConns          int
	JWTSecret               string
	JWTExpiryHours          int
	CacheTTLMinutes         int
	HttpReadTimeout         int
	HttpWriteTimeout        int
	HttpIdleTimeout         int
	HttpShutdownGracePeriod int
	RateLimitEnabled        bool
	RateLimitWindowSecond   int
	RateLimitMaxRequest     int
}

func loadConfig() *Config {
	return &Config{
		ServerHost:              getEnv("SERVER_HOST", "0.0.0.0"),
		ServerPort:              getEnvAsInt("SERVER_PORT", 8080),
		DBHost:                  getEnv("DB_HOST", "localhost"),
		DBPort:                  getEnvAsInt("DB_PORT", 5432),
		DBUser:                  getEnv("DB_USER", "bloguser"),
		DBPassword:              getEnv("DB_PASSWORD", "blogpassword"),
		DBName:                  getEnv("DB_NAME", "blogdb"),
		DBSSLMode:               getEnv("DB_SSLMODE", "disable"),
		DBMaxOpenConns:          getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:          getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
		JWTSecret:               getEnv("JWT_SECRET", ""),
		JWTExpiryHours:          getEnvAsInt("JWT_EXPIRY_HOURS", 24),
		CacheTTLMinutes:         getEnvAsInt("CACHE_TTL_MINUTES", 60),
		HttpReadTimeout:         getEnvAsInt("HTTP_READ_TIMEOUT", 15),
		HttpWriteTimeout:        getEnvAsInt("HTTP_WRITE_TIMEOUT", 15),
		HttpIdleTimeout:         getEnvAsInt("HTTP_IDLE_TIMEOUT", 60),
		HttpShutdownGracePeriod: getEnvAsInt("HTTP_SHUTDOWN_GRACE_PERIOD", 30),
		RateLimitEnabled:        os.Getenv("RATE_LIMIT_ENABLED") == "true",
		RateLimitWindowSecond:   getEnvAsInt("RATE_LIMIT_WINDOW_SECONDS", 60),
		RateLimitMaxRequest:     getEnvAsInt("RATE_LIMIT_MAX_REQUESTS", 100),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func setupRouter() interface{} {
	// TODO: Создать chi роутер
	// Используйте chi.NewRouter()

	// TODO: Зарегистрировать middleware в порядке:
	// 1. LoggingMiddleware (должен быть первым)
	// 2. RecoveryMiddleware
	// 3. CORSMiddleware

	// TODO: Зарегистрировать все публичные эндпоинты:
	// - GET /api/health (HealthCheckHandler)
	// - POST /api/register
	// - POST /api/login
	// - GET /api/posts
	// - GET /api/posts/{id}
	// - GET /api/users/{id}/posts
	// - GET /api/posts/{postId}/comments

	// TODO: Зарегистрировать защищенные эндпоинты (требуют AuthMiddleware):
	// - POST /api/posts
	// - PUT /api/posts/{id}
	// - DELETE /api/posts/{id}
	// - POST /api/posts/{postId}/comments
	// - PUT /api/comments/{id}
	// - DELETE /api/comments/{id}
	//
	// Подсказка для защиты: используйте r.With(middleware.AuthMiddleware)
	// или создайте отдельный роут-группу для защищенных эндпоинтов

	// TODO: Вернуть настроенный роутер
	return nil
}
