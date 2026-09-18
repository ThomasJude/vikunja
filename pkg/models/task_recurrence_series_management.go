// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"fmt"
	"time"

	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// TaskRecurrenceEditScope controls how an edit to a generated recurring task
// propagates through its series.
type TaskRecurrenceEditScope int

const (
	TaskRecurrenceEditOccurrence TaskRecurrenceEditScope = iota
	TaskRecurrenceEditThisAndFuture
	TaskRecurrenceEditEntireSeries
)

func validateTaskRecurrenceEditScope(scope TaskRecurrenceEditScope) error {
	switch scope {
	case TaskRecurrenceEditOccurrence,
		TaskRecurrenceEditThisAndFuture,
		TaskRecurrenceEditEntireSeries:
		return nil
	default:
		return fmt.Errorf("invalid recurrence edit scope: %d", scope)
	}
}

func getTaskRecurrenceSeriesForTask(
	s *xorm.Session,
	taskID int64,
) (*TaskRecurrenceSeries, *TaskRecurrenceOccurrence, error) {
	if taskID <= 0 {
		return nil, nil, fmt.Errorf("task id is required")
	}

	occurrence, err := getTaskRecurrenceOccurrenceByTaskID(s, taskID)
	if err != nil {
		return nil, nil, err
	}
	if occurrence == nil {
		return nil, nil, fmt.Errorf("task %d is not part of a recurrence series", taskID)
	}

	series, err := getTaskRecurrenceSeriesByID(s, occurrence.SeriesID)
	if err != nil {
		return nil, nil, err
	}

	return series, occurrence, nil
}

func requireTaskRecurrenceSeriesWrite(
	s *xorm.Session,
	taskID int64,
	a web.Auth,
) error {
	task := &Task{ID: taskID}

	can, err := task.CanUpdate(s, a)
	if err != nil {
		return err
	}
	if !can {
		return ErrGenericForbidden{}
	}

	return nil
}

// RemoveTaskRecurrenceSeriesForTask removes the recurrence definition and
// occurrence metadata without deleting any tasks. Existing task occurrences
// become ordinary standalone tasks and no future occurrences are generated.
func RemoveTaskRecurrenceSeriesForTask(
	s *xorm.Session,
	taskID int64,
	a web.Auth,
) error {
	if s == nil {
		return fmt.Errorf("database session is required")
	}

	series, _, err := getTaskRecurrenceSeriesForTask(s, taskID)
	if err != nil {
		return err
	}

	// Removing recurrence changes the series configuration rather than
	// deleting a task, so normal series write permission is sufficient.
	if err := requireTaskRecurrenceSeriesWrite(
		s,
		series.RootTaskID,
		a,
	); err != nil {
		return err
	}

	// Remove occurrence metadata first. The actual tasks are intentionally
	// left untouched and therefore become normal standalone tasks.
	if _, err := s.
		Where("series_id = ?", series.ID).
		Delete(&TaskRecurrenceOccurrence{}); err != nil {
		return fmt.Errorf(
			"could not remove recurrence occurrence metadata: %w",
			err,
		)
	}

	affected, err := s.
		ID(series.ID).
		Delete(&TaskRecurrenceSeries{})
	if err != nil {
		return fmt.Errorf(
			"could not remove recurrence series: %w",
			err,
		)
	}

	if affected != 1 {
		return fmt.Errorf(
			"recurrence series %d could not be removed",
			series.ID,
		)
	}

	return nil
}

// setTaskRecurrenceSeriesPaused changes only the execution state of a series.
// The recurrence definition itself remains untouched.
func setTaskRecurrenceSeriesPaused(
	s *xorm.Session,
	seriesID int64,
	paused bool,
	a web.Auth,
) (*TaskRecurrenceSeries, error) {
	series, err := getTaskRecurrenceSeriesByID(s, seriesID)
	if err != nil {
		return nil, err
	}

	if err := requireTaskRecurrenceSeriesWrite(s, series.RootTaskID, a); err != nil {
		return nil, err
	}

	if series.Paused == paused {
		return series, nil
	}

	series.Paused = paused

	if err := updateTaskRecurrenceSeries(s, series); err != nil {
		return nil, err
	}

	return series, nil
}

