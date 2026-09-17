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

package migration

import (
	"fmt"
	"strings"
	"time"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

type taskRecurrenceSeries20260917163000 struct {
	ID int64 `xorm:"bigint autoincr not null unique pk"`

	RootTaskID  int64 `xorm:"bigint not null"`
	ProjectID   int64 `xorm:"bigint not null"`
	CreatedByID int64 `xorm:"bigint not null"`

	Frequency int `xorm:"int not null"`
	Interval  int `xorm:"int not null default 1"`
	Basis     int `xorm:"int not null default 0"`

	ByWeekdays int `xorm:"smallint not null default 0"`
	ByMonth    int `xorm:"smallint not null default 0"`
	ByMonthDay int `xorm:"smallint not null default 0"`
	BySetPos   int `xorm:"smallint not null default 0"`

	MissingPolicy int `xorm:"int not null default 0"`

	StartDate           time.Time `xorm:"datetime not null"`
	EndType             int       `xorm:"int not null default 0"`
	EndDate             time.Time `xorm:"datetime null"`
	EndAfterOccurrences int       `xorm:"int not null default 0"`

	CreateBeforeDays int `xorm:"int not null default 0"`

	WeekendPolicy int  `xorm:"int not null default 0"`
	MissedPolicy  int  `xorm:"int not null default 0"`
	Paused        bool `xorm:"not null default false"`

	Created time.Time `xorm:"created not null"`
	Updated time.Time `xorm:"updated not null"`
}

func (taskRecurrenceSeries20260917163000) TableName() string {
	return "task_recurrence_series"
}

type taskRecurrenceOccurrence20260917163000 struct {
	ID int64 `xorm:"bigint autoincr not null unique pk"`

	SeriesID int64 `xorm:"bigint not null"`
	TaskID   int64 `xorm:"bigint not null"`

	Sequence int `xorm:"int not null"`

	ScheduledDueDate time.Time `xorm:"datetime not null"`
	DueDate          time.Time `xorm:"datetime not null"`

	IsException bool `xorm:"not null default false"`

	Created time.Time `xorm:"created not null"`
	Updated time.Time `xorm:"updated not null"`
}

func (taskRecurrenceOccurrence20260917163000) TableName() string {
	return "task_recurrence_occurrences"
}

func createTaskRecurrenceIndex20260917163000(
	tx *xorm.Engine,
	name string,
	table string,
	columns string,
	unique bool,
) error {
	kind := ""
	if unique {
		kind = "UNIQUE "
	}

	query := fmt.Sprintf(
		"CREATE %sINDEX IF NOT EXISTS %s ON %s (%s)",
		kind,
		name,
		table,
		columns,
	)

	if tx.Dialect().URI().DBType == schemas.MYSQL {
		query = fmt.Sprintf(
			"CREATE %sINDEX %s ON %s (%s)",
			kind,
			name,
			table,
			columns,
		)
	}

	_, err := tx.Exec(query)
	if err != nil &&
		tx.Dialect().URI().DBType == schemas.MYSQL &&
		strings.Contains(err.Error(), "Duplicate key name") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("could not create index %s: %w", name, err)
	}

	return nil
}

func createTaskRecurrenceSeriesTables20260917163000(tx *xorm.Engine) error {
	if err := partialSync(
		tx,
		taskRecurrenceSeries20260917163000{},
		taskRecurrenceOccurrence20260917163000{},
	); err != nil {
		return err
	}

	indexes := []struct {
		name    string
		table   string
		columns string
		unique  bool
	}{
		{
			name:    "IDX_task_recurrence_series_root_task_id",
			table:   "task_recurrence_series",
			columns: "root_task_id",
		},
		{
			name:    "IDX_task_recurrence_series_project_id",
			table:   "task_recurrence_series",
			columns: "project_id",
		},
		{
			name:    "IDX_task_recurrence_series_created_by_id",
			table:   "task_recurrence_series",
			columns: "created_by_id",
		},
		{
			name:    "IDX_task_recurrence_occurrences_series_id",
			table:   "task_recurrence_occurrences",
			columns: "series_id",
		},
		{
			name:    "UQE_task_recurrence_occurrences_task_id",
			table:   "task_recurrence_occurrences",
			columns: "task_id",
			unique:  true,
		},
		{
			name:    "UQE_task_recurrence_occurrences_series_sequence",
			table:   "task_recurrence_occurrences",
			columns: "series_id, sequence",
			unique:  true,
		},
	}

	for _, index := range indexes {
		if err := createTaskRecurrenceIndex20260917163000(
			tx,
			index.name,
			index.table,
			index.columns,
			index.unique,
		); err != nil {
			return err
		}
	}

	return nil
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260917163000",
		Description: "Create recurring task series and occurrence tables",
		Migrate:     createTaskRecurrenceSeriesTables20260917163000,
		Rollback: func(_ *xorm.Engine) error {
			return nil
		},
	})
}
