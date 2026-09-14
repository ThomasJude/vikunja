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

	_, err = createTeamRelation(s, parentTeamID, childTwoID)
	require.NoError(t, err)

	u, err := user.GetUserByID(s, userID)
	require.NoError(t, err)

	result, count, total, err := (&TeamRelation{
		ParentTeamID: parentTeamID,
	}).ReadAll(s, u, "", 1, 50)

	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Equal(t, int64(2), total)

	teams, ok := result.([]*Team)
	require.True(t, ok)
	require.Len(t, teams, 2)

	assert.Equal(t, childOneID, teams[0].ID)
	assert.Equal(t, childTwoID, teams[1].ID)
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