func markTaskRecurrenceOccurrenceException(
	s *xorm.Session,
	occurrenceID int64,
	exception bool,
) error {
	if occurrenceID <= 0 {
		return fmt.Errorf("recurrence occurrence id is required")
	}

	_, err := s.
		ID(occurrenceID).
		Cols("is_exception").
		Update(&TaskRecurrenceOccurrence{
			IsException: exception,
		})

	return err
}

func getTaskRecurrenceOccurrencesFromSequence(
	s *xorm.Session,
	seriesID int64,
	sequence int,
) ([]*TaskRecurrenceOccurrence, error) {
	if seriesID <= 0 {
		return nil, fmt.Errorf("recurrence series id is required")
	}
	if sequence < 1 {
		return nil, fmt.Errorf("recurrence sequence must be at least 1")
	}

	occurrences := []*TaskRecurrenceOccurrence{}

	err := s.
		Where("series_id = ? AND sequence >= ?", seriesID, sequence).
		Asc("sequence").
		Find(&occurrences)
	if err != nil {
		return nil, err
	}

	return occurrences, nil
}

func getAllTaskRecurrenceOccurrences(
	s *xorm.Session,
	seriesID int64,
) ([]*TaskRecurrenceOccurrence, error) {
	return getTaskRecurrenceOccurrencesFromSequence(s, seriesID, 1)
}

// splitTaskRecurrenceSeriesAtOccurrence creates a new continuation series whose
// root is the selected occurrence. Occurrences before the split remain in the
// old series and selected/later materialized occurrences move to the new one.
//
// Splitting sequence 1 is a no-op because there is no historical prefix to
// preserve.
func splitTaskRecurrenceSeriesAtOccurrence(
	s *xorm.Session,
	series *TaskRecurrenceSeries,
	selected *TaskRecurrenceOccurrence,
) (*TaskRecurrenceSeries, error) {
	if series == nil || series.ID <= 0 {
		return nil, fmt.Errorf("recurrence series is required")
	}
	if selected == nil || selected.ID <= 0 {
		return nil, fmt.Errorf("recurrence occurrence is required")
	}
	if selected.SeriesID != series.ID {
		return nil, fmt.Errorf(
			"recurrence occurrence %d does not belong to series %d",
			selected.ID,
			series.ID,
		)
	}

	if selected.Sequence == 1 {
		return series, nil
	}

	oldEndType := series.EndType
	oldEndAfterOccurrences := series.EndAfterOccurrences

	continuation := *series
	continuation.ID = 0
	continuation.RootTaskID = selected.TaskID
	continuation.StartDate = selected.ScheduledDueDate
	continuation.Created = time.Time{}
	continuation.Updated = time.Time{}

	if oldEndType == TaskRecurrenceEndOccurrences {
		remaining := oldEndAfterOccurrences - selected.Sequence + 1
		if remaining < 1 {
			return nil, fmt.Errorf(
				"recurrence split sequence %d is beyond occurrence limit %d",
				selected.Sequence,
				oldEndAfterOccurrences,
			)
		}
		continuation.EndAfterOccurrences = remaining
	}

	if err := createTaskRecurrenceSeries(s, &continuation); err != nil {
		return nil, err
	}

	// The old series ends immediately before the selected occurrence.
	series.EndType = TaskRecurrenceEndOccurrences
	series.EndAfterOccurrences = selected.Sequence - 1

	if err := updateTaskRecurrenceSeries(s, series); err != nil {
		return nil, err
	}

	toMove, err := getTaskRecurrenceOccurrencesFromSequence(
		s,
		series.ID,
		selected.Sequence,
	)
	if err != nil {
		return nil, err
	}

	for _, occurrence := range toMove {
		newSequence := occurrence.Sequence - selected.Sequence + 1

		_, err := s.
			ID(occurrence.ID).
			Cols("series_id", "sequence").
			Update(&TaskRecurrenceOccurrence{
				SeriesID: continuation.ID,
				Sequence: newSequence,
			})
		if err != nil {
			return nil, err
		}
	}

	return &continuation, nil
}

var recurrenceOccurrenceOnlyTaskFields = map[string]bool{
	"title":        true,
	"description":  true,
	"done":         true,
	"due_date":     true,
	"priority":     true,
	"start_date":   true,
	"end_date":     true,
	"hex_color":    true,
	"percent_done": true,
}

var recurrencePropagatedTaskFields = map[string]bool{
	"title":        true,
	"description":  true,
	"priority":     true,
	"hex_color":    true,
	"percent_done": true,
}

