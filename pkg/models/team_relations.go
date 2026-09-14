// Vikunja is a to-do list application to facilitate your life.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"fmt"
	"slices"
	"time"

	"xorm.io/xorm"
)

// TeamRelation represents a parent-child relationship between two teams.
//
// Members of a child team inherit project access granted to its parent teams.
// Team administration itself is deliberately not inherited.
type TeamRelation struct {
	ID int64 `xorm:"bigint autoincr not null unique pk" json:"id"`

	ParentTeamID int64 `xorm:"bigint not null index unique(team_relation)" json:"parent_team_id"`
	ChildTeamID  int64 `xorm:"bigint not null index unique(team_relation)" json:"child_team_id"`

	Created time.Time `xorm:"created not null" json:"created"`
}

// TableName returns the database table name for team hierarchy relations.
func (TeamRelation) TableName() string {
	return "team_relations"
}

// getAncestorTeamIDs returns the supplied teams and all of their ancestor
// teams. The seen set also protects against infinite traversal if corrupt
// hierarchy data containing a cycle somehow exists in the database.
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

		err := s.
			In("child_team_id", frontier).
			Find(&relations)
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

// getEffectiveTeamIDsForUser returns all teams a user belongs to directly,
// plus all ancestor teams inherited through nested-team relationships.
func getEffectiveTeamIDsForUser(s *xorm.Session, userID int64) ([]int64, error) {
	memberships := []TeamMember{}

	err := s.
		Where("user_id = ?", userID).
		Find(&memberships)
	if err != nil {
		return nil, err
	}

	directTeamIDs := make([]int64, 0, len(memberships))
	for _, membership := range memberships {
		directTeamIDs = append(directTeamIDs, membership.TeamID)
	}

	return getAncestorTeamIDs(s, directTeamIDs)
}

// validateTeamRelation checks whether a parent-child relationship can be
// created without introducing invalid hierarchy data.
func validateTeamRelation(s *xorm.Session, parentTeamID, childTeamID int64) error {
	if parentTeamID <= 0 || childTeamID <= 0 {
		return fmt.Errorf("team IDs must be greater than zero")
	}

	if parentTeamID == childTeamID {
		return fmt.Errorf("a team cannot contain itself")
	}

	parentExists, err := s.ID(parentTeamID).Exist(&Team{})
	if err != nil {
		return err
	}
	if !parentExists {
		return fmt.Errorf("parent team %d does not exist", parentTeamID)
	}

	childExists, err := s.ID(childTeamID).Exist(&Team{})
	if err != nil {
		return err
	}
	if !childExists {
		return fmt.Errorf("child team %d does not exist", childTeamID)
	}

	exists, err := s.
		Where("parent_team_id = ? AND child_team_id = ?", parentTeamID, childTeamID).
		Exist(&TeamRelation{})
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("team relationship already exists")
	}

	// Adding parent -> child creates a cycle when child is already an
	// ancestor of parent.
	ancestors, err := getAncestorTeamIDs(s, []int64{parentTeamID})
	if err != nil {
		return err
	}

	if slices.Contains(ancestors, childTeamID) {
		return fmt.Errorf("team relationship would create a cycle")
	}

	return nil
}

// createTeamRelation creates a validated parent-child team relationship.
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
