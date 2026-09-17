// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package apiv2

import (
	"context"
	"net/http"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"

	"github.com/danielgtaylor/huma/v2"
)

type taskRecurrencePauseRequest struct {
	Paused bool `json:"paused" doc:"True pauses future materialization; false resumes the series."`
}

type taskRecurrenceScopedUpdateRequest struct {
	Scope  models.TaskRecurrenceEditScope `json:"scope" doc:"0=this occurrence, 1=this and future, 2=entire series."`
	Fields []string                       `json:"fields" doc:"Task fields to update."`
	Task   models.Task                    `json:"task" doc:"Values for the named fields."`
}

type taskRecurrenceDeleteResult struct {
	Message string `json:"message"`
}

func RegisterTaskRecurrenceSeriesRoutes(api huma.API) {
	tags := []string{"tasks"}

	Register(api, huma.Operation{
		OperationID: "tasks-recurrence-series-read",
		Summary:     "Get a task recurrence series",
		Description: "Returns the recurrence series and occurrence metadata for a task. A normal non-recurring task returns null series and occurrence values.",
		Method:      http.MethodGet,
		Path:        "/tasks/{task}/recurrence-series",
		Tags:        tags,
	}, taskRecurrenceSeriesRead)

	Register(api, huma.Operation{
		OperationID: "tasks-recurrence-series-save",
		Summary:     "Create or update a task recurrence series",
		Description: "Creates a recurrence series using the task due date as occurrence one, or replaces the definition while no later occurrence has been materialized.",
		Method:      http.MethodPut,
		Path:        "/tasks/{task}/recurrence-series",
		Tags:        tags,
	}, taskRecurrenceSeriesSave)

	Register(api, huma.Operation{
		OperationID: "tasks-recurrence-series-pause",
		Summary:     "Pause or resume a task recurrence series",
		Description: "Pauses or resumes future recurrence materialization without changing the recurrence definition.",
		Method:      http.MethodPatch,
		Path:        "/tasks/{task}/recurrence-series/pause",
		Tags:        tags,
	}, taskRecurrenceSeriesPause)

	Register(api, huma.Operation{
		OperationID: "tasks-recurrence-series-scoped-update",
		Summary:     "Update recurring task occurrences with a scope",
		Description: "Updates only this occurrence, this and future occurrences, or all materialized occurrences in the series.",
		Method:      http.MethodPatch,
		Path:        "/tasks/{task}/recurrence-series/task",
		Tags:        tags,
	}, taskRecurrenceSeriesScopedUpdate)

	Register(api, huma.Operation{
		OperationID: "tasks-recurrence-series-delete",
		Summary:     "Delete recurring task occurrences with a scope",
		Description: "Soft-deletes this occurrence, this and future occurrences, or the entire recurring series using Vikunja's normal task deletion path.",
		Method:      http.MethodDelete,
		Path:        "/tasks/{task}/recurrence-series",
		Tags:        tags,
	}, taskRecurrenceSeriesDelete)
}

func init() {
	AddRouteRegistrar(RegisterTaskRecurrenceSeriesRoutes)
}

func taskRecurrenceSeriesRead(ctx context.Context, in *struct {
	TaskID int64 `path:"task" doc:"The numeric id of the task."`
}) (*singleBody[models.TaskRecurrenceSeriesState], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s := db.NewSession()
	defer s.Close()

	state, err := models.GetTaskRecurrenceSeriesState(
		s,
		in.TaskID,
		a,
	)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	if err := s.Commit(); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	return &singleBody[models.TaskRecurrenceSeriesState]{
		Body: state,
	}, nil
}

func taskRecurrenceSeriesSave(ctx context.Context, in *struct {
	TaskID int64 `path:"task" doc:"The numeric id of the task."`
	Body   models.TaskRecurrenceSeries
}) (*singleBody[models.TaskRecurrenceSeriesState], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s := db.NewSession()
	defer s.Close()

	state, err := models.SaveTaskRecurrenceSeriesForTask(
		s,
		in.TaskID,
		&in.Body,
		a,
	)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	if err := s.Commit(); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	return &singleBody[models.TaskRecurrenceSeriesState]{
		Body: state,
	}, nil
}

func taskRecurrenceSeriesPause(ctx context.Context, in *struct {
	TaskID int64 `path:"task" doc:"The numeric id of the task."`
	Body   taskRecurrencePauseRequest
}) (*singleBody[models.TaskRecurrenceSeriesState], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s := db.NewSession()
	defer s.Close()

	state, err := models.SetTaskRecurrenceSeriesPausedForTask(
		s,
		in.TaskID,
		in.Body.Paused,
		a,
	)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	if err := s.Commit(); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	return &singleBody[models.TaskRecurrenceSeriesState]{
		Body: state,
	}, nil
}

func taskRecurrenceSeriesScopedUpdate(ctx context.Context, in *struct {
	TaskID int64 `path:"task" doc:"The numeric id of the task."`
	Body   taskRecurrenceScopedUpdateRequest
}) (*singleBody[models.TaskRecurrenceSeriesState], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s := db.NewSession()
	defer s.Close()

	state, err := models.UpdateTaskRecurrenceScoped(
		s,
		in.TaskID,
		&in.Body.Task,
		in.Body.Fields,
		in.Body.Scope,
		a,
	)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	if err := s.Commit(); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	return &singleBody[models.TaskRecurrenceSeriesState]{
		Body: state,
	}, nil
}

func taskRecurrenceSeriesDelete(ctx context.Context, in *struct {
	TaskID int64 `path:"task" doc:"The numeric id of the task."`
	Scope  int   `query:"scope" minimum:"0" maximum:"2" doc:"0=this occurrence, 1=this and future, 2=entire series."`
}) (*singleBody[taskRecurrenceDeleteResult], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s := db.NewSession()
	defer s.Close()

	err = models.DeleteTaskRecurrenceScoped(
		s,
		in.TaskID,
		models.TaskRecurrenceDeleteScope(in.Scope),
		a,
	)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	if err := s.Commit(); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	return &singleBody[taskRecurrenceDeleteResult]{
		Body: &taskRecurrenceDeleteResult{
			Message: "success",
		},
	}, nil
}
