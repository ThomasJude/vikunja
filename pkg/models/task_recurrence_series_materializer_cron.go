// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"time"

	"code.vikunja.io/api/pkg/cron"
	"code.vikunja.io/api/pkg/log"
)

// RegisterTaskRecurrenceSeriesMaterializerCron registers the recurring-series
// materializer. It runs once per minute so CreateAt times are acted on without
// introducing a separate scheduler.
func RegisterTaskRecurrenceSeriesMaterializerCron() {
	err := cron.Schedule("* * * * *", func() {
		created, err := materializeDueTaskRecurrenceSeriesAt(time.Now())
		if err != nil {
			log.Errorf(
				"[Task Recurrence Series Cron] Could not materialize recurring task series: %s",
				err,
			)
			return
		}

		if created > 0 {
			log.Debugf(
				"[Task Recurrence Series Cron] Materialized %d recurring task occurrences",
				created,
			)
		}
	})

	if err != nil {
		log.Errorf(
			"Could not register task recurrence series materializer cron: %s",
			err.Error(),
		)
	}
}
