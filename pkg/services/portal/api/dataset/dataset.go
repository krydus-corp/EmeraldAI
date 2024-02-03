/*
 * File: dataset.go
 * Project: dataset
 * File Created: Monday, 16th May 2022 11:28:56 am
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
// Package dataset contains dataset application services
package dataset

import (
	"github.com/labstack/echo/v4"

	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
)

func (d Dataset) View(c echo.Context, userid, datasetid string) (*models.Dataset, error) {
	dataset, err := d.platform.DatasetDB.View(d.db, userid, datasetid)
	if err != nil {
		return nil, err
	}

	return dataset, nil
}

// Update dataset information
func (d Dataset) Update(c echo.Context, dataset models.Dataset) (*models.Dataset, error) {

	if err := d.platform.DatasetDB.Update(d.db, &dataset); err != nil {
		return nil, err
	}

	return d.platform.DatasetDB.View(d.db, dataset.UserID, dataset.ID.Hex())
}
