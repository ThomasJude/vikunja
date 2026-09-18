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

	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// TaskRecurrenceSeriesState is the task-facing representation of a recurrence
// series. Occurrence identifies where the requested task sits inside Series.
type TaskRecurrenceSeriesState struct {
	Series     *TaskRecurrenceSeries     `json:"series"`
	Occurrence *TaskRecurrenceOccurrence `json:"occurrence"`
}

func requireTaskRecurrenceSeriesRead(
	s *xorm.Session,
	taskID int64,
	a web.Auth,
) error {
	task := &Task{ID: taskID}

	can, _, err := task.CanRead(s, a)
	if err != nil {
		return err
	}
	if !can {
		return ErrGenericForbidden{}
	}

	return nil
}

// GetTaskRecurrenceSeriesState returns the series and occurrence for taskID.
// A normal, non-recurring task returns an empty state instead of an error.
func GetTaskRecurrenceSeriesState(
	s *xorm.Session,
	taskID int64,
	a web.Auth,
) (*TaskRecurrenceSeriesState, error) {
	if s == nil {
		return nil, fmt.Errorf("database session is required")
	}
	if taskID <= 0 {
		return nil, fmt.Errorf("task id is required")
	}

	if err := requireTaskRecurrenceSeriesRead(s, taskID, a); err != nil {
		return nil, err
	}

	occurrence, err := getTaskRecurrenceOccurrenceByTaskID(s, taskID)
	if err != nil {
		return nil, err
	}

	state := &TaskRecurrenceSeriesState{}
	if occurrence == nil {
		return state, nil
	}

	series, err := getTaskRecurrenceSeriesByID(s, occurrence.SeriesID)
	if err != nil {
		return nil, err
	}

	state.Series = series
	state.Occurrence = occurrence

	return state, nil
}

func normalizeTaskRecurrenceSeriesForTask(
	series *TaskRecurrenceSeries,
	task *Task,
	doerID int64,
) error {
	if series == nil {
		return fmt.Errorf("recurrence series is required")
	}
	if task == nil || task.ID <= 0 {
		return fmt.Errorf("task is required")
	}
	if doerID <= 0 {
		return fmt.Errorf("recurrence series creator is required")
	}
	if task.DueDate.IsZero() {
		return fmt.Errorf(
			"task must have a due date before a recurrence series can be created",
		)
	}

	series.RootTaskID = task.ID
	series.ProjectID = task.ProjectID
	series.CreatedByID = doerID

	// The task's due date is occurrence #1 and therefore the stable recurrence
	// anchor. The frontend may omit start_date when creating a series.
	if series.StartDate.IsZero() {
		series.StartDate = task.DueDate
	}

	if !series.StartDate.Equal(task.DueDate) {
		return fmt.Errorf(
			"recurrence start date must match the task due date",
		)
	}

	return nil
}

func removeLegacyTaskRecurrenceForSeries(
	s *xorm.Session,
	task *Task,
	a web.Auth,
) error {
	if task == nil || task.ID <= 0 {
		return fmt.Errorf("task is required")
	}

	// Load the full task so updateSingleTask keeps assignees/reminders and all
	// unrelated task state intact.
	fullTask := &Task{ID: task.ID}
	if err := fullTask.ReadOne(s, a); err != nil {
		return err
	}

	fullTask.RepeatAfter = 0
	fullTask.RepeatMode = TaskRepeatModeDefault
	fullTask.Recurrence = nil

	return fullTask.updateSingleTask(
		s,
		a,
		[]string{
			"repeat_after",
			"repeat_mode",
			"recurrence",
		},
	)
}

func countTaskRecurrenceSeriesOccurrences(
	s *xorm.Session,
	seriesID int64,
) (int64, error) {
	if seriesID <= 0 {
		return 0, fmt.Errorf("recurrence series id is required")
	}

	return s.
		Where("series_id = ?", seriesID).
		Count(&TaskRecurrenceOccurrence{})
}

