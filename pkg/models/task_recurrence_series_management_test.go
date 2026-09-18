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

	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func materializeRecurrenceManagementFixture(
	t *testing.T,
) *taskRecurrenceMaterializerFixture {
	t.Helper()

	f := newTaskRecurrenceMaterializerFixture(t, 0)

	created, err := materializeTaskRecurrenceSeriesAtSession(
		f.s,
		f.series.ID,
		f.rootDue.AddDate(0, 0, 2),
	)
	require.NoError(t, err)
	require.Equal(t, 2, created)

	return f
}

func TestTaskRecurrenceSeriesPauseResume(t *testing.T) {
	f := newTaskRecurrenceMaterializerFixture(t, 0)
	auth := &user.User{ID: 1}

	series, err := setTaskRecurrenceSeriesPaused(
		f.s,
		f.series.ID,
		true,
		auth,
	)
	require.NoError(t, err)
	require.True(t, series.Paused)

	stored, err := getTaskRecurrenceSeriesByID(f.s, f.series.ID)
	require.NoError(t, err)
	assert.True(t, stored.Paused)

	series, err = setTaskRecurrenceSeriesPaused(
		f.s,
		f.series.ID,
		false,
		auth,
	)
	require.NoError(t, err)
	require.False(t, series.Paused)

	stored, err = getTaskRecurrenceSeriesByID(f.s, f.series.ID)
	require.NoError(t, err)
	assert.False(t, stored.Paused)
}

func TestRemoveTaskRecurrenceSeriesKeepsTasks(t *testing.T) {
	f := materializeRecurrenceManagementFixture(t)
	auth := &user.User{ID: 1}

	occurrences, err := getAllTaskRecurrenceOccurrences(
		f.s,
		f.series.ID,
	)
	require.NoError(t, err)
	require.Len(t, occurrences, 3)

	taskIDs := make([]int64, 0, len(occurrences))
	for _, occurrence := range occurrences {
		taskIDs = append(taskIDs, occurrence.TaskID)

		task, err := GetTaskByIDSimple(f.s, occurrence.TaskID)
		require.NoError(t, err)
		require.NotNil(t, task)
	}

	// Remove recurrence from one task in the series. This must remove only
	// recurrence metadata and preserve every already-created task.
	err = RemoveTaskRecurrenceSeriesForTask(
		f.s,
		occurrences[1].TaskID,
		auth,
	)
	require.NoError(t, err)

	hasSeries, err := f.s.
		ID(f.series.ID).
		Exist(&TaskRecurrenceSeries{})
	require.NoError(t, err)
	assert.False(t, hasSeries)

	occurrenceCount, err := f.s.
		Where("series_id = ?", f.series.ID).
		Count(&TaskRecurrenceOccurrence{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), occurrenceCount)

	for _, taskID := range taskIDs {
		task, err := GetTaskByIDSimple(f.s, taskID)
		require.NoError(t, err)
		require.NotNil(t, task)
		assert.Equal(t, taskID, task.ID)

		state, err := GetTaskRecurrenceSeriesState(
			f.s,
			taskID,
			auth,
		)
		require.NoError(t, err)
		require.NotNil(t, state)
		assert.Nil(t, state.Series)
		assert.Nil(t, state.Occurrence)
	}
}

