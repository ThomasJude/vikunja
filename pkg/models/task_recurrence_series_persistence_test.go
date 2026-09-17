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

package models

import (
	"testing"
	"time"

	"code.vikunja.io/api/pkg/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskRecurrenceSeriesPersistence(t *testing.T) {
	db.LoadAndAssertFixtures(t)

	s := db.NewSession()
	defer s.Close()

	now := time.Date(
		2026,
		time.September,
		17,
		9,
		0,
		0,
		0,
		time.UTC,
	)

	series := &TaskRecurrenceSeries{
		RootTaskID:  1,
		ProjectID:   1,
		CreatedByID: 1,

		Frequency: TaskRecurrenceFrequencyMonth,
		Interval:  3,
		Basis:     TaskRecurrenceBasisSchedule,

		ByMonthDay: 1,

		StartDate: now,

		EndType: TaskRecurrenceEndNever,

		CreateBeforeDays: 5,

		WeekendPolicy: TaskRecurrenceWeekendKeep,
		MissedPolicy:  TaskRecurrenceMissedNextFuture,
	}

	t.Run("create and read series", func(t *testing.T) {
		require.NoError(t, createTaskRecurrenceSeries(s, series))
		require.NotZero(t, series.ID)

		stored, err := getTaskRecurrenceSeriesByID(s, series.ID)
		require.NoError(t, err)

		assert.Equal(t, series.ID, stored.ID)
		assert.Equal(t, int64(1), stored.RootTaskID)
		assert.Equal(t, int64(1), stored.ProjectID)
		assert.Equal(t, int64(1), stored.CreatedByID)
		assert.Equal(t, TaskRecurrenceFrequencyMonth, stored.Frequency)
		assert.Equal(t, 3, stored.Interval)
		assert.Equal(t, 5, stored.CreateBeforeDays)
	})

	t.Run("lookup by root task", func(t *testing.T) {
		stored, err := getTaskRecurrenceSeriesByRootTaskID(
			s,
			series.RootTaskID,
		)
		require.NoError(t, err)
		require.NotNil(t, stored)

		assert.Equal(t, series.ID, stored.ID)
	})

	t.Run("update series", func(t *testing.T) {
		series.Paused = true
		series.CreateBeforeDays = 7
		series.EndType = TaskRecurrenceEndOccurrences
		series.EndAfterOccurrences = 12

		require.NoError(t, updateTaskRecurrenceSeries(s, series))

		stored, err := getTaskRecurrenceSeriesByID(s, series.ID)
		require.NoError(t, err)

		assert.True(t, stored.Paused)
		assert.Equal(t, 7, stored.CreateBeforeDays)
		assert.Equal(t, TaskRecurrenceEndOccurrences, stored.EndType)
		assert.Equal(t, 12, stored.EndAfterOccurrences)
	})

	t.Run("create and read occurrence", func(t *testing.T) {
		occurrence := &TaskRecurrenceOccurrence{
			SeriesID: series.ID,
			TaskID:   2,
			Sequence: 1,

			ScheduledDueDate: now.AddDate(0, 3, 0),
			DueDate:          now.AddDate(0, 3, 0),
		}

		require.NoError(
			t,
			createTaskRecurrenceOccurrence(s, occurrence),
		)
		require.NotZero(t, occurrence.ID)

		stored, err := getTaskRecurrenceOccurrenceByTaskID(
			s,
			occurrence.TaskID,
		)
		require.NoError(t, err)
		require.NotNil(t, stored)

		assert.Equal(t, occurrence.ID, stored.ID)
		assert.Equal(t, series.ID, stored.SeriesID)
		assert.Equal(t, 1, stored.Sequence)
		assert.True(
			t,
			occurrence.ScheduledDueDate.Equal(stored.ScheduledDueDate),
			"scheduled due date differs: expected %s, got %s",
			occurrence.ScheduledDueDate,
			stored.ScheduledDueDate,
		)
		assert.True(
			t,
			occurrence.DueDate.Equal(stored.DueDate),
			"due date differs: expected %s, got %s",
			occurrence.DueDate,
			stored.DueDate,
		)
	})

	t.Run("occurrence validation", func(t *testing.T) {
		err := createTaskRecurrenceOccurrence(
			s,
			&TaskRecurrenceOccurrence{
				SeriesID: series.ID,
			},
		)

		assert.EqualError(t, err, "task id is required")
	})
}
