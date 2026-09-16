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

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func monthlyRecurrenceForTest(day, interval int) *TaskRecurrence {
	return &TaskRecurrence{
		Frequency:     TaskRecurrenceFrequencyMonth,
		Interval:      interval,
		Basis:         TaskRecurrenceBasisSchedule,
		ByMonthDay:    day,
		MissingPolicy: TaskRecurrenceMissingPolicyLastValid,
	}
}

func TestTaskRecurrencePersistence(t *testing.T) {
	u := &user.User{ID: 1}

	t.Run("create and read", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)

		s := db.NewSession()
		defer s.Close()

		task := &Task{
			Title:      "task with recurrence",
			ProjectID:  1,
			Recurrence: monthlyRecurrenceForTest(31, 1),
		}

		err := task.Create(s, u)
		require.NoError(t, err)
		require.NoError(t, s.Commit())

		db.AssertExists(t, "task_recurrences", map[string]interface{}{
			"task_id":        task.ID,
			"frequency":      TaskRecurrenceFrequencyMonth,
			"interval":       1,
			"by_month_day":   31,
			"missing_policy": TaskRecurrenceMissingPolicyLastValid,
		}, false)

		s2 := db.NewSession()
		defer s2.Close()

		read := &Task{ID: task.ID}
		err = read.ReadOne(s2, u)
		require.NoError(t, err)
		require.NotNil(t, read.Recurrence)

		assert.NotZero(t, read.Recurrence.ID)
		assert.Equal(t, task.ID, read.Recurrence.TaskID)
		assert.Equal(t, TaskRecurrenceFrequencyMonth, read.Recurrence.Frequency)
		assert.Equal(t, 1, read.Recurrence.Interval)
		assert.Equal(t, 31, read.Recurrence.ByMonthDay)
	})

	t.Run("update existing rule", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)

		s := db.NewSession()
		defer s.Close()

		rule := monthlyRecurrenceForTest(15, 1)
		err := saveTaskRecurrence(s, 1, rule)
		require.NoError(t, err)
		originalID := rule.ID

		task := &Task{ID: 1}
		err = task.ReadOne(s, u)
		require.NoError(t, err)
		require.NotNil(t, task.Recurrence)

		task.Recurrence.ByMonthDay = 20
		task.Recurrence.Interval = 3

		err = task.Update(s, u)
		require.NoError(t, err)
		require.NoError(t, s.Commit())

		s2 := db.NewSession()
		defer s2.Close()

		stored := &TaskRecurrence{}
		has, err := s2.Where("task_id = ?", 1).Get(stored)
		require.NoError(t, err)
		require.True(t, has)

		assert.Equal(t, originalID, stored.ID)
		assert.Equal(t, 20, stored.ByMonthDay)
		assert.Equal(t, 3, stored.Interval)

		count, err := s2.Where("task_id = ?", 1).Count(&TaskRecurrence{})
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("partial update preserves rule", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)

		s := db.NewSession()
		defer s.Close()

		err := saveTaskRecurrence(s, 1, monthlyRecurrenceForTest(18, 1))
		require.NoError(t, err)

		task := &Task{
			ID:    1,
			Title: "partial update",
		}
		err = task.updateSingleTask(s, u, []string{"title"})
		require.NoError(t, err)
		require.NoError(t, s.Commit())

		s2 := db.NewSession()
		defer s2.Close()

		stored := &TaskRecurrence{}
		has, err := s2.Where("task_id = ?", 1).Get(stored)
		require.NoError(t, err)
		require.True(t, has)

		assert.Equal(t, 18, stored.ByMonthDay)
	})

	t.Run("full update removes rule", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)

		s := db.NewSession()
		defer s.Close()

		err := saveTaskRecurrence(s, 1, monthlyRecurrenceForTest(12, 1))
		require.NoError(t, err)

		task := &Task{ID: 1}
		err = task.ReadOne(s, u)
		require.NoError(t, err)
		require.NotNil(t, task.Recurrence)

		task.Recurrence = nil

		err = task.Update(s, u)
		require.NoError(t, err)
		require.NoError(t, s.Commit())

		db.AssertMissing(t, "task_recurrences", map[string]interface{}{
			"task_id": 1,
		})
	})

	t.Run("bulk recurrence field", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)

		s := db.NewSession()
		defer s.Close()

		first := &Task{
			Title:     "bulk recurrence one",
			ProjectID: 1,
		}
		require.NoError(t, first.Create(s, u))

		second := &Task{
			Title:     "bulk recurrence two",
			ProjectID: 1,
		}
		require.NoError(t, second.Create(s, u))

		updated, err := updateTasks(
			s,
			u,
			&Task{Recurrence: monthlyRecurrenceForTest(7, 2)},
			[]int64{first.ID, second.ID},
			[]string{"recurrence"},
		)
		require.NoError(t, err)
		require.Len(t, updated, 2)
		require.NoError(t, s.Commit())

		for _, taskID := range []int64{first.ID, second.ID} {
			db.AssertExists(t, "task_recurrences", map[string]interface{}{
				"task_id":      taskID,
				"interval":     2,
				"by_month_day": 7,
			}, false)
		}
	})

	t.Run("soft delete preserves rule", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)

		s := db.NewSession()
		defer s.Close()

		task := &Task{
			Title:      "soft delete recurrence",
			ProjectID:  1,
			Recurrence: monthlyRecurrenceForTest(10, 1),
		}
		require.NoError(t, task.Create(s, u))
		require.NoError(t, s.Commit())

		s2 := db.NewSession()
		defer s2.Close()

		err := (&Task{ID: task.ID}).Delete(s2, u)
		require.NoError(t, err)
		require.NoError(t, s2.Commit())

		db.AssertExists(t, "task_recurrences", map[string]interface{}{
			"task_id": task.ID,
		}, false)
	})

	t.Run("hard delete removes rule", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)

		s := db.NewSession()
		defer s.Close()

		task := &Task{
			Title:      "hard delete recurrence",
			ProjectID:  1,
			Recurrence: monthlyRecurrenceForTest(22, 1),
		}
		require.NoError(t, task.Create(s, u))
		require.NoError(t, s.Commit())

		s2 := db.NewSession()
		defer s2.Close()

		err := hardDeleteTask(s2, &Task{ID: task.ID})
		require.NoError(t, err)
		require.NoError(t, s2.Commit())

		db.AssertMissing(t, "task_recurrences", map[string]interface{}{
			"task_id": task.ID,
		})
		db.AssertMissing(t, "tasks", map[string]interface{}{
			"id": task.ID,
		})
	})

	t.Run("invalid rule is rejected", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)

		s := db.NewSession()
		defer s.Close()

		task := &Task{
			Title:     "invalid recurrence",
			ProjectID: 1,
			Recurrence: &TaskRecurrence{
				Frequency: TaskRecurrenceFrequencyWeek,
				Interval:  1,
			},
		}

		err := task.Create(s, u)
		require.Error(t, err)
		assert.True(t, IsErrInvalidTaskRecurrence(err))
	})
}
