/*
 * File: annotations.go
 * Project: common
 * File Created: Tuesday, 16th August 2022 9:44:47 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package train

import (
	"context"
	"math/rand"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"

	common "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common"
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
)

// Annotations is a struct for modeling required info retrieved from the annotation DB
type Annotations []*Annotation
type Annotation struct {
	ID        primitive.ObjectID        `bson:"_id"`
	TagIDs    []string                  `bson:"tagids"`
	ContentID string                    `bson:"contentid"`
	Metadata  models.AnnotationMetadata `bson:"metadata,omitempty"`
}

func (a *Annotations) Length() int {
	return len(*a)
}

// FetchContent is a method for retrieving annotations
func FetchAnnotations(dataset *models.Dataset, platform *platform.Platform, db *db.DB) (*Annotations, error) {
	options := options.FindOptions{Projection: bson.M{"_id": 1, "tagids": 1, "contentid": 1, "metadata": 1}, AllowDiskUse: common.Ptr(true)}
	cursor, err := platform.AnnotationDB.FindDatasetAnnotations(db, dataset.UserID, dataset.ID.Hex(), &options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var a Annotations
	if err = cursor.All(context.TODO(), &a); err != nil {
		return nil, err
	}

	return &a, nil
}

// Shuffle is a method for randomly shuffling annotations
func (a *Annotations) Shuffle() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	rand.Shuffle(len(*a), func(i, j int) { (*a)[i], (*a)[j] = (*a)[j], (*a)[i] })
}

func (a *Annotations) Split(splits SplitCounts) (train Annotations, validation Annotations, test Annotations) {
	return (*a)[0:splits.TrainCount], (*a)[splits.TrainCount : splits.TrainCount+splits.ValidationCount], (*a)[splits.TrainCount+splits.ValidationCount : splits.TrainCount+splits.ValidationCount+splits.TestCount]
}