// SaveTaskRecurrenceSeriesForTask creates the series for a task or replaces its
// definition while only occurrence #1 exists.
//
// Once later occurrences are materialized, schedule-definition edits require
// an explicit recurrence scope. Refusing a blind replacement here prevents
// already-created occurrences from silently retaining stale schedule dates.
func SaveTaskRecurrenceSeriesForTask(
	s *xorm.Session,
	taskID int64,
	requested *TaskRecurrenceSeries,
	a web.Auth,
) (*TaskRecurrenceSeriesState, error) {
	if s == nil {
		return nil, fmt.Errorf("database session is required")
	}
	if taskID <= 0 {
		return nil, fmt.Errorf("task id is required")
	}
	if requested == nil {
		return nil, fmt.Errorf("recurrence series is required")
	}

	if err := requireTaskRecurrenceSeriesWrite(s, taskID, a); err != nil {
		return nil, err
	}

	task, err := GetTaskByIDSimple(s, taskID)
	if err != nil {
		return nil, err
	}

	doer := doerFromAuth(s, a)
	if doer == nil || doer.ID <= 0 {
		return nil, fmt.Errorf("authenticated recurrence creator is required")
	}

	existingOccurrence, err := getTaskRecurrenceOccurrenceByTaskID(
		s,
		taskID,
	)
	if err != nil {
		return nil, err
	}

	// New series.
	if existingOccurrence == nil {
		series := *requested
		series.ID = 0

		if err := normalizeTaskRecurrenceSeriesForTask(
			&series,
			&task,
			doer.ID,
		); err != nil {
			return nil, err
		}

		if err := createTaskRecurrenceSeries(s, &series); err != nil {
			return nil, err
		}

		// The first occurrence is the task the user is currently editing.
		// Never retroactively weekend-shift it: its persisted due date is the
		// authoritative due date for occurrence #1. Policies apply when the
		// planner creates subsequent occurrences.
		rootOccurrence := &TaskRecurrenceOccurrence{
			SeriesID:         series.ID,
			TaskID:           task.ID,
			Sequence:         1,
			ScheduledDueDate: series.StartDate,
			DueDate:          task.DueDate,
		}

		if err := createTaskRecurrenceOccurrence(
			s,
			rootOccurrence,
		); err != nil {
			return nil, err
		}

		if err := removeLegacyTaskRecurrenceForSeries(
			s,
			&task,
			a,
		); err != nil {
			return nil, err
		}

		return &TaskRecurrenceSeriesState{
			Series:     &series,
			Occurrence: rootOccurrence,
		}, nil
	}

	// Existing series definition replacement.
	existing, err := getTaskRecurrenceSeriesByID(
		s,
		existingOccurrence.SeriesID,
	)
	if err != nil {
		return nil, err
	}

	count, err := countTaskRecurrenceSeriesOccurrences(
		s,
		existing.ID,
	)
	if err != nil {
		return nil, err
	}

	if count > 1 {
		return nil, fmt.Errorf(
			"recurrence series %d already has materialized future occurrences; use a scoped schedule edit",
			existing.ID,
		)
	}

	updated := *requested
	updated.ID = existing.ID
	updated.RootTaskID = existing.RootTaskID
	updated.ProjectID = existing.ProjectID
	updated.CreatedByID = existing.CreatedByID

	// An existing occurrence may have an occurrence-only due-date exception.
	// The recurrence schedule anchor therefore must not be forced to match the
	// task's current due date. If no start date was supplied, preserve the
	// existing series anchor.
	if updated.StartDate.IsZero() {
		updated.StartDate = existing.StartDate
	}

	if err := updateTaskRecurrenceSeries(s, &updated); err != nil {
		return nil, err
	}

	_, err = s.
		ID(existingOccurrence.ID).
		Cols("scheduled_due_date").
		Update(&TaskRecurrenceOccurrence{
			ScheduledDueDate: updated.StartDate,
		})
	if err != nil {
		return nil, err
	}

	existingOccurrence.ScheduledDueDate = updated.StartDate

	if err := removeLegacyTaskRecurrenceForSeries(
		s,
		&task,
		a,
	); err != nil {
		return nil, err
	}

	return &TaskRecurrenceSeriesState{
		Series:     &updated,
		Occurrence: existingOccurrence,
	}, nil
}

// SetTaskRecurrenceSeriesPausedForTask pauses or resumes the series containing
// taskID.
func SetTaskRecurrenceSeriesPausedForTask(
	s *xorm.Session,
	taskID int64,
	paused bool,
	a web.Auth,
) (*TaskRecurrenceSeriesState, error) {
	series, occurrence, err := getTaskRecurrenceSeriesForTask(s, taskID)
	if err != nil {
		return nil, err
	}

	series, err = setTaskRecurrenceSeriesPaused(
		s,
		series.ID,
		paused,
		a,
	)
	if err != nil {
		return nil, err
	}

	return &TaskRecurrenceSeriesState{
		Series:     series,
		Occurrence: occurrence,
	}, nil
}

// UpdateTaskRecurrenceScoped exposes the tested scoped task-update service to
// API routes without exposing its persistence helpers.
func UpdateTaskRecurrenceScoped(
	s *xorm.Session,
	taskID int64,
	patch *Task,
	fields []string,
	scope TaskRecurrenceEditScope,
	a web.Auth,
) (*TaskRecurrenceSeriesState, error) {
	series, err := updateTaskRecurrenceScoped(
		s,
		taskID,
		patch,
		fields,
		scope,
		a,
	)
	if err != nil {
		return nil, err
	}

	occurrence, err := getTaskRecurrenceOccurrenceByTaskID(s, taskID)
	if err != nil {
		return nil, err
	}

	// This-and-future can move the selected occurrence to the newly split
	// continuation series, so return the post-operation series.
	if occurrence != nil && occurrence.SeriesID != series.ID {
		series, err = getTaskRecurrenceSeriesByID(
			s,
			occurrence.SeriesID,
		)
		if err != nil {
			return nil, err
		}
	}

	return &TaskRecurrenceSeriesState{
		Series:     series,
		Occurrence: occurrence,
	}, nil
}

// DeleteTaskRecurrenceScoped exposes the tested scoped soft-delete service.
func DeleteTaskRecurrenceScoped(
	s *xorm.Session,
	taskID int64,
	scope TaskRecurrenceDeleteScope,
	a web.Auth,
) error {
	return deleteTaskRecurrenceScoped(
		s,
		taskID,
		scope,
		a,
	)
}
