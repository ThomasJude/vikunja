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
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskStructuredRecurrenceCompletion(t *testing.T) {
	t.Run("scheduled month day", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:     TaskRecurrenceFrequencyMonth,
			Interval:      1,
			Basis:         TaskRecurrenceBasisSchedule,
			ByMonthDay:    31,
			MissingPolicy: TaskRecurrenceMissingPolicyLastValid,
		}

		oldTask := &Task{
			DueDate:    time.Date(2025, time.January, 31, 9, 0, 0, 0, time.UTC),
			StartDate:  time.Date(2025, time.January, 30, 9, 0, 0, 0, time.UTC),
			EndDate:    time.Date(2025, time.January, 31, 10, 0, 0, 0, time.UTC),
			Recurrence: rule,
			Reminders: []*TaskReminder{
				{Reminder: time.Date(2025, time.January, 31, 8, 0, 0, 0, time.UTC)},
			},
		}
		newTask := &Task{Done: true}

		err := setTaskDatesRecurrence(
			oldTask,
			newTask,
			time.Date(2025, time.February, 1, 12, 0, 0, 0, time.UTC),
		)

		require.NoError(t, err)
		assert.Equal(t, time.Date(2025, time.February, 28, 9, 0, 0, 0, time.UTC), newTask.DueDate)
		assert.Equal(t, time.Date(2025, time.February, 27, 9, 0, 0, 0, time.UTC), newTask.StartDate)
		assert.Equal(t, time.Date(2025, time.February, 28, 10, 0, 0, 0, time.UTC), newTask.EndDate)
		require.Len(t, newTask.Reminders, 1)
		assert.Equal(t, time.Date(2025, time.February, 28, 8, 0, 0, 0, time.UTC), newTask.Reminders[0].Reminder)
		assert.False(t, newTask.Done)
	})

	t.Run("overdue schedule skips missed occurrences", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:     TaskRecurrenceFrequencyMonth,
			Interval:      1,
			Basis:         TaskRecurrenceBasisSchedule,
			ByMonthDay:    31,
			MissingPolicy: TaskRecurrenceMissingPolicyLastValid,
		}

		oldTask := &Task{
			DueDate:    time.Date(2026, time.January, 31, 9, 0, 0, 0, time.UTC),
			Recurrence: rule,
		}
		newTask := &Task{Done: true}

		err := setTaskDatesRecurrence(
			oldTask,
			newTask,
			time.Date(2026, time.September, 16, 12, 0, 0, 0, time.UTC),
		)

		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, time.September, 30, 9, 0, 0, 0, time.UTC), newTask.DueDate)
		assert.False(t, newTask.Done)
	})

	t.Run("completion basis preserves scheduled time", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyMonth,
			Interval:   1,
			Basis:      TaskRecurrenceBasisCompletion,
			ByMonthDay: 15,
		}

		oldTask := &Task{
			DueDate:    time.Date(2026, time.January, 10, 9, 30, 0, 0, time.UTC),
			Recurrence: rule,
		}
		newTask := &Task{Done: true}

		err := setTaskDatesRecurrence(
			oldTask,
			newTask,
			time.Date(2026, time.January, 20, 16, 45, 0, 0, time.UTC),
		)

		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, time.February, 15, 9, 30, 0, 0, time.UTC), newTask.DueDate)
		assert.False(t, newTask.Done)
	})

	t.Run("next period fifth weekday preserves cadence", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:     TaskRecurrenceFrequencyMonth,
			Interval:      1,
			Basis:         TaskRecurrenceBasisSchedule,
			ByWeekdays:    1 << uint(time.Monday),
			BySetPos:      5,
			MissingPolicy: TaskRecurrenceMissingPolicyNextPeriod,
		}

		next, err := nextTaskRecurrenceAfter(
			rule,
			time.Date(2026, time.March, 2, 9, 0, 0, 0, time.UTC),
			time.Date(2026, time.March, 3, 12, 0, 0, 0, time.UTC),
		)

		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, time.March, 30, 9, 0, 0, 0, time.UTC), next)
	})

	t.Run("no scheduled date creates next due date", func(t *testing.T) {
		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyMonth,
			Interval:   1,
			Basis:      TaskRecurrenceBasisCompletion,
			ByMonthDay: 15,
		}

		oldTask := &Task{Recurrence: rule}
		newTask := &Task{Done: true}

		err := setTaskDatesRecurrence(
			oldTask,
			newTask,
			time.Date(2026, time.January, 20, 16, 45, 0, 0, time.UTC),
		)

		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, time.February, 15, 16, 45, 0, 0, time.UTC), newTask.DueDate)
		assert.False(t, newTask.Done)
	})

	t.Run("partial done update loads stored recurrence", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)

		u := &user.User{ID: 1}
		now := time.Now().UTC()
		base := now.AddDate(0, 2, 0)
		due := time.Date(base.Year(), base.Month(), 15, 9, 0, 0, 0, time.UTC)

		rule := &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyMonth,
			Interval:   1,
			Basis:      TaskRecurrenceBasisSchedule,
			ByMonthDay: 15,
		}

		s := db.NewSession()
		task := &Task{
			Title:      "structured recurring completion",
			ProjectID:  1,
			DueDate:    due,
			Recurrence: rule,
		}
		require.NoError(t, task.Create(s, u))
		require.NoError(t, s.Commit())
		s.Close()

		expected, err := nextTaskRecurrenceAfter(rule, due, now)
		require.NoError(t, err)

		s2 := db.NewSession()
		defer s2.Close()

		update := &Task{
			ID:   task.ID,
			Done: true,
		}

		err = update.updateSingleTask(s2, u, []string{"done"})
		require.NoError(t, err)
		require.NoError(t, s2.Commit())

		assert.False(t, update.Done)
		require.NotNil(t, update.Recurrence)
		assert.True(
			t,
			expected.Equal(update.DueDate),
			"expected %s, got %s",
			expected,
			update.DueDate,
		)

		db.AssertExists(t, "task_recurrences", map[string]interface{}{
			"task_id": task.ID,
		}, false)
	})
}

func TestTaskIsRepeatingWithStructuredRecurrence(t *testing.T) {
	task := &Task{
		Recurrence: &TaskRecurrence{
			Frequency:  TaskRecurrenceFrequencyMonth,
			Interval:   1,
			ByMonthDay: 1,
		},
	}

	assert.True(t, task.isRepeating())
}
