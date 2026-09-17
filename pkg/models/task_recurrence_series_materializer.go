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

	plan, err := planNextTaskRecurrenceSeriesOccurrence(
		series,
		current,
		reference,
	)
	if err != nil {
		return nil, err
	}

	// Paused or ended series have no next occurrence.
	if plan == nil {
		return nil, nil
	}

	// Do not create the task before its configured creation time.
	if plan.CreateAt.After(reference) {
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

	// The planner is authoritative, including weekend adjustments.
	newTask.DueDate = plan.DueDate

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
