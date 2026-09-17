// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package migration

import (
	"fmt"
	"strings"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260917233000",
		Description: "Add recurrence series materializer scan index",
		Migrate: func(tx *xorm.Engine) error {
			query := `
				CREATE INDEX IF NOT EXISTS IDX_task_recurrence_series_materializer
				ON task_recurrence_series (paused, id)
			`

			if tx.Dialect().URI().DBType == schemas.MYSQL {
				query = `
					CREATE INDEX IDX_task_recurrence_series_materializer
					ON task_recurrence_series (paused, id)
				`
			}

			_, err := tx.Exec(query)
			if err != nil &&
				tx.Dialect().URI().DBType == schemas.MYSQL &&
				strings.Contains(err.Error(), "Duplicate key name") {
				return nil
			}

			if err != nil {
				return fmt.Errorf(
					"could not create recurrence materializer index: %w",
					err,
				)
			}

			return nil
		},
		Rollback: func(tx *xorm.Engine) error {
			return nil
		},
	})
}
