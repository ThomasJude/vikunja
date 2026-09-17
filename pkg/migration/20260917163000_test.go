// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package migration

import (
	"strings"
	"testing"
	"time"

	"code.vikunja.io/api/pkg/db"

	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

func recurrenceIndexByColumns20260917163000(
	t *testing.T,
	x *xorm.Engine,
	tableName string,
	columns ...string,
) *schemas.Index {
	t.Helper()

	tables, err := x.DBMetas()
	require.NoError(t, err)

	for _, table := range tables {
		if table.Name != tableName {
			continue
		}

		for _, index := range table.Indexes {
			if len(index.Cols) != len(columns) {
				continue
			}

			matches := true
			for i := range columns {
				if !strings.EqualFold(index.Cols[i], columns[i]) {
					matches = false
					break
				}
			}

			if matches {
				return index
			}
		}
	}

	return nil
}

func TestCreateTaskRecurrenceSeriesTables20260917163000(t *testing.T) {
	x, err := db.CreateTestEngine()
	require.NoError(t, err)

	_, _ = x.Exec("DROP TABLE IF EXISTS task_recurrence_occurrences")
	_, _ = x.Exec("DROP TABLE IF EXISTS task_recurrence_series")

	t.Cleanup(func() {
		_, _ = x.Exec("DROP TABLE IF EXISTS task_recurrence_occurrences")
		_, _ = x.Exec("DROP TABLE IF EXISTS task_recurrence_series")
	})

	require.NoError(t, createTaskRecurrenceSeriesTables20260917163000(x))

	// Migration must also be safe to run again.
	require.NoError(t, createTaskRecurrenceSeriesTables20260917163000(x))

	require.NotNil(
		t,
		recurrenceIndexByColumns20260917163000(
			t,
			x,
			"task_recurrence_series",
			"root_task_id",
		),
	)

	require.NotNil(
		t,
		recurrenceIndexByColumns20260917163000(
			t,
			x,
			"task_recurrence_series",
			"project_id",
		),
	)

	require.NotNil(
		t,
		recurrenceIndexByColumns20260917163000(
			t,
			x,
			"task_recurrence_series",
			"created_by_id",
		),
	)

	require.NotNil(
		t,
		recurrenceIndexByColumns20260917163000(
			t,
			x,
			"task_recurrence_occurrences",
			"series_id",
		),
	)

	taskUnique := recurrenceIndexByColumns20260917163000(
		t,
		x,
		"task_recurrence_occurrences",
		"task_id",
	)
	require.NotNil(t, taskUnique)
	require.Equal(t, schemas.UniqueType, taskUnique.Type)

	seriesSequenceUnique := recurrenceIndexByColumns20260917163000(
		t,
		x,
		"task_recurrence_occurrences",
		"series_id",
		"sequence",
	)
	require.NotNil(t, seriesSequenceUnique)
	require.Equal(t, schemas.UniqueType, seriesSequenceUnique.Type)

	now := time.Date(2026, time.September, 17, 9, 0, 0, 0, time.UTC)

	result, err := x.Exec(`
		INSERT INTO task_recurrence_series (
			root_task_id,
			project_id,
			created_by_id,
			frequency,
			interval,
			basis,
			by_weekdays,
			by_month,
			by_month_day,
			by_set_pos,
			missing_policy,
			start_date,
			end_type,
			end_after_occurrences,
			create_before_days,
			weekend_policy,
			missed_policy,
			paused,
			created,
			updated
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		1,
		1,
		1,
		3,
		1,
		0,
		0,
		0,
		15,
		0,
		0,
		now,
		0,
		0,
		0,
		0,
		0,
		false,
		now,
		now,
	)
	require.NoError(t, err)

	seriesID, err := result.LastInsertId()
	require.NoError(t, err)

	insertOccurrence := func(taskID int64, sequence int) error {
		_, err := x.Exec(`
			INSERT INTO task_recurrence_occurrences (
				series_id,
				task_id,
				sequence,
				scheduled_due_date,
				due_date,
				is_exception,
				created,
				updated
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`,
			seriesID,
			taskID,
			sequence,
			now,
			now,
			false,
			now,
			now,
		)
		return err
	}

	require.NoError(t, insertOccurrence(1001, 1))

	err = insertOccurrence(1001, 2)
	require.Error(t, err, "task_id must be unique")

	err = insertOccurrence(1002, 1)
	require.Error(t, err, "series_id + sequence must be unique")
}
