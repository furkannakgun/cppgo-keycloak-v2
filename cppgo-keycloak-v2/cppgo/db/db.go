package db

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func SetupDatabase(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&PhoneNumber{}, &CallLog{}); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate database: %v", err)
	}

	return db, nil
}