func TestTaskRecurrenceScopedUpdate(t *testing.T) {
	t.Run("occurrence only marks exception", func(t *testing.T) {
		f := materializeRecurrenceManagementFixture(t)
		auth := &user.User{ID: 1}

		selected, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
			f.s,
			f.series.ID,
			2,
		)
		require.NoError(t, err)
		require.NotNil(t, selected)

		later, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
			f.s,
			f.series.ID,
			3,
		)
		require.NoError(t, err)
		require.NotNil(t, later)

		_, err = updateTaskRecurrenceScoped(
			f.s,
			selected.TaskID,
			&Task{Title: "Only this occurrence"},
			[]string{"title"},
			TaskRecurrenceEditOccurrence,
			auth,
		)
		require.NoError(t, err)

		updated, err := GetTaskByIDSimple(f.s, selected.TaskID)
		require.NoError(t, err)
		assert.Equal(t, "Only this occurrence", updated.Title)

		untouched, err := GetTaskByIDSimple(f.s, later.TaskID)
		require.NoError(t, err)
		assert.NotEqual(t, "Only this occurrence", untouched.Title)

		selected, err = getTaskRecurrenceOccurrenceByTaskID(
			f.s,
			selected.TaskID,
		)
		require.NoError(t, err)
		require.NotNil(t, selected)
		assert.True(t, selected.IsException)
	})

	t.Run("this and future splits series", func(t *testing.T) {
		f := materializeRecurrenceManagementFixture(t)
		auth := &user.User{ID: 1}

		selected, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
			f.s,
			f.series.ID,
			2,
		)
		require.NoError(t, err)
		require.NotNil(t, selected)

		later, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
			f.s,
			f.series.ID,
			3,
		)
		require.NoError(t, err)
		require.NotNil(t, later)

		newSeries, err := updateTaskRecurrenceScoped(
			f.s,
			selected.TaskID,
			&Task{Title: "Changed from here"},
			[]string{"title"},
			TaskRecurrenceEditThisAndFuture,
			auth,
		)
		require.NoError(t, err)
		require.NotNil(t, newSeries)
		assert.NotEqual(t, f.series.ID, newSeries.ID)
		assert.Equal(t, selected.TaskID, newSeries.RootTaskID)

		oldSeries, err := getTaskRecurrenceSeriesByID(f.s, f.series.ID)
		require.NoError(t, err)
		assert.Equal(t, TaskRecurrenceEndOccurrences, oldSeries.EndType)
		assert.Equal(t, 1, oldSeries.EndAfterOccurrences)

		movedSelected, err := getTaskRecurrenceOccurrenceByTaskID(
			f.s,
			selected.TaskID,
		)
		require.NoError(t, err)
		require.NotNil(t, movedSelected)
		assert.Equal(t, newSeries.ID, movedSelected.SeriesID)
		assert.Equal(t, 1, movedSelected.Sequence)

		movedLater, err := getTaskRecurrenceOccurrenceByTaskID(
			f.s,
			later.TaskID,
		)
		require.NoError(t, err)
		require.NotNil(t, movedLater)
		assert.Equal(t, newSeries.ID, movedLater.SeriesID)
		assert.Equal(t, 2, movedLater.Sequence)

		root, err := GetTaskByIDSimple(f.s, f.current.TaskID)
		require.NoError(t, err)
		assert.NotEqual(t, "Changed from here", root.Title)

		changedSelected, err := GetTaskByIDSimple(f.s, selected.TaskID)
		require.NoError(t, err)
		assert.Equal(t, "Changed from here", changedSelected.Title)

		changedLater, err := GetTaskByIDSimple(f.s, later.TaskID)
		require.NoError(t, err)
		assert.Equal(t, "Changed from here", changedLater.Title)
	})

	t.Run("entire series updates all materialized occurrences", func(t *testing.T) {
		f := materializeRecurrenceManagementFixture(t)
		auth := &user.User{ID: 1}

		selected, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
			f.s,
			f.series.ID,
			2,
		)
		require.NoError(t, err)
		require.NotNil(t, selected)

		series, err := updateTaskRecurrenceScoped(
			f.s,
			selected.TaskID,
			&Task{Priority: 4},
			[]string{"priority"},
			TaskRecurrenceEditEntireSeries,
			auth,
		)
		require.NoError(t, err)
		assert.Equal(t, f.series.ID, series.ID)

		occurrences, err := getAllTaskRecurrenceOccurrences(
			f.s,
			f.series.ID,
		)
		require.NoError(t, err)
		require.Len(t, occurrences, 3)

		for _, occurrence := range occurrences {
			task, err := GetTaskByIDSimple(f.s, occurrence.TaskID)
			require.NoError(t, err)
			assert.Equal(t, int64(4), task.Priority)
		}
	})

	t.Run("future scope rejects absolute due date propagation", func(t *testing.T) {
		f := materializeRecurrenceManagementFixture(t)
		auth := &user.User{ID: 1}

		selected, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
			f.s,
			f.series.ID,
			2,
		)
		require.NoError(t, err)
		require.NotNil(t, selected)

		_, err = updateTaskRecurrenceScoped(
			f.s,
			selected.TaskID,
			&Task{DueDate: f.rootDue},
			[]string{"due_date"},
			TaskRecurrenceEditThisAndFuture,
			auth,
		)
		require.Error(t, err)
	})
}

