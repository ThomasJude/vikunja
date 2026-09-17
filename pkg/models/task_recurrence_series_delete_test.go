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

func taskRecurrenceTaskIsSoftDeleted(
	t *testing.T,
	f *taskRecurrenceMaterializerFixture,
	taskID int64,
) bool {
	t.Helper()

	exists, err := f.s.
		Unscoped().
		Where("id = ? AND deleted_at IS NOT NULL", taskID).
		Exist(&Task{})
	require.NoError(t, err)

	return exists
}

func TestTaskRecurrenceScopedDelete(t *testing.T) {
	t.Run("occurrence only deletes one and series continues", func(t *testing.T) {
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

		err = deleteTaskRecurrenceScoped(
			f.s,
			selected.TaskID,
			TaskRecurrenceDeleteOccurrence,
			auth,
		)
		require.NoError(t, err)

		assert.True(
			t,
			taskRecurrenceTaskIsSoftDeleted(t, f, selected.TaskID),
		)

		assert.False(
			t,
			taskRecurrenceTaskIsSoftDeleted(t, f, later.TaskID),
		)

		selected, err = getTaskRecurrenceOccurrenceByTaskID(
			f.s,
			selected.TaskID,
		)
		require.NoError(t, err)
		require.NotNil(t, selected)
		assert.True(t, selected.IsException)
		assert.False(t, selected.ExceptionAnchor.IsZero())

		created, err := materializeTaskRecurrenceSeriesAtSession(
			f.s,
			f.series.ID,
			f.rootDue.AddDate(0, 0, 3),
		)
		require.NoError(t, err)
		assert.Equal(t, 1, created)
	})

	t.Run("root occurrence only rebases template", func(t *testing.T) {
		f := materializeRecurrenceManagementFixture(t)
		auth := &user.User{ID: 1}

		replacement, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
			f.s,
			f.series.ID,
			2,
		)
		require.NoError(t, err)
		require.NotNil(t, replacement)

		err = deleteTaskRecurrenceScoped(
			f.s,
			f.current.TaskID,
			TaskRecurrenceDeleteOccurrence,
			auth,
		)
		require.NoError(t, err)

		storedSeries, err := getTaskRecurrenceSeriesByID(
			f.s,
			f.series.ID,
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			replacement.TaskID,
			storedSeries.RootTaskID,
		)
		assert.True(
			t,
			taskRecurrenceTaskIsSoftDeleted(
				t,
				f,
				f.current.TaskID,
			),
		)
	})

	t.Run("only root cannot be deleted without ending series", func(t *testing.T) {
		f := newTaskRecurrenceMaterializerFixture(t, 0)
		auth := &user.User{ID: 1}

		err := deleteTaskRecurrenceScoped(
			f.s,
			f.current.TaskID,
			TaskRecurrenceDeleteOccurrence,
			auth,
		)

		require.Error(t, err)
		assert.False(
			t,
			taskRecurrenceTaskIsSoftDeleted(
				t,
				f,
				f.current.TaskID,
			),
		)
	})

	t.Run("this and future deletes boundary and stops generation", func(t *testing.T) {
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

		err = deleteTaskRecurrenceScoped(
			f.s,
			selected.TaskID,
			TaskRecurrenceDeleteThisAndFuture,
			auth,
		)
		require.NoError(t, err)

		assert.False(
			t,
			taskRecurrenceTaskIsSoftDeleted(
				t,
				f,
				f.current.TaskID,
			),
		)
		assert.True(
			t,
			taskRecurrenceTaskIsSoftDeleted(
				t,
				f,
				selected.TaskID,
			),
		)
		assert.True(
			t,
			taskRecurrenceTaskIsSoftDeleted(
				t,
				f,
				later.TaskID,
			),
		)

		storedSeries, err := getTaskRecurrenceSeriesByID(
			f.s,
			f.series.ID,
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			TaskRecurrenceEndOccurrences,
			storedSeries.EndType,
		)
		assert.Equal(t, 1, storedSeries.EndAfterOccurrences)

		created, err := materializeTaskRecurrenceSeriesAtSession(
			f.s,
			f.series.ID,
			f.rootDue.AddDate(0, 0, 30),
		)
		require.NoError(t, err)
		assert.Equal(t, 0, created)
	})

	t.Run("entire series deletes every occurrence and stops generation", func(t *testing.T) {
		f := materializeRecurrenceManagementFixture(t)
		auth := &user.User{ID: 1}

		occurrences, err := getAllTaskRecurrenceOccurrences(
			f.s,
			f.series.ID,
		)
		require.NoError(t, err)
		require.Len(t, occurrences, 3)

		err = deleteTaskRecurrenceScoped(
			f.s,
			occurrences[1].TaskID,
			TaskRecurrenceDeleteEntireSeries,
			auth,
		)
		require.NoError(t, err)

		for _, occurrence := range occurrences {
			assert.True(
				t,
				taskRecurrenceTaskIsSoftDeleted(
					t,
					f,
					occurrence.TaskID,
				),
			)
		}

		created, err := materializeTaskRecurrenceSeriesAtSession(
			f.s,
			f.series.ID,
			f.rootDue.AddDate(0, 0, 30),
		)
		require.NoError(t, err)
		assert.Equal(t, 0, created)
	})

	t.Run("completion basis deleted occurrence uses exception anchor", func(t *testing.T) {
		f := materializeRecurrenceManagementFixture(t)
		auth := &user.User{ID: 1}

		f.series.Basis = TaskRecurrenceBasisCompletion
		f.series.MissedPolicy = TaskRecurrenceMissedNextFuture
		require.NoError(
			t,
			updateTaskRecurrenceSeries(f.s, f.series),
		)

		current, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
			f.s,
			f.series.ID,
			3,
		)
		require.NoError(t, err)
		require.NotNil(t, current)

		err = deleteTaskRecurrenceScoped(
			f.s,
			current.TaskID,
			TaskRecurrenceDeleteOccurrence,
			auth,
		)
		require.NoError(t, err)

		current, err = getTaskRecurrenceOccurrenceByTaskID(
			f.s,
			current.TaskID,
		)
		require.NoError(t, err)
		require.NotNil(t, current)
		require.True(t, current.IsException)
		require.False(t, current.ExceptionAnchor.IsZero())

		created, err := materializeTaskRecurrenceSeriesAtSession(
			f.s,
			f.series.ID,
			current.ExceptionAnchor.AddDate(0, 0, 2),
		)
		require.NoError(t, err)
		assert.Equal(t, 1, created)
	})
}
