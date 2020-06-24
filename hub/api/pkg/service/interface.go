package service

import (
	"github.com/jinzhu/gorm"
	"github.com/tektoncd/hub/api/pkg/app"
	"go.uber.org/zap"
)

type Service interface {
	Resource() *Resource
	Category() *Category
	Rating() *Rating
	User() *User
}

type ServiceImpl struct {
	app app.Config
	log *zap.SugaredLogger
	db  *gorm.DB
	gh  *app.GitHub
}

func New(app app.Config) *ServiceImpl {
	return &ServiceImpl{
		app: app,
		log: app.Logger().With("name", "db"),
		db:  app.DB(),
		gh:  app.GitHub(),
	}
}

func (s *ServiceImpl) Resource() *Resource {
	return &Resource{s.db, s.log}
}

func (s *ServiceImpl) Category() *Category {
	return &Category{s.db, s.log}
}

func (s *ServiceImpl) Rating() *Rating {
	return &Rating{s.db, s.log}
}

func (s *ServiceImpl) User() *User {
	return &User{s.db, s.log, s.gh}
}
