// Vikunja is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"testing"
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/require"
)

func TestTaskRecurrenceSeriesWithoutTaskDueDate(t *testing.T) {
	db.LoadAndAssertFixtures(t)

	s := db.NewSession()
	t.Cleanup(func() {
		_ = s.Close()
	})

	auth := &user.User{ID: 1}

	loc, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	start := time.Date(
		2026,
		time.September,
		23,
		9,
		0,
		0,
		0,
		loc,
	)

	// The task itself intentionally has no due date.
	_, err = s.
		ID(int64(1)).
		Cols("due_date").
		Update(&Task{
			DueDate: time.Time{},
		})
	require.NoError(t, err)

	rootTask, err := GetTaskByIDSimple(s, 1)
	require.NoError(t, err)
	require.True(t, rootTask.DueDate.IsZero())

	requested := &TaskRecurrenceSeries{
		Frequency: TaskRecurrenceFrequencyDay,
		Interval:  1,
		Basis:     TaskRecurrenceBasisSchedule,

		StartDate: start,

		EndType: TaskRecurrenceEndNever,

		// This must be ignored because the task has no due date.
		CreateBeforeDays: 5,

		WeekendPolicy: TaskRecurrenceWeekendKeep,
		MissedPolicy:  TaskRecurrenceMissedEveryOccurrence,
	}

	state, err := SaveTaskRecurrenceSeriesForTask(
		s,
		rootTask.ID,
		requested,
		auth,
	)
	require.NoError(t, err)
	require.NotNil(t, state)
	require.NotNil(t, state.Series)
	require.NotNil(t, state.Occurrence)

	// StartDate, not the task DueDate, is the recurrence anchor.
	require.True(t, state.Series.StartDate.Equal(start))

	// Create-before does not apply without a task due date.
	require.Equal(t, 0, state.Series.CreateBeforeDays)

	// The root occurrence keeps an internal scheduling anchor.
	require.True(t, state.Occurrence.ScheduledDueDate.Equal(start))
	require.True(t, state.Occurrence.DueDate.Equal(start))

	// Saving recurrence must not add a due date to the actual task.
	rootTask, err = GetTaskByIDSimple(s, rootTask.ID)
	require.NoError(t, err)
	require.True(t, rootTask.DueDate.IsZero())

	// Materialize the next daily occurrence.
	created, err := materializeTaskRecurrenceSeriesAtSession(
		s,
		state.Series.ID,
		start.AddDate(0, 0, 1),
	)
	require.NoError(t, err)
	require.Equal(t, 1, created)

	nextOccurrence, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
		s,
		state.Series.ID,
		2,
	)
	require.NoError(t, err)
	require.NotNil(t, nextOccurrence)

	generatedTask, err := GetTaskByIDSimple(
		s,
		nextOccurrence.TaskID,
	)
	require.NoError(t, err)

	// Generated recurring tasks must also remain due-date-free.
	require.True(t, generatedTask.DueDate.IsZero())

	// Scheduling metadata still tracks the recurrence calendar.
	require.False(t, nextOccurrence.ScheduledDueDate.IsZero())
	require.False(t, nextOccurrence.DueDate.IsZero())
}
