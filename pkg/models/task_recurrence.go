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

type TaskRecurrenceFrequency int

const (
	TaskRecurrenceFrequencyInvalid TaskRecurrenceFrequency = iota
	TaskRecurrenceFrequencyDay
	TaskRecurrenceFrequencyWeek
	TaskRecurrenceFrequencyMonth
	TaskRecurrenceFrequencyYear
)

type TaskRecurrenceBasis int

const (
	TaskRecurrenceBasisSchedule TaskRecurrenceBasis = iota
	TaskRecurrenceBasisCompletion
)

type TaskRecurrenceMissingPolicy int

const (
	TaskRecurrenceMissingPolicyDefault TaskRecurrenceMissingPolicy = iota
	TaskRecurrenceMissingPolicyLastValid
	TaskRecurrenceMissingPolicySkip
	TaskRecurrenceMissingPolicyLastOccurrence
	TaskRecurrenceMissingPolicyNextPeriod
)

type TaskRecurrence struct {
	ID     int64 `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true"`
	TaskID int64 `xorm:"bigint not null unique" json:"task_id" readOnly:"true"`

	Frequency TaskRecurrenceFrequency `xorm:"int not null" json:"frequency"`
	Interval  int                     `xorm:"int not null default 1" json:"interval"`
	Basis     TaskRecurrenceBasis     `xorm:"int not null default 0" json:"basis"`

	ByWeekdays int `xorm:"smallint not null default 0" json:"by_weekdays"`
	ByMonth    int `xorm:"smallint not null default 0" json:"by_month"`
	ByMonthDay int `xorm:"smallint not null default 0" json:"by_month_day"`
	BySetPos   int `xorm:"smallint not null default 0" json:"by_set_pos"`

	MissingPolicy TaskRecurrenceMissingPolicy `xorm:"int not null default 0" json:"missing_policy"`

	Created time.Time `xorm:"created not null" json:"created" readOnly:"true"`
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true"`
}

func (*TaskRecurrence) TableName() string {
	return "task_recurrences"
}