func validateTaskRecurrenceScopedFields(
	fields []string,
	scope TaskRecurrenceEditScope,
) error {
	if len(fields) == 0 {
		return fmt.Errorf("at least one task field is required")
	}

	allowed := recurrenceOccurrenceOnlyTaskFields
	if scope != TaskRecurrenceEditOccurrence {
		allowed = recurrencePropagatedTaskFields
	}

	for _, field := range fields {
		if !allowed[field] {
			return fmt.Errorf(
				"task field %q cannot be updated with recurrence scope %d",
				field,
				scope,
			)
		}
	}

	return nil
}

func applyTaskRecurrencePatchField(
	target *Task,
	patch *Task,
	field string,
) {
	switch field {
	case "title":
		target.Title = patch.Title
	case "description":
		target.Description = patch.Description
	case "done":
		target.Done = patch.Done
	case "due_date":
		target.DueDate = patch.DueDate
	case "priority":
		target.Priority = patch.Priority
	case "start_date":
		target.StartDate = patch.StartDate
	case "end_date":
		target.EndDate = patch.EndDate
	case "hex_color":
		target.HexColor = patch.HexColor
	case "percent_done":
		target.PercentDone = patch.PercentDone
	}
}

// updateTaskRecurrenceOccurrenceFields applies a normal Vikunja task update,
// preserving all unrelated task state and using the native update path.
func updateTaskRecurrenceOccurrenceFields(
	s *xorm.Session,
	taskID int64,
	patch *Task,
	fields []string,
	a web.Auth,
) error {
	stored := &Task{ID: taskID}
	if err := stored.ReadOne(s, a); err != nil {
		return err
	}

	for _, field := range fields {
		applyTaskRecurrencePatchField(stored, patch, field)
	}

	return stored.updateSingleTask(s, a, fields)
}

// updateTaskRecurrenceScoped updates materialized task properties according to
// the selected recurrence scope.
//
// Recurrence rule/schedule changes are intentionally separate from this helper.
// Propagated task fields are limited to template-safe values so we never assign
// the same absolute due/start/end date to every occurrence.
func updateTaskRecurrenceScoped(
	s *xorm.Session,
	taskID int64,
	patch *Task,
	fields []string,
	scope TaskRecurrenceEditScope,
	a web.Auth,
) (*TaskRecurrenceSeries, error) {
	if s == nil {
		return nil, fmt.Errorf("database session is required")
	}
	if patch == nil {
		return nil, fmt.Errorf("task update is required")
	}
	if err := validateTaskRecurrenceEditScope(scope); err != nil {
		return nil, err
	}
	if err := validateTaskRecurrenceScopedFields(fields, scope); err != nil {
		return nil, err
	}

	series, selected, err := getTaskRecurrenceSeriesForTask(s, taskID)
	if err != nil {
		return nil, err
	}

	if err := requireTaskRecurrenceSeriesWrite(s, taskID, a); err != nil {
		return nil, err
	}

	switch scope {
	case TaskRecurrenceEditOccurrence:
		if err := updateTaskRecurrenceOccurrenceFields(
			s,
			taskID,
			patch,
			fields,
			a,
		); err != nil {
			return nil, err
		}

		if err := markTaskRecurrenceOccurrenceException(
			s,
			selected.ID,
			true,
		); err != nil {
			return nil, err
		}

		return series, nil

	case TaskRecurrenceEditThisAndFuture:
		series, err = splitTaskRecurrenceSeriesAtOccurrence(
			s,
			series,
			selected,
		)
		if err != nil {
			return nil, err
		}

		occurrences, err := getAllTaskRecurrenceOccurrences(s, series.ID)
		if err != nil {
			return nil, err
		}

		for _, occurrence := range occurrences {
			if err := updateTaskRecurrenceOccurrenceFields(
				s,
				occurrence.TaskID,
				patch,
				fields,
				a,
			); err != nil {
				return nil, err
			}
		}

		return series, nil

	case TaskRecurrenceEditEntireSeries:
		occurrences, err := getAllTaskRecurrenceOccurrences(s, series.ID)
		if err != nil {
			return nil, err
		}

		for _, occurrence := range occurrences {
			if err := updateTaskRecurrenceOccurrenceFields(
				s,
				occurrence.TaskID,
				patch,
				fields,
				a,
			); err != nil {
				return nil, err
			}
		}

		return series, nil
	}

	return nil, fmt.Errorf("invalid recurrence edit scope: %d", scope)
}
