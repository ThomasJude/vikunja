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

// materializeTaskRecurrenceSeriesAtSession performs all currently eligible
// materialization work for one series using the caller's transaction.
func materializeTaskRecurrenceSeriesAtSession(
	s *xorm.Session,
	seriesID int64,
	reference time.Time,
) (created int, err error) {
	if s == nil {
		return 0, fmt.Errorf("database session is required")
	}
	if seriesID <= 0 {
		return 0, fmt.Errorf("recurrence series id is required")
	}
	if reference.IsZero() {
		return 0, fmt.Errorf("recurrence reference time is required")
	}

	series, err := getTaskRecurrenceSeriesByID(s, seriesID)
	if err != nil {
		return 0, err
	}

	if series.Paused {
		return 0, nil
	}

	for created < taskRecurrenceSeriesMaterializationBatchLimit {
		current, err := getLatestTaskRecurrenceOccurrenceBySeriesID(
			s,
			series.ID,
		)
		if err != nil {
			return created, err
		}

		if current == nil {
			return created, fmt.Errorf(
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
			return created, err
		}

		if occurrence == nil {
			return created, nil
		}

		created++
	}

	return created, nil
}

// materializeTaskRecurrenceSeriesAt is the production transaction wrapper.
//
// db.NewSession already creates an active transaction. Do not call Begin()
// again here.
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

	s := db.NewSession()
	defer s.Close()

	created, err = materializeTaskRecurrenceSeriesAtSession(
		s,
		seriesID,
		reference,
	)
	if err != nil {
		_ = s.Rollback()
		return created, err
	}

	if err := s.Commit(); err != nil {
		_ = s.Rollback()
		return 0, err
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
// each series in its own transaction.
func materializeDueTaskRecurrenceSeriesAt(
	reference time.Time,
) (created int, err error) {
	if reference.IsZero() {
		return 0, fmt.Errorf("recurrence reference time is required")
	}

	// Use one short transaction only for discovering active series.
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
