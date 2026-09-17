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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskRecurrenceSeriesMaterializerRunner(t *testing.T) {
	t.Run("does not create before create at", func(t *testing.T) {
		f := newTaskRecurrenceMaterializerFixture(t, 0)

		created, err := materializeTaskRecurrenceSeriesAtSession(
			f.s,
			f.series.ID,
			f.nextDue.Add(-time.Minute),
		)

		require.NoError(t, err)
		assert.Equal(t, 0, created)
	})

	t.Run("creates eligible schedule occurrence", func(t *testing.T) {
		f := newTaskRecurrenceMaterializerFixture(t, 0)

		created, err := materializeTaskRecurrenceSeriesAtSession(
			f.s,
			f.series.ID,
			f.nextDue,
		)

		require.NoError(t, err)
		assert.Equal(t, 1, created)

		occurrence, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
			f.s,
			f.series.ID,
			2,
		)
		require.NoError(t, err)
		require.NotNil(t, occurrence)
		assert.True(t, f.nextDue.Equal(occurrence.DueDate))
	})

	t.Run("generate every missed occurrence catches up", func(t *testing.T) {
		f := newTaskRecurrenceMaterializerFixture(t, 0)

		reference := f.rootDue.AddDate(0, 0, 3)

		created, err := materializeTaskRecurrenceSeriesAtSession(
			f.s,
			f.series.ID,
			reference,
		)

		require.NoError(t, err)
		assert.Equal(t, 3, created)

		for sequence := 2; sequence <= 4; sequence++ {
			occurrence, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
				f.s,
				f.series.ID,
				sequence,
			)
			require.NoError(t, err)
			require.NotNil(t, occurrence)
		}
	})

	t.Run("create before can materialize multiple future occurrences", func(t *testing.T) {
		f := newTaskRecurrenceMaterializerFixture(t, 3)

		created, err := materializeTaskRecurrenceSeriesAtSession(
			f.s,
			f.series.ID,
			f.rootDue,
		)

		require.NoError(t, err)
		assert.Equal(t, 3, created)
	})

	t.Run("paused series is ignored", func(t *testing.T) {
		f := newTaskRecurrenceMaterializerFixture(t, 0)

		f.series.Paused = true
		require.NoError(
			t,
			updateTaskRecurrenceSeries(f.s, f.series),
		)

		created, err := materializeTaskRecurrenceSeriesAtSession(
			f.s,
			f.series.ID,
			f.nextDue,
		)

		require.NoError(t, err)
		assert.Equal(t, 0, created)
	})

	t.Run("active series scan excludes paused series", func(t *testing.T) {
		f := newTaskRecurrenceMaterializerFixture(t, 0)

		ids, err := getActiveTaskRecurrenceSeriesIDs(f.s)
		require.NoError(t, err)
		assert.Contains(t, ids, f.series.ID)

		f.series.Paused = true
		require.NoError(
			t,
			updateTaskRecurrenceSeries(f.s, f.series),
		)

		ids, err = getActiveTaskRecurrenceSeriesIDs(f.s)
		require.NoError(t, err)
		assert.NotContains(t, ids, f.series.ID)
	})

	t.Run("completion basis does not advance unfinished occurrence", func(t *testing.T) {
		f := newTaskRecurrenceMaterializerFixture(t, 0)

		f.series.Basis = TaskRecurrenceBasisCompletion
		f.series.MissedPolicy = TaskRecurrenceMissedNextFuture
		require.NoError(
			t,
			updateTaskRecurrenceSeries(f.s, f.series),
		)

		created, err := materializeTaskRecurrenceSeriesAtSession(
			f.s,
			f.series.ID,
			f.rootDue.AddDate(0, 0, 30),
		)

		require.NoError(t, err)
		assert.Equal(t, 0, created)

		next, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
			f.s,
			f.series.ID,
			2,
		)
		require.NoError(t, err)
		assert.Nil(t, next)
	})

	t.Run("completion basis uses actual done at anchor", func(t *testing.T) {
		f := newTaskRecurrenceMaterializerFixture(t, 0)

		f.series.Basis = TaskRecurrenceBasisCompletion
		f.series.MissedPolicy = TaskRecurrenceMissedNextFuture
		require.NoError(
			t,
			updateTaskRecurrenceSeries(f.s, f.series),
		)

		completedAt := f.rootDue.Add(5 * time.Hour)

		_, err := f.s.ID(int64(1)).
			Cols("done", "done_at").
			Update(&Task{
				Done:   true,
				DoneAt: completedAt,
			})
		require.NoError(t, err)

		// Daily completion recurrence preserves the scheduled wall-clock time.
		expectedDue := f.rootDue.AddDate(0, 0, 1)

		// Completion itself is not enough when CreateAt is still in the future.
		created, err := materializeTaskRecurrenceSeriesAtSession(
			f.s,
			f.series.ID,
			completedAt,
		)
		require.NoError(t, err)
		assert.Equal(t, 0, created)

		created, err = materializeTaskRecurrenceSeriesAtSession(
			f.s,
			f.series.ID,
			expectedDue,
		)
		require.NoError(t, err)
		assert.Equal(t, 1, created)

		next, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
			f.s,
			f.series.ID,
			2,
		)
		require.NoError(t, err)
		require.NotNil(t, next)

		assert.True(
			t,
			expectedDue.Equal(next.ScheduledDueDate),
			"expected %s, got %s",
			expectedDue,
			next.ScheduledDueDate,
		)
	})

	t.Run("completion basis create before can create immediately after completion", func(t *testing.T) {
		f := newTaskRecurrenceMaterializerFixture(t, 2)

		f.series.Basis = TaskRecurrenceBasisCompletion
		f.series.MissedPolicy = TaskRecurrenceMissedNextFuture
		require.NoError(
			t,
			updateTaskRecurrenceSeries(f.s, f.series),
		)

		completedAt := f.rootDue.Add(2 * time.Hour)

		_, err := f.s.ID(int64(1)).
			Cols("done", "done_at").
			Update(&Task{
				Done:   true,
				DoneAt: completedAt,
			})
		require.NoError(t, err)

		created, err := materializeTaskRecurrenceSeriesAtSession(
			f.s,
			f.series.ID,
			completedAt,
		)

		require.NoError(t, err)
		assert.Equal(t, 1, created)

		next, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
			f.s,
			f.series.ID,
			2,
		)
		require.NoError(t, err)
		require.NotNil(t, next)

		assert.True(t, next.DueDate.After(completedAt))
	})
}
