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
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package migration

import (
	"fmt"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260915150653",
		Description: "Add unique index for nested team relations",
		Migrate: func(tx *xorm.Engine) error {
			_, err := tx.Exec(`
				CREATE UNIQUE INDEX UQE_team_relations_team_relation
				ON team_relations (parent_team_id, child_team_id)
			`)
			if err != nil {
				return fmt.Errorf("could not create unique index on team relations: %w", err)
			}

			return nil
		},
		Rollback: func(tx *xorm.Engine) error {
			return nil
		},
	})
}
