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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskRecurrenceSeriesSchedule(t *testing.T) {
	location := time.FixedZone("PKT", 5*60*60)

	t.Run("series converts to recurrence rule", func(t *testing.T) {
		series := validTaskRecurrenceSeries()
		series.Interval = 3
		series.ByMonthDay = 15
		series.MissingPolicy = TaskRecurrenceMissingPolicyLastValid

		rule, err := taskRecurrenceRuleFromSeries(series)
		require.NoError(t, err)

		assert.Equal(t, series.Frequency, rule.Frequency)
		assert.Equal(t, 3, rule.Interval)
		assert.Equal(t, series.Basis, rule.Basis)
		assert.Equal(t, 15, rule.ByMonthDay)
		assert.Equal(
			t,
			TaskRecurrenceMissingPolicyLastValid,
			rule.MissingPolicy,
		)
	})

	t.Run("keep weekend date", func(t *testing.T) {
		series := validTaskRecurrenceSeries()
		series.WeekendPolicy = TaskRecurrenceWeekendKeep

		scheduled := time.Date(
			2026,
			time.September,
			19,
			9,
			0,
			0,
			0,
			location,
		)

		due, err := taskRecurrenceSeriesDueDate(series, scheduled)
		require.NoError(t, err)
		assert.True(t, due.Equal(scheduled))
	})

	t.Run("move saturday to previous business day", func(t *testing.T) {
		series := validTaskRecurrenceSeries()
		series.WeekendPolicy = TaskRecurrenceWeekendPreviousBusinessDay

		scheduled := time.Date(
			2026,
			time.September,
			19,
			9,
			0,
			0,
			0,
			location,
		)

		due, err := taskRecurrenceSeriesDueDate(series, scheduled)
		require.NoError(t, err)

		expected := time.Date(
			2026,
			time.September,
			18,
			9,
			0,
			0,
			0,
			location,
		)
		assert.True(t, due.Equal(expected))
	})

	t.Run("move sunday to previous business day", func(t *testing.T) {
		series := validTaskRecurrenceSeries()
		series.WeekendPolicy = TaskRecurrenceWeekendPreviousBusinessDay

		scheduled := time.Date(
			2026,
			time.September,
			20,
			9,
			0,
			0,
			0,
			location,
		)

		due, err := taskRecurrenceSeriesDueDate(series, scheduled)
		require.NoError(t, err)

		expected := time.Date(
			2026,
			time.September,
			18,
			9,
			0,
			0,
			0,
			location,
		)
		assert.True(t, due.Equal(expected))
	})

	t.Run("move saturday to next business day", func(t *testing.T) {
		series := validTaskRecurrenceSeries()
		series.WeekendPolicy = TaskRecurrenceWeekendNextBusinessDay

		scheduled := time.Date(
			2026,
			time.September,
			19,
			9,
			0,
			0,
			0,
			location,
		)

		due, err := taskRecurrenceSeriesDueDate(series, scheduled)
		require.NoError(t, err)

		expected := time.Date(
			2026,
			time.September,
			21,
			9,
			0,
			0,
			0,
			location,
		)
		assert.True(t, due.Equal(expected))
	})

	t.Run("move sunday to next business day", func(t *testing.T) {
		series := validTaskRecurrenceSeries()
		series.WeekendPolicy = TaskRecurrenceWeekendNextBusinessDay

		scheduled := time.Date(
			2026,
			time.September,
			20,
			9,
			0,
			0,
			0,
			location,
		)

		due, err := taskRecurrenceSeriesDueDate(series, scheduled)
		require.NoError(t, err)

		expected := time.Date(
			2026,
			time.September,
			21,
			9,
			0,
			0,
			0,
			location,
		)
		assert.True(t, due.Equal(expected))
	})

	t.Run("create before due date", func(t *testing.T) {
		series := validTaskRecurrenceSeries()
		series.CreateBeforeDays = 5

		due := time.Date(
			2026,
			time.October,
			10,
			9,
			0,
			0,
			0,
			location,
		)

		createAt, err := taskRecurrenceSeriesCreateAt(series, due)
		require.NoError(t, err)

		expected := time.Date(
			2026,
			time.October,
			5,
			9,
			0,
			0,
			0,
			location,
		)
		assert.True(t, createAt.Equal(expected))
	})

	t.Run("create on due date by default", func(t *testing.T) {
		series := validTaskRecurrenceSeries()

		due := time.Date(
			2026,
			time.October,
			10,
			9,
			0,
			0,
			0,
			location,
		)

		createAt, err := taskRecurrenceSeriesCreateAt(series, due)
		require.NoError(t, err)
		assert.True(t, createAt.Equal(due))
	})

	t.Run("never end allows occurrence", func(t *testing.T) {
		series := validTaskRecurrenceSeries()

		allowed, err := taskRecurrenceSeriesAllowsOccurrence(
			series,
			100,
			series.StartDate.AddDate(10, 0, 0),
		)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("end date is inclusive", func(t *testing.T) {
		series := validTaskRecurrenceSeries()
		series.EndType = TaskRecurrenceEndDate
		series.EndDate = series.StartDate.AddDate(0, 3, 0)

		allowed, err := taskRecurrenceSeriesAllowsOccurrence(
			series,
			4,
			series.EndDate,
		)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("occurrence after end date rejected", func(t *testing.T) {
		series := validTaskRecurrenceSeries()
		series.EndType = TaskRecurrenceEndDate
		series.EndDate = series.StartDate.AddDate(0, 3, 0)

		allowed, err := taskRecurrenceSeriesAllowsOccurrence(
			series,
			5,
			series.EndDate.Add(time.Second),
		)
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("occurrence count is inclusive", func(t *testing.T) {
		series := validTaskRecurrenceSeries()
		series.EndType = TaskRecurrenceEndOccurrences
		series.EndAfterOccurrences = 5

		allowed, err := taskRecurrenceSeriesAllowsOccurrence(
			series,
			5,
			series.StartDate,
		)
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = taskRecurrenceSeriesAllowsOccurrence(
			series,
			6,
			series.StartDate,
		)
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}
