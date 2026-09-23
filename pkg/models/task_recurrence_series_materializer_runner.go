// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"errors"
	"fmt"
	"time"

	"code.vikunja.io/api/pkg/db"

	"xorm.io/xorm"
)

const taskRecurrenceSeriesMaterializationBatchLimit = 100

const taskRecurrenceSeriesSequenceConstraint = "UQE_task_recurrence_occurrences_series_sequence"

func getLatestTaskRecurrenceOccurrenceBySeriesID(
	s *xorm.Session,
	seriesID int64,
) (*TaskRecurrenceOccurrence, error) {
	if seriesID <= 0 {
		return nil, fmt.Errorf("recurrence series id is required")
	}

	occurrence := &TaskRecurrenceOccurrence{}

	has, err := s.
		Where("series_id = ?", seriesID).
		Desc("sequence").
		Get(occurrence)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}

	return occurrence, nil
}

func isTaskRecurrenceSeriesSequenceConflict(err error) bool {
	if err == nil {
		return false
	}

	// Production MariaDB/PostgreSQL use the named migration constraint.
	if db.IsUniqueConstraintError(
		err,
		taskRecurrenceSeriesSequenceConstraint,
	) {
		return true
	}

	// Vikunja's unit-test database reports SQLite unique failures as
	// table.column pairs rather than the migration's index name.
	return db.IsUniqueConstraintError(
		err,
		"task_recurrence_occurrences.series_id",
	)
}

// materializeTaskRecurrenceSeriesOnceAtSession performs at most one
// materialization attempt using the caller's active transaction.
func materializeTaskRecurrenceSeriesOnceAtSession(
	s *xorm.Session,
	seriesID int64,
	reference time.Time,
) (created bool, err error) {
	if s == nil {
		return false, fmt.Errorf("database session is required")
	}
	if seriesID <= 0 {
		return false, fmt.Errorf("recurrence series id is required")
	}
	if reference.IsZero() {
		return false, fmt.Errorf("recurrence reference time is required")
	}

	series, err := getTaskRecurrenceSeriesByID(s, seriesID)
	if err != nil {
		return false, err
	}

	if series.Paused {
		return false, nil
	}

	current, err := getLatestTaskRecurrenceOccurrenceBySeriesID(
		s,
		series.ID,
	)
	if err != nil {
		return false, err
	}

	if current == nil {
		return false, fmt.Errorf(
			"recurrence series %d has no current occurrence",
			series.ID,
		)
	}

	occurrence, err := materializeTaskRecurrenceSeriesOccurrence(
		s,
		series,
		current,
		reference,
	)
	if err != nil {
		return false, err
	}

	return occurrence != nil, nil
}

// materializeTaskRecurrenceSeriesAtSession runs materialization using the
// caller's transaction.
func materializeTaskRecurrenceSeriesAtSession(
	s *xorm.Session,
	seriesID int64,
	reference time.Time,
) (created int, err error) {
	for created < taskRecurrenceSeriesMaterializationBatchLimit {
		didCreate, err := materializeTaskRecurrenceSeriesOnceAtSession(
			s,
			seriesID,
			reference,
		)
		if err != nil {
			return created, err
		}

		if !didCreate {
			return created, nil
		}

		created++
	}

	return created, nil
}

// materializeTaskRecurrenceSeriesAt is the production runner.
//
// Every generated occurrence gets its own transaction. This keeps locks short,
// makes rollback atomic for task + occurrence creation, and prevents a large
// missed-occurrence catch-up from holding one long-running transaction.
func materializeTaskRecurrenceSeriesAt(
	seriesID int64,
	reference time.Time,
) (created int, err error) {
	if seriesID <= 0 {
		return 0, fmt.Errorf("recurrence series id is required")
	}
	if reference.IsZero() {
		return 0, fmt.Errorf("recurrence reference time is required")
	}

	for created < taskRecurrenceSeriesMaterializationBatchLimit {
		s := db.NewSession()

		didCreate, materializeErr := materializeTaskRecurrenceSeriesOnceAtSession(
			s,
			seriesID,
			reference,
		)

		if materializeErr != nil {
			_ = s.Rollback()
			_ = s.Close()

			// Another worker may have materialized the same sequence between
			// our idempotency lookup and insert. Its transaction wins; ours
			// rolls back the newly created task and exits quietly.
			if isTaskRecurrenceSeriesSequenceConflict(materializeErr) {
				return created, nil
			}

			return created, materializeErr
		}

		if commitErr := s.Commit(); commitErr != nil {
			_ = s.Rollback()
			_ = s.Close()

			if isTaskRecurrenceSeriesSequenceConflict(commitErr) {
				return created, nil
			}

			return created, commitErr
		}

		if closeErr := s.Close(); closeErr != nil {
			return created, closeErr
		}

		if !didCreate {
			return created, nil
		}

		created++
	}

	return created, nil
}

func getActiveTaskRecurrenceSeriesIDs(
	s *xorm.Session,
) ([]int64, error) {
	series := []*TaskRecurrenceSeries{}

	err := s.
		Cols("id").
		Where("paused = ?", false).
		OrderBy("id").
		Find(&series)
	if err != nil {
		return nil, err
	}

	ids := make([]int64, 0, len(series))
	for _, recurrenceSeries := range series {
		ids = append(ids, recurrenceSeries.ID)
	}

	return ids, nil
}

// materializeDueTaskRecurrenceSeriesAt scans active series and materializes
// each series independently. An error in one series does not block others.
func materializeDueTaskRecurrenceSeriesAt(
	reference time.Time,
) (created int, err error) {
	if reference.IsZero() {
		return 0, fmt.Errorf("recurrence reference time is required")
	}

	scan := db.NewSession()

	seriesIDs, scanErr := getActiveTaskRecurrenceSeriesIDs(scan)
	if scanErr != nil {
		_ = scan.Rollback()
		_ = scan.Close()
		return 0, scanErr
	}

	if commitErr := scan.Commit(); commitErr != nil {
		_ = scan.Rollback()
		_ = scan.Close()
		return 0, commitErr
	}

	if closeErr := scan.Close(); closeErr != nil {
		return 0, closeErr
	}

	var errs []error

	for _, seriesID := range seriesIDs {
		count, materializeErr := materializeTaskRecurrenceSeriesAt(
			seriesID,
			reference,
		)

		created += count

		if materializeErr != nil {
			errs = append(
				errs,
				fmt.Errorf(
					"recurrence series %d: %w",
					seriesID,
					materializeErr,
				),
			)
		}
	}

	return created, errors.Join(errs...)
}
