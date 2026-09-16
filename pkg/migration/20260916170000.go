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
	"time"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

type taskRecurrence20260916170000 struct {
	ID     int64 `xorm:"bigint autoincr not null unique pk"`
	TaskID int64 `xorm:"bigint not null"`

	Frequency int `xorm:"int not null"`
	Interval  int `xorm:"int not null default 1"`
	Basis     int `xorm:"int not null default 0"`

	ByWeekdays int `xorm:"smallint not null default 0"`
	ByMonth    int `xorm:"smallint not null default 0"`
	ByMonthDay int `xorm:"smallint not null default 0"`
	BySetPos   int `xorm:"smallint not null default 0"`

	MissingPolicy int `xorm:"int not null default 0"`

	Created time.Time `xorm:"created not null"`
	Updated time.Time `xorm:"updated not null"`
}

func (taskRecurrence20260916170000) TableName() string {
	return "task_recurrences"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260916170000",
		Description: "Create task recurrence rules",
		Migrate: func(tx *xorm.Engine) error {
			return partialSync(tx, taskRecurrence20260916170000{})
		},
		Rollback: func(tx *xorm.Engine) error {
			return nil
		},
	})
}
