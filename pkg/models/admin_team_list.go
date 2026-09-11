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
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package models

import (
	"fmt"

	"code.vikunja.io/api/pkg/user"
	"xorm.io/builder"
	"xorm.io/xorm"
)

func ListTeamsAsAdmin(s *xorm.Session, doer *user.User, search string, page, perPage int) ([]InviteLinkTeam, int64, error) {
	if err := requireInviteLinkAdmin(s, doer); err != nil {
		return nil, 0, err
	}
	limit, start := getLimitFromPageIndex(page, perPage)
	teams := []InviteLinkTeam{}
	cond := builder.Or(builder.IsNull{"external_id"}, builder.Eq{"external_id": ""})
	if search != "" {
		cond = cond.And(builder.Like{"name", "%" + search + "%"})
	}
	total, err := s.Table("teams").Select("id, name").Where(cond).Limit(limit, start).OrderBy("id ASC").FindAndCount(&teams)
	if err != nil {
		return nil, 0, fmt.Errorf("list invite teams: %w", err)
	}
	return teams, total, nil
}
