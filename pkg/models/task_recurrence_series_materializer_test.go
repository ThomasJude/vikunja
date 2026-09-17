// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"testing"
	"time"

	"code.vikunja.io/api/pkg/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

type taskRecurrenceMaterializerFixture struct {
	s          *xorm.Session
	series     *TaskRecurrenceSeries
	current    *TaskRecurrenceOccurrence
	rootDue    time.Time
	nextDue    time.Time
	nextCreate time.Time
}

func newTaskRecurrenceMaterializerFixture(
	t *testing.T,
	createBeforeDays int,
) *taskRecurrenceMaterializerFixture {
	t.Helper()

	db.LoadAndAssertFixtures(t)

	s := db.NewSession()
	t.Cleanup(func() {
		_ = s.Close()
	})

	loc, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	rootDue := time.Date(
		2026,
		time.September,
		17,
		9,
		0,
		0,
		0,
		loc,
	)

	// Make the root task deterministic for recurrence materialization.
	_, err = s.ID(int64(1)).
		Cols(
			"title",
			"description",
			"due_date",
			"start_date",
			"end_date",
			"repeat_after",
			"percent_done",
		).
		Update(&Task{
			Title: "Recurring series template",
			Description: `<ul data-type="taskList"><li data-checked="true">` +
				`<label><input type="checkbox" checked="checked"></label>` +
				`<div>Recurring checklist item</div></li></ul>`,
			DueDate:     rootDue,
			StartDate:   rootDue.Add(-time.Hour),
			EndDate:     rootDue.Add(time.Hour),
			RepeatAfter: 86400,
			PercentDone: 0.75,
		})
	require.NoError(t, err)

	// Give the template one deterministic reminder. The materialized task must
	// preserve the offset relative to its new due date.
	reminderTask := &Task{
		ID: 1,
		Reminders: []*TaskReminder{
			{
				Reminder: rootDue.Add(-30 * time.Minute),
			},
		},
	}
	require.NoError(
		t,
		reminderTask.updateReminders(s, reminderTask),
	)

	series := &TaskRecurrenceSeries{
		RootTaskID:  1,
		ProjectID:   1,
		CreatedByID: 1,

		Frequency: TaskRecurrenceFrequencyDay,
		Interval:  1,
		Basis:     TaskRecurrenceBasisSchedule,

		StartDate: rootDue,

		EndType: TaskRecurrenceEndNever,

		CreateBeforeDays: createBeforeDays,

		WeekendPolicy: TaskRecurrenceWeekendKeep,
		MissedPolicy:  TaskRecurrenceMissedEveryOccurrence,
	}

	require.NoError(
		t,
		createTaskRecurrenceSeries(s, series),
	)
	require.NotZero(t, series.ID)

	current := &TaskRecurrenceOccurrence{
		SeriesID:         series.ID,
		TaskID:           1,
		Sequence:         1,
		ScheduledDueDate: rootDue,
		DueDate:          rootDue,
	}

	require.NoError(
		t,
		createTaskRecurrenceOccurrence(s, current),
	)

	nextDue := rootDue.AddDate(0, 0, 1)
	nextCreate := nextDue.AddDate(0, 0, -createBeforeDays)

	return &taskRecurrenceMaterializerFixture{
		s:          s,
		series:     series,
		current:    current,
		rootDue:    rootDue,
		nextDue:    nextDue,
		nextCreate: nextCreate,
	}
}

