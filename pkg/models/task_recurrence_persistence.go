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

import "xorm.io/xorm"

func getTaskRecurrenceMap(s *xorm.Session, taskIDs []int64) (map[int64]*TaskRecurrence, error) {
	recurrences := make(map[int64]*TaskRecurrence)
	if len(taskIDs) == 0 {
		return recurrences, nil
	}

	rules := []*TaskRecurrence{}
	if err := s.In("task_id", taskIDs).Find(&rules); err != nil {
		return nil, err
	}

	for _, rule := range rules {
		recurrences[rule.TaskID] = rule
	}

	return recurrences, nil
}

func saveTaskRecurrence(s *xorm.Session, taskID int64, recurrence *TaskRecurrence) error {
	if recurrence == nil {
		_, err := s.Where("task_id = ?", taskID).Delete(&TaskRecurrence{})
		return err
	}

	if err := validateTaskRecurrence(recurrence); err != nil {
		return err
	}

	existing := &TaskRecurrence{}
	has, err := s.Where("task_id = ?", taskID).Get(existing)
	if err != nil {
		return err
	}

	stored := *recurrence
	stored.TaskID = taskID

	if has {
		stored.ID = existing.ID
		_, err = s.ID(existing.ID).
			Cols(
				"frequency",
				"interval",
				"basis",
				"by_weekdays",
				"by_month",
				"by_month_day",
				"by_set_pos",
				"missing_policy",
			).
			Update(&stored)
	} else {
		stored.ID = 0
		_, err = s.Insert(&stored)
	}
	if err != nil {
		return err
	}

	refreshed := &TaskRecurrence{}
	has, err = s.Where("task_id = ?", taskID).Get(refreshed)
	if err != nil {
		return err
	}
	if !has {
		return ErrInvalidTaskRecurrence{Reason: "recurrence rule could not be stored"}
	}

	*recurrence = *refreshed
	return nil
}
