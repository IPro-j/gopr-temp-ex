package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Host         string
	Port         int
	User         string
	Password     string
	DBName       string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
}

// NewPostgresDB создает новое подключение к PostgreSQL
func NewPostgresDB(cfg Config) (*sql.DB, error) {

	// 1. Создать DSN (Data Source Name) строку в формате:
	//    host=<host> port=<port> user=<user> password=<password> dbname=<dbname> sslmode=<sslmode>
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	// 2. Открыть подключение используя sql.Open("postgres", dsn)'
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// 3. Проверить подключение используя db.Ping()
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)

	return db, nil
}

// RunMigrations выполняет SQL миграции
func RunMigrations(db *sql.DB, migrations []string) error {
	if len(migrations) == 0 {
		logrus.WithField("component", "migrations").Info("no migrations to apply")
		return nil
	}

	logrus.WithField("component", "migrations").Infof("starting %d migration(s)", len(migrations))

	for i, path := range migrations {
		// Читаем файл
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("migration %d (%s): failed to read file: %w", i+1, path, err)
		}

		query := string(content)

		// Выполняем SQL
		_, err = db.Exec(query)
		if err != nil {
			return fmt.Errorf("migration %d (%s) failed: %w", i+1, path, err)
		}

	}

	return nil
}
