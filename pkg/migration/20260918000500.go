// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package migration

import (
	"time"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

type taskRecurrenceOccurrence20260918000500 struct {
	ID              int64     `xorm:"bigint autoincr not null unique pk"`
	ExceptionAnchor time.Time `xorm:"datetime null"`
}

func (taskRecurrenceOccurrence20260918000500) TableName() string {
	return "task_recurrence_occurrences"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260918000500",
		Description: "Add recurrence occurrence exception anchor",
		Migrate: func(tx *xorm.Engine) error {
			return partialSync(
				tx,
				taskRecurrenceOccurrence20260918000500{},
			)
		},
		Rollback: func(tx *xorm.Engine) error {
			return nil
		},
	})
}
