package db

import (
	"time"

	"gorm.io/gorm"
)

type PhoneNumber struct {
	gorm.Model
	PhoneNumber string `gorm:"unique"`
	DisplayName string
}

type CallLog struct {
	gorm.Model
	PhoneNumberID     uint
	CalledPhoneNumber string
	Timestamp         time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}
