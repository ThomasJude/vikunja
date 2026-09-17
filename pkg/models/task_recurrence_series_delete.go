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

type TaskRecurrenceDeleteScope = TaskRecurrenceEditScope

const (
	TaskRecurrenceDeleteOccurrence    TaskRecurrenceDeleteScope = TaskRecurrenceEditOccurrence
	TaskRecurrenceDeleteThisAndFuture TaskRecurrenceDeleteScope = TaskRecurrenceEditThisAndFuture
	TaskRecurrenceDeleteEntireSeries  TaskRecurrenceDeleteScope = TaskRecurrenceEditEntireSeries
)

func requireTaskRecurrenceDeletePermission(
	s *xorm.Session,
	taskID int64,
	a web.Auth,
) error {
	task := &Task{ID: taskID}

	can, err := task.CanDelete(s, a)
	if err != nil {
		return err
	}
	if !can {
		return ErrGenericForbidden{}
	}

	return nil
}

func softDeleteTaskRecurrenceOccurrence(
	s *xorm.Session,
	occurrence *TaskRecurrenceOccurrence,
	a web.Auth,
) error {
	if occurrence == nil || occurrence.TaskID <= 0 {
		return fmt.Errorf("recurrence occurrence task is required")
	}

	if err := requireTaskRecurrenceDeletePermission(
		s,
		occurrence.TaskID,
		a,
	); err != nil {
		return err
	}

	return (&Task{ID: occurrence.TaskID}).Delete(s, a)
}

func endTaskRecurrenceSeriesAtSequence(
	s *xorm.Session,
	series *TaskRecurrenceSeries,
	lastSequence int,
) error {
	if series == nil || series.ID <= 0 {
		return fmt.Errorf("recurrence series is required")
	}

	// The recurrence validator requires occurrence limits to be >= 1.
	// Setting 1 still prevents a sequence-2 task after occurrence 1 has
	// intentionally been deleted.
	if lastSequence < 1 {
		lastSequence = 1
	}

	series.EndType = TaskRecurrenceEndOccurrences
	series.EndDate = time.Time{}
	series.EndAfterOccurrences = lastSequence

	return updateTaskRecurrenceSeries(s, series)
}

func findTaskRecurrenceReplacementRoot(
	s *xorm.Session,
	seriesID int64,
	excludedTaskID int64,
) (int64, error) {
	occurrences := []*TaskRecurrenceOccurrence{}

	err := s.
		Where(
			"series_id = ? AND task_id <> ? AND is_exception = ?",
			seriesID,
			excludedTaskID,
			false,
		).
		Asc("sequence").
		Find(&occurrences)
	if err != nil {
		return 0, err
	}

	for _, occurrence := range occurrences {
		exists, err := s.
			ID(occurrence.TaskID).
			Exist(&Task{})
		if err != nil {
			return 0, err
		}
		if exists {
			return occurrence.TaskID, nil
		}
	}

	return 0, nil
}

func deleteTaskRecurrenceOccurrenceOnly(
	s *xorm.Session,
	series *TaskRecurrenceSeries,
	occurrence *TaskRecurrenceOccurrence,
	a web.Auth,
) error {
	// The materializer uses RootTaskID as the template. Never leave an active
	// series pointing at a task which will eventually be permanently deleted.
	if occurrence.TaskID == series.RootTaskID {
		replacementID, err := findTaskRecurrenceReplacementRoot(
			s,
			series.ID,
			occurrence.TaskID,
		)
		if err != nil {
			return err
		}

		if replacementID == 0 {
			return fmt.Errorf(
				"cannot delete the only recurrence template occurrence without ending the series",
			)
		}

		if err := updateTaskRecurrenceSeriesRootTaskID(
			s,
			series,
			replacementID,
		); err != nil {
			return err
		}
	}

	deletedAt := time.Now()

	if err := softDeleteTaskRecurrenceOccurrence(
		s,
		occurrence,
		a,
	); err != nil {
		return err
	}

	return updateTaskRecurrenceOccurrenceException(
		s,
		occurrence,
		true,
		deletedAt,
	)
}

func deleteTaskRecurrenceFromSequence(
	s *xorm.Session,
	series *TaskRecurrenceSeries,
	selected *TaskRecurrenceOccurrence,
	a web.Auth,
) error {
	occurrences, err := getTaskRecurrenceOccurrencesFromSequence(
		s,
		series.ID,
		selected.Sequence,
	)
	if err != nil {
		return err
	}

	// Stop generation before deleting tasks. Everything is still inside the
	// caller transaction, so any later error rolls the whole operation back.
	if err := endTaskRecurrenceSeriesAtSequence(
		s,
		series,
		selected.Sequence-1,
	); err != nil {
		return err
	}

	for _, occurrence := range occurrences {
		if err := softDeleteTaskRecurrenceOccurrence(
			s,
			occurrence,
			a,
		); err != nil {
			return err
		}

		if err := updateTaskRecurrenceOccurrenceException(
			s,
			occurrence,
			true,
			time.Time{},
		); err != nil {
			return err
		}
	}

	return nil
}

func deleteEntireTaskRecurrenceSeries(
	s *xorm.Session,
	series *TaskRecurrenceSeries,
	a web.Auth,
) error {
	occurrences, err := getAllTaskRecurrenceOccurrences(s, series.ID)
	if err != nil {
		return err
	}
	if len(occurrences) == 0 {
		return fmt.Errorf(
			"recurrence series %d has no occurrences",
			series.ID,
		)
	}

	lastSequence := occurrences[len(occurrences)-1].Sequence

	if err := endTaskRecurrenceSeriesAtSequence(
		s,
		series,
		lastSequence,
	); err != nil {
		return err
	}

	for _, occurrence := range occurrences {
		if err := softDeleteTaskRecurrenceOccurrence(
			s,
			occurrence,
			a,
		); err != nil {
			return err
		}

		if err := updateTaskRecurrenceOccurrenceException(
			s,
			occurrence,
			true,
			time.Time{},
		); err != nil {
			return err
		}
	}

	return nil
}

// deleteTaskRecurrenceScoped soft-deletes recurring task occurrences using
// Vikunja's native Task.Delete path while changing the recurrence definition
// atomically in the same transaction.
func deleteTaskRecurrenceScoped(
	s *xorm.Session,
	taskID int64,
	scope TaskRecurrenceDeleteScope,
	a web.Auth,
) error {
	if s == nil {
		return fmt.Errorf("database session is required")
	}

	if err := validateTaskRecurrenceEditScope(scope); err != nil {
		return err
	}

	series, occurrence, err := getTaskRecurrenceSeriesForTask(s, taskID)
	if err != nil {
		return err
	}

	if err := requireTaskRecurrenceDeletePermission(
		s,
		taskID,
		a,
	); err != nil {
		return err
	}

	switch scope {
	case TaskRecurrenceDeleteOccurrence:
		return deleteTaskRecurrenceOccurrenceOnly(
			s,
			series,
			occurrence,
			a,
		)

	case TaskRecurrenceDeleteThisAndFuture:
		return deleteTaskRecurrenceFromSequence(
			s,
			series,
			occurrence,
			a,
		)

	case TaskRecurrenceDeleteEntireSeries:
		return deleteEntireTaskRecurrenceSeries(
			s,
			series,
			a,
		)

	default:
		return fmt.Errorf(
			"invalid recurrence delete scope: %d",
			scope,
		)
	}
}
