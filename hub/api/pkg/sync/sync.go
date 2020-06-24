package sync

import (
	"context"
	"fmt"
	"strings"

	"github.com/tektoncd/hub/api/pkg/app"
	"github.com/tektoncd/hub/api/pkg/db/model"
	"github.com/tektoncd/hub/api/pkg/git"
	gitclient "github.com/tektoncd/hub/api/pkg/git"
	"go.uber.org/zap"
)

type SyncService struct {
	app       app.Config
	log       *zap.SugaredLogger
	clonePath string
}

func New(app app.Config, clonePath string) *SyncService {
	return &SyncService{
		app:       app,
		log:       app.Logger().With("service", "sync"),
		clonePath: clonePath,
	}
}

func (s *SyncService) Init() error {
	log := s.log.With("action", "init")
	db := s.app.DB().Unscoped()

	count := 0
	db.Model(&model.SyncJob{}).Count(&count)
	log.Info("job count: ", count)

	db.Where("status <> ?", model.Queued).Delete(model.SyncJob{})

	db.Model(&model.SyncJob{}).Count(&count)
	log.Info("job count: ", count)
	return nil
}

func (s *SyncService) Sync(context context.Context) error {
	log := s.log.With("action", "sync")
	db := s.app.DB()

	count := 0
	db.Model(&model.SyncJob{}).Count(&count)
	log.Info("job count: ", count)

	job := model.SyncJob{}
	if err := db.Where("status = ?", "queued").First(&job).Error; err != nil {
		return err
	}

	log.With("url", job.CatalogID, "status", job.Status).Info("Found job")
	if job.IsRunning() {
		log.Info("Already running")
		return nil
	}

	job.SetState(model.Running)
	db.Model(&job).Updates(job)
	defer db.Unscoped().Where(&model.SyncJob{Status: "done"}).Delete(&job)

	catalog := model.Catalog{}
	db.Model(job).Related(&catalog)
	fetchSpec := gitclient.FetchSpec{
		URL:      catalog.URL,
		Revision: catalog.Revision,
		Path:     s.clonePath,
	}

	git := gitclient.New(s.app.Logger())
	repo, err := git.Fetch(fetchSpec)
	if err != nil {
		// TODO: handle /return it
		s.log.Error(err, "clone failed")
		return nil
	}

	s.log.Infof("Repo: %s | HEAD: %s", repo.Path, repo.Head())

	res, err := repo.ParseTektonResources()
	if err != nil {
		// TODO(sthaha): handle updation failure better
		s.log.Error(err)

		job.SetState(model.Queued)
		db.Model(&job).Updates(job)

		return err
	}

	if err := s.updateResources(job, repo, res); err != nil {
		// TODO(sthaha): handle updation failure better
		s.log.Error(err)

		job.SetState(model.Queued)
		db.Model(&job).Updates(job)

		return err
	}

	job.SetState(model.Done)
	db.Model(&job).Updates(job)
	return nil

}

func (s *SyncService) updateResources(job model.SyncJob, repo git.Repo, res []gitclient.TektonResource) error {
	log := s.log.With("action", "updatedb")

	txn := s.app.DB().Begin()

	catalog := model.Catalog{}
	txn.Model(&job).Related(&catalog)

	catalog.SHA = repo.Head()

	others := model.Category{}
	txn.Model(&model.Category{}).Where(&model.Category{Name: "Others"}).First(&others)

	for _, r := range res {

		s.log.Infof("Res: %s | Name: %s ", r.Kind, r.Name)
		if len(r.Versions) == 0 {
			s.log.Infof("      >>> Res: %s | Name: %s has no versions - skipping ", r.Kind, r.Name)
			continue
		}

		dbRes := model.Resource{
			Name:      r.Name,
			Type:      r.Kind,
			CatalogID: catalog.ID,
		}

		txn.Model(&model.Resource{}).Where(&dbRes).FirstOrCreate(&dbRes)
		txn.Save(&dbRes)

		log.Info("Resource ID: ", dbRes.ID)

		for _, v := range r.Versions {
			ver := &model.ResourceVersion{
				Version:    v.Version,
				ResourceID: dbRes.ID,
			}

			txn.Model(&model.ResourceVersion{}).Where(&model.ResourceVersion{ResourceID: dbRes.ID, Version: v.Version}).FirstOrCreate(&ver)

			ver.DisplayName = v.DisplayName
			ver.Description = v.Description
			// TODO(sthaha): use gh client to get the path?
			// this heuristic works fine so far
			ver.URL = fmt.Sprintf("%s/tree/%s/%s", catalog.URL, catalog.Revision, v.Path)

			txn.Save(&ver)
			s.log.Infof("      >>> Version: %d -> %s | Path: %s | tags: %s", ver.ID, ver.Version, v.Path, strings.Join(v.Tags, ", "))

			for _, t := range v.Tags {
				tag := model.Tag{Name: t, CategoryID: others.ID}

				txn.Model(&model.Tag{}).Where(&model.Tag{Name: t}).FirstOrCreate(&tag)

				resTag := model.ResourceTag{ResourceID: dbRes.ID, TagID: tag.ID}
				txn.Model(&model.ResourceTag{}).Where(&resTag).FirstOrCreate(&resTag)
				s.log.Infof("      >>> Resource: %d: %s | tag: %s (%d)", dbRes.ID, dbRes.Name, tag.Name, tag.ID)
			}
		}

	}

	txn.Save(&catalog)
	txn.Commit()
	return nil
}
