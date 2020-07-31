package model

import (
	"github.com/jinzhu/gorm"
	"go.uber.org/zap"
	"gopkg.in/gormigrate.v1"
)

func addModifiedAtToResourceVersion(log *zap.SugaredLogger) *gormigrate.Migration {

	return &gormigrate.Migration{
		ID: "20200731_addmodified_at_to_resourceversion",
		Migrate: func(db *gorm.DB) error {
			return db.AutoMigrate(&ResourceVersion{}).Error
		},
	}
}
