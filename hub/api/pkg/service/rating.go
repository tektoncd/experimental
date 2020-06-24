package service

import (
	"errors"
	"math"

	"github.com/jinzhu/gorm"
	"github.com/tektoncd/hub/api/pkg/db/model"
	"go.uber.org/zap"
)

// Rating Service
type Rating struct {
	db  *gorm.DB
	log *zap.SugaredLogger
}

type UserResource struct {
	UserID     int
	ResourceID int
}

type UpdateRatingDetails struct {
	UserID         uint `json:"userId"`
	ResourceID     uint `json:"resourceId"`
	ResourceRating uint `json:"rating"`
}

type RatingDetails struct {
	ResourceRating uint `json:"rating"`
}

type ResourceAverageRating struct {
	Rating float64 `json:"avgRating"`
}

// GetResourceRating returns user's rating of a resource
func (r *Rating) GetResourceRating(ur UserResource) (RatingDetails, error) {

	rating := &model.UserResourceRating{}
	if r.db.Where("user_id = ? AND resource_id = ?", ur.UserID, ur.ResourceID).Find(&rating).RecordNotFound() {
		return RatingDetails{
			ResourceRating: 0,
		}, nil
	}
	var resRating RatingDetails
	resRating.ResourceRating = rating.Rating

	return resRating, nil
}

// UpdateResourceRating update user's rating of a resource and resource's average rating
func (r *Rating) UpdateResourceRating(rd UpdateRatingDetails) (ResourceAverageRating, error) {

	// TODO(shivam): need to use the err; perhaps return it?
	if err := r.db.Where("user_id = ? AND resource_id = ?", rd.UserID, rd.ResourceID).
		Assign(&model.UserResourceRating{Rating: rd.ResourceRating}).
		FirstOrCreate(&model.UserResourceRating{
			UserID:     rd.UserID,
			ResourceID: rd.ResourceID,
			Rating:     rd.ResourceRating,
		}).Error; err != nil {
		return ResourceAverageRating{}, errors.New("Failed to update user's rating")
	}

	//TODO: Add goroutine to update resource's average rating
	var avg float64
	r.db.Model(&model.UserResourceRating{}).Where("resource_id = ?", rd.ResourceID).
		Select("avg(rating)").Row().Scan(&avg)

	avg = math.Round(avg*10) / 10

	if err := r.db.Model(&model.Resource{}).Where("id = ?", rd.ResourceID).
		Update("rating", avg).Error; err != nil {
		return ResourceAverageRating{}, errors.New("Failed to update resource's avg rating")
	}

	return ResourceAverageRating{Rating: avg}, nil
}
