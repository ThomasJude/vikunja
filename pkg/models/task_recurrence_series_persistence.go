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
	"fmt"
	"time"

	"xorm.io/xorm"
)

func createTaskRecurrenceSeries(
	s *xorm.Session,
	series *TaskRecurrenceSeries,
) error {
	if series == nil {
		return fmt.Errorf("recurrence series is required")
	}

	if err := validateTaskRecurrenceSeries(series); err != nil {
		return err
	}

	series.ID = 0

	_, err := s.Insert(series)
	return err
}

func updateTaskRecurrenceSeries(
	s *xorm.Session,
	series *TaskRecurrenceSeries,
) error {
	if series == nil || series.ID <= 0 {
		return fmt.Errorf("recurrence series id is required")
	}

	if err := validateTaskRecurrenceSeries(series); err != nil {
		return err
	}

	_, err := s.ID(series.ID).
		Cols(
			"frequency",
			"interval",
			"basis",
			"by_weekdays",
			"by_month",
			"by_month_day",
			"by_set_pos",
			"missing_policy",
			"start_date",
			"end_type",
			"end_date",
			"end_after_occurrences",
			"create_before_days",
			"weekend_policy",
			"missed_policy",
			"paused",
		).
		Update(series)

	return err
}

func getTaskRecurrenceSeriesByID(
	s *xorm.Session,
	id int64,
) (*TaskRecurrenceSeries, error) {
	if id <= 0 {
		return nil, fmt.Errorf("recurrence series id is required")
	}

	series := &TaskRecurrenceSeries{ID: id}

	has, err := s.ID(id).Get(series)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, fmt.Errorf("recurrence series %d does not exist", id)
	}

	return series, nil
}

func getTaskRecurrenceSeriesByRootTaskID(
	s *xorm.Session,
	taskID int64,
) (*TaskRecurrenceSeries, error) {
	if taskID <= 0 {
		return nil, fmt.Errorf("root task id is required")
	}

	series := &TaskRecurrenceSeries{}

	has, err := s.Where("root_task_id = ?", taskID).Get(series)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}

	return series, nil
}

func validateTaskRecurrenceOccurrence(
	occurrence *TaskRecurrenceOccurrence,
) error {
	if occurrence == nil {
		return fmt.Errorf("recurrence occurrence is required")
	}

	if occurrence.SeriesID <= 0 {
		return fmt.Errorf("recurrence series id is required")
	}

	if occurrence.TaskID <= 0 {
		return fmt.Errorf("task id is required")
	}

	if occurrence.Sequence < 1 {
		return fmt.Errorf("occurrence sequence must be at least 1")
	}

	if occurrence.ScheduledDueDate.IsZero() {
		return fmt.Errorf("scheduled due date is required")
	}

	if occurrence.DueDate.IsZero() {
		return fmt.Errorf("due date is required")
	}

	return nil
}

func createTaskRecurrenceOccurrence(
	s *xorm.Session,
	occurrence *TaskRecurrenceOccurrence,
) error {
	if err := validateTaskRecurrenceOccurrence(occurrence); err != nil {
		return err
	}

	occurrence.ID = 0

	_, err := s.Insert(occurrence)
	return err
}

func getTaskRecurrenceOccurrenceByTaskID(
	s *xorm.Session,
	taskID int64,
) (*TaskRecurrenceOccurrence, error) {
	if taskID <= 0 {
		return nil, fmt.Errorf("task id is required")
	}

	occurrence := &TaskRecurrenceOccurrence{}

	has, err := s.Where("task_id = ?", taskID).Get(occurrence)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}

	return occurrence, nil
}

func updateTaskRecurrenceSeriesRootTaskID(
	s *xorm.Session,
	series *TaskRecurrenceSeries,
	taskID int64,
) error {
	if series == nil || series.ID <= 0 {
		return fmt.Errorf("recurrence series id is required")
	}
	if taskID <= 0 {
		return fmt.Errorf("root task id is required")
	}

	_, err := s.
		ID(series.ID).
		Cols("root_task_id").
		Update(&TaskRecurrenceSeries{
			RootTaskID: taskID,
		})
	if err != nil {
		return err
	}

	series.RootTaskID = taskID
	return nil
}

func updateTaskRecurrenceOccurrenceException(
	s *xorm.Session,
	occurrence *TaskRecurrenceOccurrence,
	exception bool,
	anchor time.Time,
) error {
	if occurrence == nil || occurrence.ID <= 0 {
		return fmt.Errorf("recurrence occurrence id is required")
	}

	_, err := s.
		ID(occurrence.ID).
		Cols("is_exception", "exception_anchor").
		Update(&TaskRecurrenceOccurrence{
			IsException:     exception,
			ExceptionAnchor: anchor,
		})
	if err != nil {
		return err
	}

	occurrence.IsException = exception
	occurrence.ExceptionAnchor = anchor
	return nil
}
