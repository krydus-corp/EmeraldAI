package models

import (
	"crypto/md5"
	"encoding/hex"
	"time"
)

// Content represents content domain model
//
// swagger:model Content
type Content struct {
	// ID of the Content
	//
	ID string `json:"id" bson:"_id"`
	// UserID associated with Content
	//
	UserID string `json:"userid" bson:"userid"`
	// Project IDs associated with Content
	//
	Projects []string `json:"-" bson:"projects"`
	// Project S3 bucket
	//
	StoredDir string `json:"-" bson:"stored_dir"`
	// Project S3 path
	//
	StoredPath string `json:"-" bson:"stored_path"`
	// Name of Content
	//
	Name string `json:"name" bson:"name"`
	// Content hash
	//
	Hash string `json:"hash" bson:"hash"`
	// Mimetype of content
	//
	ContentType string `json:"content_type" bson:"content_type"`
	// Content size in bytes
	//
	Size int `json:"size" bson:"size"`
	// Content height
	//
	Height int `json:"height" bson:"height"`
	// Content width
	//
	Width int `json:"width" bson:"width"`
	// Content b64 string
	//
	Base64Image string `json:"b64_image" bson:"b64_image"`
	// Annotation associated with content
	// This is never populated. It is used in Mongo aggregation pipelines when joining the content
	// collection with the annotation collection so that we can locate all content with an annotation
	// and return the annotation embedded in the Content model.
	Annotation []Annotation `json:"annotation" bson:"annotation"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

func NewContent(_type, dir, path, hash, userid, projectid, name, b64Image string, size, height, width int) *Content {

	return &Content{
		ID:          NewContentID(hash, userid),
		StoredDir:   dir,
		StoredPath:  path,
		UserID:      userid,
		Projects:    []string{projectid},
		Name:        name,
		Hash:        hash,
		ContentType: _type,
		Size:        size,
		Height:      height,
		Width:       width,
		Base64Image: b64Image,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func NewContentID(hash, userid string) string {
	hmd5 := md5.Sum([]byte(userid + hash))
	return hex.EncodeToString(hmd5[:])
}
