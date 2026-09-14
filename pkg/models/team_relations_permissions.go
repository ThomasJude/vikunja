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
	"code.vikunja.io/api/pkg/web"
	"xorm.io/xorm"
)

func (tr *TeamRelation) CanCreate(s *xorm.Session, a web.Auth) (bool, error) {
	parentAdmin, err := (&Team{ID: tr.ParentTeamID}).IsAdmin(s, a)
	if err != nil || !parentAdmin {
		return parentAdmin, err
	}

	return (&Team{ID: tr.ChildTeamID}).IsAdmin(s, a)
}

func (tr *TeamRelation) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	parentAdmin, err := (&Team{ID: tr.ParentTeamID}).IsAdmin(s, a)
	if err != nil {
		return false, err
	}
	if parentAdmin {
		return true, nil
	}

	return (&Team{ID: tr.ChildTeamID}).IsAdmin(s, a)
}

func (tr *TeamRelation) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	can, err := (&Team{ID: tr.ParentTeamID}).IsAdmin(s, a)
	if !can || err != nil {
		return can, 0, err
	}

	return true, int(PermissionAdmin), nil
}

func (tr *TeamRelation) CanUpdate(_ *xorm.Session, _ web.Auth) (bool, error) {
	return false, nil
}
