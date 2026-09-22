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

	"code.vikunja.io/api/pkg/user"

	"xorm.io/xorm"
)

func getTaskRecurrenceOccurrenceBySeriesAndSequence(
	s *xorm.Session,
	seriesID int64,
	sequence int,
) (*TaskRecurrenceOccurrence, error) {
	if seriesID <= 0 {
		return nil, fmt.Errorf("recurrence series id is required")
	}
	if sequence < 1 {
		return nil, fmt.Errorf("occurrence sequence must be at least 1")
	}

	occurrence := &TaskRecurrenceOccurrence{}
	has, err := s.
		Where("series_id = ? AND sequence = ?", seriesID, sequence).
		Get(occurrence)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}

	return occurrence, nil
}

func copyTaskRecurrenceSeriesLabels(
	s *xorm.Session,
	rootTaskID int64,
	newTaskID int64,
) error {
	labelTasks := []*LabelTask{}

	err := s.Where("task_id = ?", rootTaskID).Find(&labelTasks)
	if err != nil {
		return err
	}

	for _, labelTask := range labelTasks {
		labelTask.ID = 0
		labelTask.TaskID = newTaskID
	}

	if len(labelTasks) == 0 {
		return nil
	}

	_, err = s.Insert(&labelTasks)
	return err
}

// planTaskRecurrenceSeriesOccurrenceForMaterialization resolves the next
// occurrence using the correct recurrence anchor and checks whether it is
// eligible to be created at reference.
//
// Schedule-based recurrence uses reference as the planner's catch-up point.
// Completion-based recurrence must use the actual DoneAt timestamp of the
// current occurrence task as its recurrence anchor. This prevents cron time
// from being mistaken for task completion time.
func planTaskRecurrenceSeriesOccurrenceForMaterialization(
	s *xorm.Session,
	series *TaskRecurrenceSeries,
	current *TaskRecurrenceOccurrence,
	reference time.Time,
) (*TaskRecurrenceSeriesOccurrencePlan, error) {
	if series == nil {
		return nil, fmt.Errorf("recurrence series is required")
	}
	if current == nil {
		return nil, fmt.Errorf("current recurrence occurrence is required")
	}
	if reference.IsZero() {
		return nil, fmt.Errorf("recurrence reference time is required")
	}

	// A series explicitly truncated at the current occurrence cannot produce
	// another occurrence. Check this before loading a completion-basis task,
	// because that task may have intentionally been soft-deleted.
	if series.EndType == TaskRecurrenceEndOccurrences &&
		current.Sequence >= series.EndAfterOccurrences {
		return nil, nil
	}

	var plannerReference time.Time

	switch series.Basis {
	case TaskRecurrenceBasisSchedule:
		plannerReference = reference

	case TaskRecurrenceBasisCompletion:
		// A deliberately deleted occurrence behaves like a skipped completion.
		// Its persisted exception anchor keeps recurrence stable even after the
		// soft-deleted task is permanently cleaned up.
		if current.IsException && !current.ExceptionAnchor.IsZero() {
			plannerReference = current.ExceptionAnchor
			break
		}

		currentTask, err := GetTaskByIDSimple(s, current.TaskID)
		if err != nil {
			return nil, err
		}

		// Repeat After Completion must never advance merely because time has
		// passed. The current occurrence has to be actually completed first.
		if !currentTask.Done || currentTask.DoneAt.IsZero() {
			return nil, nil
		}

		plannerReference = currentTask.DoneAt

	default:
		return nil, fmt.Errorf(
			"invalid recurrence basis: %d",
			series.Basis,
		)
	}

	plan, err := planNextTaskRecurrenceSeriesOccurrence(
		series,
		current,
		plannerReference,
	)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, nil
	}

	// The recurrence may already be known after completion but still have a
	// future CreateAt time. Cron will revisit the series when that time arrives.
	if plan.CreateAt.After(reference) {
		return nil, nil
	}

	return plan, nil
}

