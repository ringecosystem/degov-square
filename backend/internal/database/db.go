package database

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	migratePg "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() error {
	var err error

	// build database connection string
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_SSLMODE"),
	)

	// connect to the database
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	slog.Info("Successfully connected to database")

	// run database migrations
	if err := runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// // auto migrate models
	// if err := autoMigrate(); err != nil {
	// 	return fmt.Errorf("failed to auto migrate: %w", err)
	// }

	return nil
}

func runMigrations() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	driver, err := migratePg.WithInstance(sqlDB, &migratePg.Config{})
	if err != nil {
		return err
	}

	// create a new migration instance
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return err
	}

	// run the migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	slog.Info("Database migrations completed successfully")
	return nil
}

// func autoMigrate() error {
// 	err := DB.AutoMigrate(
// 		&models.User{},
// 	)
// 	if err != nil {
// 		return err
// 	}
// 	slog.Info("Auto migration completed successfully")
// 	return nil
// }

func GetDB() *gorm.DB {
	return DB
}
