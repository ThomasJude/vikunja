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

		count, err := f.s.
			Where(
				"series_id = ? AND sequence > ?",
				f.series.ID,
				1,
			).
			Count(&TaskRecurrenceOccurrence{})
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("creates eligible occurrence", func(t *testing.T) {
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
			require.NotNil(
				t,
				occurrence,
				"sequence %d should have been materialized",
				sequence,
			)
		}

		next, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
			f.s,
			f.series.ID,
			5,
		)
		require.NoError(t, err)
		assert.Nil(t, next)
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

		for sequence := 2; sequence <= 4; sequence++ {
			occurrence, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
				f.s,
				f.series.ID,
				sequence,
			)
			require.NoError(t, err)
			require.NotNil(t, occurrence)
			assert.True(t, occurrence.DueDate.After(f.rootDue))
		}
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
}
