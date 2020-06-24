package service

import (
	"errors"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/tektoncd/hub/api/pkg/db/model"
	"go.uber.org/zap"
)

type Resource struct {
	db  *gorm.DB
	log *zap.SugaredLogger
}

// ResourceDetail abstracts necessary fields for UI
type ResourceDetail struct {
	ID            uint      `json:"id"`
	Name          string    `json:"name"`
	DisplayName   string    `json:"displayName"`
	Catalog       Catalog   `json:"catalog"`
	Type          string    `json:"type"`
	Description   string    `json:"description"`
	LatestVersion string    `json:"latestVersion"`
	Tags          []Tag     `json:"tags"`
	Rating        float64   `json:"rating"`
	LastUpdatedAt time.Time `json:"lastUpdatedAt"`
}

// ResourceVersionDetail abstracts necessary fields for UI
type ResourceVersionDetail struct {
	Version     string `json:"version"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	WebURL      string `json:"webUrl"`
	RawURL      string `json:"rawUrl"`
}

type Catalog struct {
	ID   uint   `json:"id"`
	Type string `json:"type"`
}

type Version struct {
	ID      uint   `json:"id"`
	Version string `json:"version"`
}

type Tag struct {
	ID  uint   `json:"id"`
	Tag string `json:"name"`
}

type Filter struct {
	Limit int
}

// Init Convert Resource object to ResourceDetails
func (d *ResourceDetail) Init(r *model.Resource) {
	d.ID = r.ID
	d.Name = r.Name
	d.Type = r.Type
	d.Rating = r.Rating

	d.Tags = make([]Tag, len(r.Tags))
	for i, t := range r.Tags {
		d.Tags[i].ID = t.ID
		d.Tags[i].Tag = t.Name
	}

	d.Catalog.ID = r.Catalog.ID
	d.Catalog.Type = r.Catalog.Type

	// TODO: Sort the Version's array on basis of Version or Updated_At
	latestVersion := r.Versions[len(r.Versions)-1]
	d.DisplayName = latestVersion.DisplayName
	d.LatestVersion = latestVersion.Version
	d.Description = latestVersion.Description
	d.LastUpdatedAt = latestVersion.UpdatedAt
}

// All Resources
func (r *Resource) All(filter Filter) ([]ResourceDetail, error) {

	var all []*model.Resource
	if err := r.db.Order("rating desc, name").Limit(filter.Limit).
		Preload("Catalog").
		Preload("Versions", func(db *gorm.DB) *gorm.DB {
			return db.Order("resource_versions.id ASC")
		}).
		Preload("Tags").
		Find(&all).Error; err != nil {
		return []ResourceDetail{}, errors.New("Failed to fetch Resources")
	}

	ret := make([]ResourceDetail, len(all))
	for i, r := range all {
		ret[i].Init(r)
	}
	return ret, nil
}

// Init converts ResourceVersion Object to ResourceVersionDetail
func (d *ResourceVersionDetail) Init(r *model.ResourceVersion) {
	d.Version = r.Version
	d.Description = r.Description
	d.DisplayName = r.DisplayName
	d.WebURL = r.URL
	replaceStrings := strings.NewReplacer("github.com", "raw.githubusercontent.com",
		"/tree/", "/")
	d.RawURL = replaceStrings.Replace(r.URL)
}

// AllVersions Get all versions of a Resource
func (r *Resource) AllVersions(resourceID uint) ([]ResourceVersionDetail, error) {

	var all []*model.ResourceVersion
	if err := r.db.Order("id").Where("resource_id = ?", resourceID).Find(&all).Error; err != nil {
		return []ResourceVersionDetail{}, errors.New("Failed to fetch Resources")
	}

	ret := make([]ResourceVersionDetail, len(all))
	for i, r := range all {
		ret[i].Init(r)
	}

	return ret, nil
}