func TestTaskRecurrenceSeriesMaterializer(t *testing.T) {
	t.Run("does not create before create at", func(t *testing.T) {
		f := newTaskRecurrenceMaterializerFixture(t, 2)

		occurrence, err := materializeTaskRecurrenceSeriesOccurrence(
			f.s,
			f.series,
			f.current,
			f.nextCreate.Add(-time.Minute),
		)

		require.NoError(t, err)
		assert.Nil(t, occurrence)

		count, err := f.s.
			Where(
				"series_id = ? AND sequence = ?",
				f.series.ID,
				2,
			).
			Count(&TaskRecurrenceOccurrence{})
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("creates occurrence when create at is reached", func(t *testing.T) {
		f := newTaskRecurrenceMaterializerFixture(t, 2)

		originalLabelCount, err := f.s.
			Where("task_id = ?", int64(1)).
			Count(&LabelTask{})
		require.NoError(t, err)

		originalAssigneeCount, err := f.s.
			Where("task_id = ?", int64(1)).
			Count(&TaskAssginee{})
		require.NoError(t, err)

		occurrence, err := materializeTaskRecurrenceSeriesOccurrence(
			f.s,
			f.series,
			f.current,
			f.nextCreate,
		)

		require.NoError(t, err)
		require.NotNil(t, occurrence)
		require.NotZero(t, occurrence.ID)
		require.NotZero(t, occurrence.TaskID)

		assert.Equal(t, f.series.ID, occurrence.SeriesID)
		assert.Equal(t, 2, occurrence.Sequence)

		assert.True(
			t,
			f.nextDue.Equal(occurrence.ScheduledDueDate),
			"scheduled due date differs: expected %s, got %s",
			f.nextDue,
			occurrence.ScheduledDueDate,
		)
		assert.True(
			t,
			f.nextDue.Equal(occurrence.DueDate),
			"due date differs: expected %s, got %s",
			f.nextDue,
			occurrence.DueDate,
		)

		newTask, err := GetTaskByIDSimple(
			f.s,
			occurrence.TaskID,
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			"Recurring series template",
			newTask.Title,
		)
		assert.False(t, newTask.Done)
		assert.Equal(t, float64(0), newTask.PercentDone)
		assert.Equal(t, int64(0), newTask.RepeatAfter)
		assert.Nil(t, newTask.Recurrence)

		assert.True(
			t,
			f.nextDue.Equal(newTask.DueDate),
			"task due date differs: expected %s, got %s",
			f.nextDue,
			newTask.DueDate,
		)

		expectedStart := f.nextDue.Add(-time.Hour)
		assert.True(
			t,
			expectedStart.Equal(newTask.StartDate),
			"task start date differs: expected %s, got %s",
			expectedStart,
			newTask.StartDate,
		)

		expectedEnd := f.nextDue.Add(time.Hour)
		assert.True(
			t,
			expectedEnd.Equal(newTask.EndDate),
			"task end date differs: expected %s, got %s",
			expectedEnd,
			newTask.EndDate,
		)

		// Checklist state must reset for each occurrence.
		assert.Contains(
			t,
			newTask.Description,
			`data-checked="false"`,
		)
		assert.NotContains(
			t,
			newTask.Description,
			`data-checked="true"`,
		)
		assert.NotContains(
			t,
			newTask.Description,
			`checked="checked"`,
		)

		reminders, err := getRemindersForTasks(
			f.s,
			[]int64{newTask.ID},
		)
		require.NoError(t, err)
		require.Len(t, reminders, 1)

		expectedReminder := f.nextDue.Add(-30 * time.Minute)
		assert.True(
			t,
			expectedReminder.Equal(reminders[0].Reminder),
			"reminder differs: expected %s, got %s",
			expectedReminder,
			reminders[0].Reminder,
		)

		newLabelCount, err := f.s.
			Where("task_id = ?", newTask.ID).
			Count(&LabelTask{})
		require.NoError(t, err)
		assert.Equal(
			t,
			originalLabelCount,
			newLabelCount,
		)

		newAssigneeCount, err := f.s.
			Where("task_id = ?", newTask.ID).
			Count(&TaskAssginee{})
		require.NoError(t, err)
		assert.Equal(
			t,
			originalAssigneeCount,
			newAssigneeCount,
		)

		// Recurrence occurrences must not look like manually duplicated tasks.
		relationCount, err := f.s.
			Where(
				"task_id = ? AND relation_kind = ?",
				newTask.ID,
				RelationKindCopiedFrom,
			).
			Count(&TaskRelation{})
		require.NoError(t, err)
		assert.Equal(t, int64(0), relationCount)
	})

	t.Run("create before days materializes future due task", func(t *testing.T) {
		f := newTaskRecurrenceMaterializerFixture(t, 5)

		// The next task is due tomorrow, but with create-before=5 it should
		// already be materializable four days before the current occurrence.
		occurrence, err := materializeTaskRecurrenceSeriesOccurrence(
			f.s,
			f.series,
			f.current,
			f.nextCreate,
		)

		require.NoError(t, err)
		require.NotNil(t, occurrence)

		assert.True(
			t,
			occurrence.DueDate.After(f.nextCreate),
			"occurrence should be created before its due date",
		)
		assert.Equal(
			t,
			5,
			int(occurrence.DueDate.Sub(f.nextCreate).Hours()/24),
		)
	})

	t.Run("repeated materialization is idempotent", func(t *testing.T) {
		f := newTaskRecurrenceMaterializerFixture(t, 2)

		first, err := materializeTaskRecurrenceSeriesOccurrence(
			f.s,
			f.series,
			f.current,
			f.nextCreate,
		)
		require.NoError(t, err)
		require.NotNil(t, first)

		taskCountBefore, err := f.s.
			Where("project_id = ?", f.series.ProjectID).
			Count(&Task{})
		require.NoError(t, err)

		second, err := materializeTaskRecurrenceSeriesOccurrence(
			f.s,
			f.series,
			f.current,
			f.nextCreate,
		)
		require.NoError(t, err)
		require.NotNil(t, second)

		assert.Equal(t, first.ID, second.ID)
		assert.Equal(t, first.TaskID, second.TaskID)
		assert.Equal(t, first.Sequence, second.Sequence)

		taskCountAfter, err := f.s.
			Where("project_id = ?", f.series.ProjectID).
			Count(&Task{})
		require.NoError(t, err)
		assert.Equal(t, taskCountBefore, taskCountAfter)

		occurrenceCount, err := f.s.
			Where(
				"series_id = ? AND sequence = ?",
				f.series.ID,
				2,
			).
			Count(&TaskRecurrenceOccurrence{})
		require.NoError(t, err)
		assert.Equal(t, int64(1), occurrenceCount)
	})

	t.Run("paused series creates nothing", func(t *testing.T) {
		f := newTaskRecurrenceMaterializerFixture(t, 2)

		f.series.Paused = true
		require.NoError(
			t,
			updateTaskRecurrenceSeries(f.s, f.series),
		)

		occurrence, err := materializeTaskRecurrenceSeriesOccurrence(
			f.s,
			f.series,
			f.current,
			f.nextCreate,
		)

		require.NoError(t, err)
		assert.Nil(t, occurrence)
	})

	t.Run("occurrence limit stops materialization", func(t *testing.T) {
		f := newTaskRecurrenceMaterializerFixture(t, 2)

		f.series.EndType = TaskRecurrenceEndOccurrences
		f.series.EndAfterOccurrences = 1

		require.NoError(
			t,
			updateTaskRecurrenceSeries(f.s, f.series),
		)

		occurrence, err := materializeTaskRecurrenceSeriesOccurrence(
			f.s,
			f.series,
			f.current,
			f.nextCreate,
		)

		require.NoError(t, err)
		assert.Nil(t, occurrence)
	})
}

func TestTaskRecurrenceSeriesMaterializerTransactionRollback(t *testing.T) {
	f := newTaskRecurrenceMaterializerFixture(t, 2)

	// The series and root occurrence are fixture state. Only materialization
	// itself is performed inside this transaction.
	require.NoError(t, f.s.Begin())

	occurrence, err := materializeTaskRecurrenceSeriesOccurrence(
		f.s,
		f.series,
		f.current,
		f.nextCreate,
	)
	require.NoError(t, err)
	require.NotNil(t, occurrence)
	require.NotZero(t, occurrence.TaskID)

	newTaskID := occurrence.TaskID

	// A caller such as the recurrence cron must be able to roll back the whole
	// materialization if the occurrence insert loses an idempotency race or any
	// later step fails.
	require.NoError(t, f.s.Rollback())

	verify := db.NewSession()
	defer verify.Close()

	taskCount, err := verify.
		Where("id = ?", newTaskID).
		Count(&Task{})
	require.NoError(t, err)
	assert.Equal(
		t,
		int64(0),
		taskCount,
		"materialized task must disappear after transaction rollback",
	)

	occurrenceCount, err := verify.
		Where(
			"series_id = ? AND sequence = ?",
			f.series.ID,
			2,
		).
		Count(&TaskRecurrenceOccurrence{})
	require.NoError(t, err)
	assert.Equal(
		t,
		int64(0),
		occurrenceCount,
		"recurrence occurrence must disappear after transaction rollback",
	)
}