func TestTaskRecurrenceScopedUpdateFutureMaterialization(t *testing.T) {
	t.Run("occurrence exception does not leak to future occurrence", func(t *testing.T) {
		f := newTaskRecurrenceMaterializerFixture(t, 0)
		auth := &user.User{ID: 1}

		created, err := materializeTaskRecurrenceSeriesAtSession(
			f.s,
			f.series.ID,
			f.nextDue,
		)
		require.NoError(t, err)
		require.Equal(t, 1, created)

		selected, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
			f.s,
			f.series.ID,
			2,
		)
		require.NoError(t, err)
		require.NotNil(t, selected)

		_, err = updateTaskRecurrenceScoped(
			f.s,
			selected.TaskID,
			&Task{Title: "One-off title"},
			[]string{"title"},
			TaskRecurrenceEditOccurrence,
			auth,
		)
		require.NoError(t, err)

		created, err = materializeTaskRecurrenceSeriesAtSession(
			f.s,
			f.series.ID,
			f.rootDue.AddDate(0, 0, 2),
		)
		require.NoError(t, err)
		require.Equal(t, 1, created)

		future, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
			f.s,
			f.series.ID,
			3,
		)
		require.NoError(t, err)
		require.NotNil(t, future)

		futureTask, err := GetTaskByIDSimple(f.s, future.TaskID)
		require.NoError(t, err)
		assert.NotEqual(t, "One-off title", futureTask.Title)
	})

	t.Run("this and future becomes new template", func(t *testing.T) {
		f := newTaskRecurrenceMaterializerFixture(t, 0)
		auth := &user.User{ID: 1}

		created, err := materializeTaskRecurrenceSeriesAtSession(
			f.s,
			f.series.ID,
			f.nextDue,
		)
		require.NoError(t, err)
		require.Equal(t, 1, created)

		selected, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
			f.s,
			f.series.ID,
			2,
		)
		require.NoError(t, err)
		require.NotNil(t, selected)

		newSeries, err := updateTaskRecurrenceScoped(
			f.s,
			selected.TaskID,
			&Task{Title: "Future template title"},
			[]string{"title"},
			TaskRecurrenceEditThisAndFuture,
			auth,
		)
		require.NoError(t, err)
		require.NotNil(t, newSeries)

		created, err = materializeTaskRecurrenceSeriesAtSession(
			f.s,
			newSeries.ID,
			f.rootDue.AddDate(0, 0, 2),
		)
		require.NoError(t, err)
		require.Equal(t, 1, created)

		future, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
			f.s,
			newSeries.ID,
			2,
		)
		require.NoError(t, err)
		require.NotNil(t, future)

		futureTask, err := GetTaskByIDSimple(f.s, future.TaskID)
		require.NoError(t, err)
		assert.Equal(t, "Future template title", futureTask.Title)
	})

	t.Run("entire series update is inherited by future occurrence", func(t *testing.T) {
		f := newTaskRecurrenceMaterializerFixture(t, 0)
		auth := &user.User{ID: 1}

		_, err := updateTaskRecurrenceScoped(
			f.s,
			f.current.TaskID,
			&Task{Priority: 5},
			[]string{"priority"},
			TaskRecurrenceEditEntireSeries,
			auth,
		)
		require.NoError(t, err)

		created, err := materializeTaskRecurrenceSeriesAtSession(
			f.s,
			f.series.ID,
			f.nextDue,
		)
		require.NoError(t, err)
		require.Equal(t, 1, created)

		next, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
			f.s,
			f.series.ID,
			2,
		)
		require.NoError(t, err)
		require.NotNil(t, next)

		nextTask, err := GetTaskByIDSimple(f.s, next.TaskID)
		require.NoError(t, err)
		assert.Equal(t, int64(5), nextTask.Priority)
	})

	t.Run("pause blocks materialization and resume continues series", func(t *testing.T) {
		f := newTaskRecurrenceMaterializerFixture(t, 0)
		auth := &user.User{ID: 1}

		_, err := setTaskRecurrenceSeriesPaused(
			f.s,
			f.series.ID,
			true,
			auth,
		)
		require.NoError(t, err)

		created, err := materializeTaskRecurrenceSeriesAtSession(
			f.s,
			f.series.ID,
			f.nextDue,
		)
		require.NoError(t, err)
		assert.Equal(t, 0, created)

		_, err = setTaskRecurrenceSeriesPaused(
			f.s,
			f.series.ID,
			false,
			auth,
		)
		require.NoError(t, err)

		created, err = materializeTaskRecurrenceSeriesAtSession(
			f.s,
			f.series.ID,
			f.nextDue,
		)
		require.NoError(t, err)
		assert.Equal(t, 1, created)
	})
}
