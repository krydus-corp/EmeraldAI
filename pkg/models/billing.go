/*
 * File: billing.go
 * Project: models
 * File Created: Sunday, 11th December 2022 8:23:04 am
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package models

import (
	"fmt"
	"time"
)

// High level billing model
type Billing struct {
	Usage map[string][]Usage `json:"-" bson:"usage"`
}

func NewBilling() Billing {
	billing := Billing{
		Usage: make(map[string][]Usage),
	}

	return billing
}

// UsageType describes what type the usage is associated with
type UsageType string

const (
	UsageTypeTrain    UsageType = "TRAIN"
	UsageTypeUpload   UsageType = "UPLOAD"
	UsageTypeEndpoint UsageType = "ENDPOINT"
)

// BillingMetric describes how the billing should be computed (hourly vs secondly rate)
type BillingMetric string

const (
	BillingMetricSecond BillingMetric = "SECOND"
	BillingMetricHour   BillingMetric = "HOUR"
	BillingMetricGB     BillingMetric = "GB"
)

// Usage describes a single usage entry
type Usage struct {
	Time          string                 `json:"-" bson:"time"`
	Type          UsageType              `json:"-" bson:"type"`
	BillingMetric BillingMetric          `json:"-" bson:"billing_metric"`
	BillableValue float64                `json:"-" bson:"billing_value"`
	Metadata      map[string]interface{} `json:"-" bson:"metadata"`
}

func usageKey() string {
	year, month, _ := time.Now().Date()
	return fmt.Sprintf("%s_%d", month.String(), year)
}

// AddUsage is a method for adding a usage entry to the billing record
func (b *Billing) AddUsage(usage Usage) {
	key := usageKey()

	if b.Usage == nil {
		b.Usage = make(map[string][]Usage)
	}

	b.Usage[key] = append(b.Usage[key], usage)
}
