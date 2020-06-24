package model

import (
	"github.com/jinzhu/gorm"
)

type JobState int

const (
	// represents Queued state
	Queued JobState = iota
	Running
	Done
	Error
)

func (s JobState) String() string {
	return [...]string{"queued", "running", "done", "error"}[s]
}

type SyncJob struct {
	gorm.Model
	Catalog   Catalog
	CatalogID uint
	Status    string
}

func (j *SyncJob) SetState(s JobState) {
	j.Status = s.String()
}

func (j *SyncJob) IsRunning() bool {
	return j.Status == Running.String()
}
