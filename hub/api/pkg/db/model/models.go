package model

import (
	"github.com/jinzhu/gorm"
)

type (

	//Category model represents categories which associated with group of tags
	Category struct {
		gorm.Model
		Name string `gorm:"size:100;not null;unique"`
		Tags []Tag
	}

	// Tag model represents tags associated with a resource
	Tag struct {
		gorm.Model
		Name       string `gorm:"size:100;not null;unique"`
		Category   Category
		CategoryID uint
		Resources  []*Resource `gorm:"many2many:resource_tags;"`
	}

	//Catalog model represents origin repo to which resource belongs to
	Catalog struct {
		gorm.Model
		Name       string
		Type       string
		URL        string
		Revision   string
		Owner      string
		ContextDir string
		SHA        string
		Resources  []Resource
	}

	// Resource represents model which describe resources from a catalog
	Resource struct {
		gorm.Model
		Name      string
		Type      string
		Rating    float64
		Catalog   Catalog
		CatalogID uint
		Versions  []ResourceVersion
		Tags      []*Tag `gorm:"many2many:resource_tags;"`
	}

	// ResourceVersion represents diffrent versions of a Resource
	ResourceVersion struct {
		gorm.Model
		Version     string
		DisplayName string
		Description string
		URL         string
		Resource    Resource
		ResourceID  uint
	}

	// ResourceTag represent struct for resource_tag, needed for creating foreign key
	ResourceTag struct {
		ResourceID uint
		TagID      uint
	}

	// User model represents user details
	User struct {
		gorm.Model
		UserName string `gorm:"not null;unique"`
		Name     string
		Email    string
		Token    string
	}

	// UserResourceRating represents User's rating of a resource
	UserResourceRating struct {
		gorm.Model
		UserID     uint
		User       User
		Resource   Resource
		ResourceID uint
		Rating     uint
	}
)
