package models

import (
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamRelationPermissions(t *testing.T) {
	db.LoadAndAssertFixtures(t)

	s := db.NewSession()
	defer s.Close()

	const (
		parentTeamID = int64(9901)
		childTeamID  = int64(9902)
		otherTeamID  = int64(9903)
		userID       = int64(1)
	)

	_, err := s.Insert(
		&Team{ID: parentTeamID, Name: "Parent", CreatedByID: userID},
		&Team{ID: childTeamID, Name: "Child", CreatedByID: userID},
		&Team{ID: otherTeamID, Name: "Other", CreatedByID: userID},
	)
	require.NoError(t, err)

	_, err = s.Insert(
		&TeamMember{TeamID: parentTeamID, UserID: userID, Admin: true},
		&TeamMember{TeamID: childTeamID, UserID: userID, Admin: true},
	)
	require.NoError(t, err)

	u, err := user.GetUserByID(s, userID)
	require.NoError(t, err)

	t.Run("create requires admin on both teams", func(t *testing.T) {
		can, err := (&TeamRelation{
			ParentTeamID: parentTeamID,
			ChildTeamID:  childTeamID,
		}).CanCreate(s, u)

		require.NoError(t, err)
		assert.True(t, can)
	})

	t.Run("create denied without child admin", func(t *testing.T) {
		can, err := (&TeamRelation{
			ParentTeamID: parentTeamID,
			ChildTeamID:  otherTeamID,
		}).CanCreate(s, u)

		require.NoError(t, err)
		assert.False(t, can)
	})

	t.Run("delete allowed from parent", func(t *testing.T) {
		can, err := (&TeamRelation{
			ParentTeamID: parentTeamID,
			ChildTeamID:  otherTeamID,
		}).CanDelete(s, u)

		require.NoError(t, err)
		assert.True(t, can)
	})

	t.Run("delete allowed from child", func(t *testing.T) {
		can, err := (&TeamRelation{
			ParentTeamID: otherTeamID,
			ChildTeamID:  childTeamID,
		}).CanDelete(s, u)

		require.NoError(t, err)
		assert.True(t, can)
	})
}

func TestTeamRelationReadAll(t *testing.T) {
	db.LoadAndAssertFixtures(t)

	s := db.NewSession()
	defer s.Close()

	const (
		parentTeamID = int64(9911)
		childOneID   = int64(9912)
		childTwoID   = int64(9913)
		userID       = int64(1)
	)

	_, err := s.Insert(
		&Team{ID: parentTeamID, Name: "Parent", CreatedByID: userID},
		&Team{ID: childOneID, Name: "Alpha", CreatedByID: userID},
		&Team{ID: childTwoID, Name: "Beta", CreatedByID: userID},
	)
	require.NoError(t, err)

	_, err = s.Insert(&TeamMember{
		TeamID: parentTeamID,
		UserID: userID,
		Admin:  true,
	})
	require.NoError(t, err)

	_, err = createTeamRelation(s, parentTeamID, childOneID)
	require.NoError(t, err)

	_, err = createTeamRelationWithAdmin(s, parentTeamID, childTwoID, true)
	require.NoError(t, err)

	u, err := user.GetUserByID(s, userID)
	require.NoError(t, err)

	result, count, total, err := (&TeamRelation{
		ParentTeamID: parentTeamID,
	}).ReadAll(s, u, "", 1, 50)

	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Equal(t, int64(2), total)

	relations, ok := result.([]*TeamRelation)
	require.True(t, ok)
	require.Len(t, relations, 2)

	require.NotNil(t, relations[0].ChildTeam)
	require.NotNil(t, relations[1].ChildTeam)

	assert.Equal(t, childOneID, relations[0].ChildTeam.ID)
	assert.False(t, relations[0].Admin)

	assert.Equal(t, childTwoID, relations[1].ChildTeam.ID)
	assert.True(t, relations[1].Admin)
}

func TestTeamDeleteRemovesRelations(t *testing.T) {
	tests := []struct {
		name     string
		deleteID int64
	}{
		{"parent", 9921},
		{"child", 9922},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db.LoadAndAssertFixtures(t)

			s := db.NewSession()
			defer s.Close()

			const (
				parentTeamID = int64(9921)
				childTeamID  = int64(9922)
				userID       = int64(1)
			)

			_, err := s.Insert(
				&Team{ID: parentTeamID, Name: "Parent", CreatedByID: userID},
				&Team{ID: childTeamID, Name: "Child", CreatedByID: userID},
			)
			require.NoError(t, err)

			_, err = createTeamRelation(s, parentTeamID, childTeamID)
			require.NoError(t, err)

			u, err := user.GetUserByID(s, userID)
			require.NoError(t, err)

			err = (&Team{ID: tt.deleteID}).Delete(s, u)
			require.NoError(t, err)

			count, err := s.
				Where("parent_team_id = ? OR child_team_id = ?", parentTeamID, childTeamID).
				Count(&TeamRelation{})
			require.NoError(t, err)

			assert.Equal(t, int64(0), count)
		})
	}
}

func TestTeamRelationUpdateTogglesAdmin(t *testing.T) {
	db.LoadAndAssertFixtures(t)

	s := db.NewSession()
	defer s.Close()

	const (
		parentTeamID = int64(9931)
		childTeamID  = int64(9932)
	)

	_, err := s.Insert(
		&Team{ID: parentTeamID, Name: "Parent", CreatedByID: 1},
		&Team{ID: childTeamID, Name: "Child", CreatedByID: 1},
	)
	require.NoError(t, err)

	_, err = createTeamRelation(s, parentTeamID, childTeamID)
	require.NoError(t, err)

	relation := &TeamRelation{
		ParentTeamID: parentTeamID,
		ChildTeamID:  childTeamID,
	}

	err = relation.Update(s, nil)
	require.NoError(t, err)
	assert.True(t, relation.Admin)

	stored := &TeamRelation{}
	has, err := s.
		Where("parent_team_id = ? AND child_team_id = ?", parentTeamID, childTeamID).
		Get(stored)
	require.NoError(t, err)
	require.True(t, has)
	assert.True(t, stored.Admin)

	err = relation.Update(s, nil)
	require.NoError(t, err)
	assert.False(t, relation.Admin)
}
