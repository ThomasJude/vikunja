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
	"slices"
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// TeamRelation defines a parent-child relation between two teams.
type TeamRelation struct {
	ID int64 `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true"`

	ParentTeamID int64 `xorm:"bigint not null index unique(team_relation)" json:"-" param:"team"`
	ChildTeamID  int64 `xorm:"bigint not null index unique(team_relation)" json:"child_team_id" param:"child_team"`

	Created time.Time `xorm:"created not null" json:"created" readOnly:"true"`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

func (*TeamRelation) TableName() string {
	return "team_relations"
}

func getAncestorTeamIDs(s *xorm.Session, teamIDs []int64) ([]int64, error) {
	seen := make(map[int64]struct{}, len(teamIDs))
	frontier := make([]int64, 0, len(teamIDs))

	for _, teamID := range teamIDs {
		if teamID <= 0 {
			continue
		}
		if _, exists := seen[teamID]; exists {
			continue
		}

		seen[teamID] = struct{}{}
		frontier = append(frontier, teamID)
	}

	for len(frontier) > 0 {
		relations := []TeamRelation{}
		err := s.In("child_team_id", frontier).Find(&relations)
		if err != nil {
			return nil, err
		}

		next := make([]int64, 0, len(relations))
		for _, relation := range relations {
			if _, exists := seen[relation.ParentTeamID]; exists {
				continue
			}

			seen[relation.ParentTeamID] = struct{}{}
			next = append(next, relation.ParentTeamID)
		}

		frontier = next
	}

	result := make([]int64, 0, len(seen))
	for teamID := range seen {
		result = append(result, teamID)
	}
	slices.Sort(result)

	return result, nil
}

func getEffectiveTeamIDsForUser(s *xorm.Session, userID int64) ([]int64, error) {
	memberships := []TeamMember{}
	err := s.Where("user_id = ?", userID).Find(&memberships)
	if err != nil {
		return nil, err
	}

	teamIDs := make([]int64, 0, len(memberships))
	for _, membership := range memberships {
		teamIDs = append(teamIDs, membership.TeamID)
	}

	return getAncestorTeamIDs(s, teamIDs)
}

func validateTeamRelation(s *xorm.Session, parentTeamID, childTeamID int64) error {
	if parentTeamID <= 0 {
		return ErrTeamDoesNotExist{TeamID: parentTeamID}
	}
	if childTeamID <= 0 {
		return ErrTeamDoesNotExist{TeamID: childTeamID}
	}
	if parentTeamID == childTeamID {
		return ErrTeamCannotContainItself{TeamID: parentTeamID}
	}

	_, err := GetTeamByID(s, parentTeamID)
	if err != nil {
		return err
	}

	_, err = GetTeamByID(s, childTeamID)
	if err != nil {
		return err
	}

	exists, err := s.
		Where("parent_team_id = ? AND child_team_id = ?", parentTeamID, childTeamID).
		Exist(&TeamRelation{})
	if err != nil {
		return err
	}
	if exists {
		return ErrTeamRelationAlreadyExists{
			ParentTeamID: parentTeamID,
			ChildTeamID:  childTeamID,
		}
	}

	ancestors, err := getAncestorTeamIDs(s, []int64{parentTeamID})
	if err != nil {
		return err
	}
	if slices.Contains(ancestors, childTeamID) {
		return ErrTeamRelationWouldCreateCycle{
			ParentTeamID: parentTeamID,
			ChildTeamID:  childTeamID,
		}
	}

	return nil
}

func createTeamRelation(s *xorm.Session, parentTeamID, childTeamID int64) (*TeamRelation, error) {
	if err := validateTeamRelation(s, parentTeamID, childTeamID); err != nil {
		return nil, err
	}

	relation := &TeamRelation{
		ParentTeamID: parentTeamID,
		ChildTeamID:  childTeamID,
	}

	_, err := s.Insert(relation)
	if err != nil {
		return nil, err
	}

	return relation, nil
}

// Create adds a child team to a team.
func (tr *TeamRelation) Create(s *xorm.Session, _ web.Auth) error {
	relation, err := createTeamRelation(s, tr.ParentTeamID, tr.ChildTeamID)
	if err != nil {
		return err
	}

	*tr = *relation
	return nil
}

// Delete removes a child team from a team.
func (tr *TeamRelation) Delete(s *xorm.Session, _ web.Auth) error {
	has, err := s.
		Where("parent_team_id = ? AND child_team_id = ?", tr.ParentTeamID, tr.ChildTeamID).
		Get(&TeamRelation{})
	if err != nil {
		return err
	}
	if !has {
		return ErrTeamRelationDoesNotExist{
			ParentTeamID: tr.ParentTeamID,
			ChildTeamID:  tr.ChildTeamID,
		}
	}

	_, err = s.
		Where("parent_team_id = ? AND child_team_id = ?", tr.ParentTeamID, tr.ChildTeamID).
		Delete(&TeamRelation{})
	return err
}

// ReadAll returns all child teams of a team.
func (tr *TeamRelation) ReadAll(s *xorm.Session, a web.Auth, search string, page int, perPage int) (result interface{}, resultCount int, totalItems int64, err error) {
	can, err := (&Team{ID: tr.ParentTeamID}).IsAdmin(s, a)
	if err != nil {
		return nil, 0, 0, err
	}
	if !can {
		return nil, 0, 0, ErrGenericForbidden{}
	}

	totalItems, err = s.
		Table("teams").
		Join("INNER", "team_relations", "team_relations.child_team_id = teams.id").
		Where("team_relations.parent_team_id = ?", tr.ParentTeamID).
		Where(db.ILIKE("teams.name", search)).
		Count(&Team{})
	if err != nil {
		return nil, 0, 0, err
	}

	limit, start := getLimitFromPageIndex(page, perPage)

	teams := []*Team{}
	query := s.
		Table("teams").
		Join("INNER", "team_relations", "team_relations.child_team_id = teams.id").
		Where("team_relations.parent_team_id = ?", tr.ParentTeamID).
		Where(db.ILIKE("teams.name", search)).
		OrderBy("teams.name ASC")

	if limit > 0 {
		query = query.Limit(limit, start)
	}

	err = query.Find(&teams)
	if err != nil {
		return nil, 0, 0, err
	}

	return teams, len(teams), totalItems, nil
}
