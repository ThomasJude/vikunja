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

package models

import "time"

type TaskRecurrenceEndType int

const (
	TaskRecurrenceEndNever TaskRecurrenceEndType = iota
	TaskRecurrenceEndDate
	TaskRecurrenceEndOccurrences
)

type TaskRecurrenceWeekendPolicy int

const (
	TaskRecurrenceWeekendKeep TaskRecurrenceWeekendPolicy = iota
	TaskRecurrenceWeekendPreviousBusinessDay
	TaskRecurrenceWeekendNextBusinessDay
)

type TaskRecurrenceMissedPolicy int

const (
	TaskRecurrenceMissedNextFuture TaskRecurrenceMissedPolicy = iota
	TaskRecurrenceMissedEveryOccurrence
)

type TaskRecurrenceSeries struct {
	ID int64 `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true"`

	RootTaskID  int64 `xorm:"bigint not null INDEX" json:"root_task_id" readOnly:"true"`
	ProjectID   int64 `xorm:"bigint not null INDEX" json:"project_id" readOnly:"true"`
	CreatedByID int64 `xorm:"bigint not null INDEX" json:"created_by_id" readOnly:"true"`

	Frequency TaskRecurrenceFrequency `xorm:"int not null" json:"frequency"`
	Interval  int                     `xorm:"int not null default 1" json:"interval"`
	Basis     TaskRecurrenceBasis     `xorm:"int not null default 0" json:"basis"`

	ByWeekdays int `xorm:"smallint not null default 0" json:"by_weekdays"`
	ByMonth    int `xorm:"smallint not null default 0" json:"by_month"`
	ByMonthDay int `xorm:"smallint not null default 0" json:"by_month_day"`
	BySetPos   int `xorm:"smallint not null default 0" json:"by_set_pos"`

	MissingPolicy TaskRecurrenceMissingPolicy `xorm:"int not null default 0" json:"missing_policy"`

	StartDate           time.Time             `xorm:"datetime not null" json:"start_date"`
	EndType             TaskRecurrenceEndType `xorm:"int not null default 0" json:"end_type"`
	EndDate             time.Time             `xorm:"datetime null" json:"end_date"`
	EndAfterOccurrences int                   `xorm:"int not null default 0" json:"end_after_occurrences"`

	CreateBeforeDays int `xorm:"int not null default 0" json:"create_before_days"`

	WeekendPolicy TaskRecurrenceWeekendPolicy `xorm:"int not null default 0" json:"weekend_policy"`
	MissedPolicy  TaskRecurrenceMissedPolicy  `xorm:"int not null default 0" json:"missed_policy"`

	Paused bool `xorm:"not null default false" json:"paused"`

	Created time.Time `xorm:"created not null" json:"created" readOnly:"true"`
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true"`
}

func (*TaskRecurrenceSeries) TableName() string {
	return "task_recurrence_series"
}

type TaskRecurrenceOccurrence struct {
	ID       int64 `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true"`
	SeriesID int64 `xorm:"bigint not null INDEX" json:"series_id" readOnly:"true"`
	TaskID   int64 `xorm:"bigint not null unique" json:"task_id" readOnly:"true"`

	Sequence int `xorm:"int not null" json:"sequence" readOnly:"true"`

	ScheduledDueDate time.Time `xorm:"datetime not null" json:"scheduled_due_date" readOnly:"true"`
	DueDate          time.Time `xorm:"datetime not null" json:"due_date" readOnly:"true"`

	IsException bool `xorm:"not null default false" json:"is_exception" readOnly:"true"`

	Created time.Time `xorm:"created not null" json:"created" readOnly:"true"`
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true"`
}

func (*TaskRecurrenceOccurrence) TableName() string {
	return "task_recurrence_occurrences"
}
