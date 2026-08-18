package main

import (
	"advanced-blog-management-system/pkg/database"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := log.New(os.Stdout, "[API] ", log.LstdFlags|log.Lshortfile)

	logrus.SetFormatter(&logrus.JSONFormatter{})
	// в Docker лучше level Info, в dev — Debug
	logrus.SetLevel(logrus.DebugLevel)

	// Создаём экземпляр  middleware
	loggingMW := middleware.NewLoggingMiddleware(logger)

	// TODO: Загрузить переменные окружения из .env файла

	if err := godotenv.Load(); err != nil {
		logger.Printf("Warning: .env file not found, using environment variables directly")
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

	defer db.Close()
	logger.Println("Database connected successfully")

	// TODO: Создать подключение к PostgreSQL
	// Используйте функцию database.NewPostgresDB из pkg/database
	// Передайте конфигурацию БД
	// Обработайте возможные ошибки подключения

	// TODO: Выполнить миграции БД
	// Прочитайте SQL файлы из папки migrations/
	// Выполните их в нужном порядке:
	// 1. 001_init_schema.sql (создание таблиц)
	// 2. 002_add_indexes.sql (создание индексов)
	// Используйте функцию database.RunMigrations()

	migrations := []string{
		"./migrations/001_init_schema.sql",
		"./migrations/002_add_indexes.sql",
	}

	if err := database.RunMigrations(db, migrations); err != nil {
		logrus.WithError(err).
			WithField("component", "migrations").
			Fatal("migrations failed")
	}
	//logrus.Info("migrations completed")
	logrus.WithField("component", "migrations").Info("migrations completed successfully")

	// TODO: Инициализировать репозитории
	// Создайте экземпляры:
	// - UserRepository
	// - PostRepository
	// - CommentRepository
	// Передайте им подключение к БД

	// TODO: Инициализировать сервисы
	// Создайте экземпляры:
	// - UserService (передайте userRepository)
	// - PostService (передайте postRepository, userRepository, commentRepository)
	// - CommentService (передайте commentRepository, postRepository, userRepository)

	// TODO: Инициализировать обработчики (handlers)
	// Создайте экземпляры:
	// - AuthHandler (передайте userService и JWT_SECRET)
	// - PostHandler (передайте postService)
	// - CommentHandler (передайте commentService)

	// TODO: Создать и настроить HTTP роутер
	// Вызовите функцию setupRouter() которая вернет chi.Mux
	// Роутер должен содержать:
	// - Все middleware (логирование, recovery, CORS, аутентификация где нужна)
	// - Все HTTP эндпоинты согласно спецификации

	// TODO: Создать HTTP сервер
	// Создайте структуру http.Server с:
	// - Addr: полученный из конфигурации адрес и порт
	// - Handler: роутер
	// - ReadTimeout: 15 секунд
	// - WriteTimeout: 15 секунд
	// - IdleTimeout: 60 секунд

	// TODO: Запустить сервер в отдельной горутине
	// go func() { ... }()
	// Обработайте ошибку http.ErrServerClosed как успех (это нормально при shutdown)

	// TODO: Установить обработчик сигналов завершения
	// Перехватите сигналы SIGINT и SIGTERM (ctrl+C, kill и т.д.)
	// Используйте os.Signal и signal.Notify()

	// TODO: Graceful shutdown
	// Когда получен сигнал завершения:
	// 1. Логируйте информацию о завершении
	// 2. Создайте контекст с таймаутом (5-10 секунд)
	// 3. Вызовите srv.Shutdown(ctx) для корректного завершения
	// 4. Закройте подключение к БД: db.Close()
	// 5. Выйдите из программы

	// TODO: Логирование
	// В главной функции логируйте ключевые события:
	// - Загрузка конфигурации
	// - Подключение к БД
	// - Выполнение миграций
	// - Запуск сервера (какой адрес и порт)
	// - Получение сигнала завершения
	// - Результат shutdown

	log.Println("Server starting... (TODO: implement main.go)")
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
		DBUser:                  getEnv("DB_USER", "blouser"),
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
