/*
 * File: platform.go
 * Project: platform
 * File Created: Monday, 2nd August 2021 7:35:30 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package platform

import db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"

// Enforce DB Interfaces
var (
	_ AnnotationDB = (*Annotation)(nil)
	_ ContentDB    = (*Content)(nil)
	_ DatasetDB    = (*Dataset)(nil)
	_ ExportDB     = (*Export)(nil)
	_ ModelDB      = (*Model)(nil)
	_ TagDB        = (*Tag)(nil)
	_ UserDB       = (*User)(nil)
	_ ProjectDB    = (*Project)(nil)
	_ PredictionDB = (*Prediction)(nil)
)

type Platform struct {
	ContentDB    *Content
	TagDB        *Tag
	UserDB       *User
	ModelDB      *Model
	ProjectDB    *Project
	ExportDB     *Export
	DatasetDB    *Dataset
	AnnotationDB *Annotation
	PredictionDB *Prediction
}

type Configuration struct {
	URL     string `mapstructure:"url,omitempty" yaml:"url,omitempty"`
	Timeout int    `mapstructure:"timeout_seconds,omitempty" yaml:"timeout_seconds,omitempty"`
}

func NewPlatform() *Platform {
	return &Platform{
		ContentDB:    NewContent(),
		TagDB:        NewTag(),
		UserDB:       NewUser(),
		ModelDB:      NewModel(),
		ProjectDB:    NewProject(),
		ExportDB:     NewExport(),
		DatasetDB:    NewDataset(),
		AnnotationDB: NewAnnotation(),
		PredictionDB: NewPrediction(),
	}
}

func (p *Platform) Indices() []func(*db.DB) error {
	return []func(*db.DB) error{
		p.UserDB.Index,
		p.TagDB.Index,
		p.AnnotationDB.Index,
		p.ContentDB.Index,
		p.DatasetDB.Index,
		p.ExportDB.Index,
		p.ModelDB.Index,
		p.PredictionDB.Index,
	}
}