// materializeTaskRecurrenceSeriesOccurrence creates the next occurrence once
// its CreateAt time has been reached.
//
// The caller must use a transactional session so task creation and occurrence
// persistence are committed or rolled back together.
func materializeTaskRecurrenceSeriesOccurrence(
	s *xorm.Session,
	series *TaskRecurrenceSeries,
	current *TaskRecurrenceOccurrence,
	reference time.Time,
) (*TaskRecurrenceOccurrence, error) {
	if series == nil {
		return nil, fmt.Errorf("recurrence series is required")
	}
	if series.ID <= 0 {
		return nil, fmt.Errorf("persisted recurrence series id is required")
	}
	if current == nil {
		return nil, fmt.Errorf("current recurrence occurrence is required")
	}
	if current.SeriesID != series.ID {
		return nil, fmt.Errorf(
			"current occurrence belongs to recurrence series %d, expected %d",
			current.SeriesID,
			series.ID,
		)
	}
	if reference.IsZero() {
		return nil, fmt.Errorf("recurrence reference time is required")
	}

	plan, err := planTaskRecurrenceSeriesOccurrenceForMaterialization(
		s,
		series,
		current,
		reference,
	)
	if err != nil {
		return nil, err
	}

	// Paused, ended, not-yet-completed, or not-yet-creatable series have no
	// occurrence to materialize at this time.
	if plan == nil {
		return nil, nil
	}

	// Normal repeated calls should return the already materialized occurrence.
	existing, err := getTaskRecurrenceOccurrenceBySeriesAndSequence(
		s,
		series.ID,
		plan.Sequence,
	)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	doer, err := user.GetUserByID(s, series.CreatedByID)
	if err != nil {
		return nil, err
	}

	// Always use the series root task as the template. An occurrence-only edit
	// must not leak into later generated occurrences.
	rootTask := &Task{ID: series.RootTaskID}
	if err := rootTask.ReadOne(s, doer); err != nil {
		return nil, err
	}

	if rootTask.ProjectID != series.ProjectID {
		return nil, fmt.Errorf(
			"recurrence series project %d does not match root task project %d",
			series.ProjectID,
			rootTask.ProjectID,
		)
	}

	rootOccurrence, err := getTaskRecurrenceOccurrenceByTaskID(
		s,
		series.RootTaskID,
	)
	if err != nil {
		return nil, err
	}
	if rootOccurrence == nil {
		return nil, fmt.Errorf(
			"root task %d has no recurrence occurrence",
			series.RootTaskID,
		)
	}
	if rootOccurrence.SeriesID != series.ID {
		return nil, fmt.Errorf(
			"root occurrence belongs to recurrence series %d, expected %d",
			rootOccurrence.SeriesID,
			series.ID,
		)
	}

	newTask := &Task{
		Title:       rootTask.Title,
		Description: resetDescriptionChecklist(rootTask.Description),
		Done:        false,
		ProjectID:   rootTask.ProjectID,
		Priority:    rootTask.Priority,
		HexColor:    rootTask.HexColor,
		PercentDone: 0,
		Assignees:   rootTask.Assignees,
	}

	// Preserve start/end/reminder offsets using Vikunja's existing recurrence
	// shifting logic.
	shiftTaskDatesForRecurrence(
		rootTask,
		newTask,
		rootOccurrence.DueDate,
		plan.DueDate,
	)

	// Preserve whether this recurring task series actually uses task due dates.
	// The recurrence calendar itself is tracked by occurrence metadata.
	if rootTask.DueDate.IsZero() {
		newTask.DueDate = time.Time{}
	} else {
		// The planner is authoritative, including weekend adjustments.
		newTask.DueDate = plan.DueDate
	}

	// Generated series occurrences must not trigger Vikunja's old in-place
	// repeat mechanism when completed.
	newTask.RepeatAfter = 0
	newTask.Recurrence = nil

	if err := createTask(s, newTask, doer, true, true); err != nil {
		return nil, err
	}

	// Preserve labels but intentionally do not duplicate attachments or
	// copied-from relations.
	if err := copyTaskRecurrenceSeriesLabels(
		s,
		rootTask.ID,
		newTask.ID,
	); err != nil {
		return nil, err
	}

	occurrence := &TaskRecurrenceOccurrence{
		SeriesID:         series.ID,
		TaskID:           newTask.ID,
		Sequence:         plan.Sequence,
		ScheduledDueDate: plan.ScheduledDueDate,
		DueDate:          plan.DueDate,
	}

	if err := createTaskRecurrenceOccurrence(s, occurrence); err != nil {
		return nil, err
	}

	return occurrence, nil
}
