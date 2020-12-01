// Package db defines database models for Result data.
package db

// Result is the database model of a Result.
type Result struct {
	Parent string `gorm:"primaryKey;index:results_by_name,priority:1"`
	ID     string `gorm:"primaryKey"`
	Name   string `gorm:"index:results_by_name,priority:2"`
}

// Record is the database model of a Record
type Record struct {
	Parent     string `gorm:"primaryKey;index:records_by_name,priority:1"`
	ResultID   string `gorm:"primaryKey"`
	ID         string `gorm:"primaryKey"`
	ResultName string `gorm:"index:records_by_name,priority:2"`
	Name       string `gorm:"index:records_by_name,priority:3"`
	Data       []byte
}
