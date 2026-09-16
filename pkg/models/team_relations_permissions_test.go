package models

import (
	"testing"
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNestedTeamProjectPermissionInheritance(t *testing.T) {
	tests := []struct {
		name       string
		permission int
		canWrite   bool
		isAdmin    bool
	}{
		{"read", 0, false, false},
		{"write", 1, true, false},
		{"admin", 2, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db.LoadAndAssertFixtures(t)

			s := db.NewSession()
			defer s.Close()

			const (
				parentTeamID = int64(9201)
				childTeamID  = int64(9202)
				projectID    = int64(9301)
				userID       = int64(1)
			)

			_, err := s.Insert(
				&Team{ID: parentTeamID, Name: "Parent Team", CreatedByID: 1},
				&Team{ID: childTeamID, Name: "Child Team", CreatedByID: 1},
			)
			require.NoError(t, err)

			_, err = s.Insert(&TeamMember{
				TeamID: childTeamID,
				UserID: userID,
			})
			require.NoError(t, err)

			_, err = createTeamRelation(s, parentTeamID, childTeamID)
			require.NoError(t, err)

			_, err = s.Insert(&Project{
				ID:      projectID,
				Title:   "Nested Team Permission Test",
				OwnerID: 999999,
			})
			require.NoError(t, err)

			now := time.Now()
			_, err = s.Exec(
				`INSERT INTO team_projects
					(team_id, project_id, permission, created, updated)
				 VALUES (?, ?, ?, ?, ?)`,
				parentTeamID,
				projectID,
				tt.permission,
				now,
				now,
			)
			require.NoError(t, err)

			u, err := user.GetUserByID(s, userID)
			require.NoError(t, err)

			project := &Project{ID: projectID}

			canRead, maxPermission, err := project.CanRead(s, u)
			require.NoError(t, err)

			canWrite, err := project.CanWrite(s, u)
			require.NoError(t, err)

			isAdmin, err := project.IsAdmin(s, u)
			require.NoError(t, err)

			assert.True(t, canRead)
			assert.Equal(t, tt.permission, maxPermission)
			assert.Equal(t, tt.canWrite, canWrite)
			assert.Equal(t, tt.isAdmin, isAdmin)
		})
	}
}

func TestNestedTeamProjectPermissionMultiLevel(t *testing.T) {
	db.LoadAndAssertFixtures(t)

	s := db.NewSession()
	defer s.Close()

	const (
		parentTeamID     = int64(9401)
		childTeamID      = int64(9402)
		grandchildTeamID = int64(9403)
		projectID        = int64(9404)
		userID           = int64(1)
	)

	_, err := s.Insert(
		&Team{ID: parentTeamID, Name: "Parent", CreatedByID: 1},
		&Team{ID: childTeamID, Name: "Child", CreatedByID: 1},
		&Team{ID: grandchildTeamID, Name: "Grandchild", CreatedByID: 1},
	)
	require.NoError(t, err)

	_, err = s.Insert(&TeamMember{TeamID: grandchildTeamID, UserID: userID})
	require.NoError(t, err)

	_, err = createTeamRelation(s, parentTeamID, childTeamID)
	require.NoError(t, err)

	_, err = createTeamRelation(s, childTeamID, grandchildTeamID)
	require.NoError(t, err)

	_, err = s.Insert(&Project{
		ID:      projectID,
		Title:   "Multi-level nested permission",
		OwnerID: 999999,
	})
	require.NoError(t, err)

	now := time.Now()
	_, err = s.Exec(
		`INSERT INTO team_projects
			(team_id, project_id, permission, created, updated)
		 VALUES (?, ?, ?, ?, ?)`,
		parentTeamID, projectID, 2, now, now,
	)
	require.NoError(t, err)

	u, err := user.GetUserByID(s, userID)
	require.NoError(t, err)

	project := &Project{ID: projectID}

	canRead, permission, err := project.CanRead(s, u)
	require.NoError(t, err)

	isAdmin, err := project.IsAdmin(s, u)
	require.NoError(t, err)

	assert.True(t, canRead)
	assert.Equal(t, 2, permission)
	assert.True(t, isAdmin)
}

func TestNestedTeamProjectPermissionHighestWins(t *testing.T) {
	db.LoadAndAssertFixtures(t)

	s := db.NewSession()
	defer s.Close()

	const (
		parentTeamID = int64(9501)
		childTeamID  = int64(9502)
		projectID    = int64(9503)
		userID       = int64(1)
	)

	_, err := s.Insert(
		&Team{ID: parentTeamID, Name: "Parent", CreatedByID: 1},
		&Team{ID: childTeamID, Name: "Child", CreatedByID: 1},
	)
	require.NoError(t, err)

	_, err = s.Insert(&TeamMember{TeamID: childTeamID, UserID: userID})
	require.NoError(t, err)

	_, err = createTeamRelation(s, parentTeamID, childTeamID)
	require.NoError(t, err)

	_, err = s.Insert(&Project{
		ID:      projectID,
		Title:   "Highest permission wins",
		OwnerID: 999999,
	})
	require.NoError(t, err)

	now := time.Now()

	// Child grants Read directly.
	_, err = s.Exec(
		`INSERT INTO team_projects
			(team_id, project_id, permission, created, updated)
		 VALUES (?, ?, ?, ?, ?)`,
		childTeamID, projectID, 0, now, now,
	)
	require.NoError(t, err)

	// Parent grants Admin through nesting.
	_, err = s.Exec(
		`INSERT INTO team_projects
			(team_id, project_id, permission, created, updated)
		 VALUES (?, ?, ?, ?, ?)`,
		parentTeamID, projectID, 2, now, now,
	)
	require.NoError(t, err)

	u, err := user.GetUserByID(s, userID)
	require.NoError(t, err)

	project := &Project{ID: projectID}

	canRead, permission, err := project.CanRead(s, u)
	require.NoError(t, err)

	assert.True(t, canRead)
	assert.Equal(t, 2, permission)
}

func TestNestedTeamProjectPermissionRevokedAfterRelationRemoval(t *testing.T) {
	db.LoadAndAssertFixtures(t)

	const (
		parentTeamID = int64(9601)
		childTeamID  = int64(9602)
		projectID    = int64(9603)
		userID       = int64(1)
	)

	s := db.NewSession()

	_, err := s.Insert(
		&Team{ID: parentTeamID, Name: "Parent", CreatedByID: 1},
		&Team{ID: childTeamID, Name: "Child", CreatedByID: 1},
	)
	require.NoError(t, err)

	_, err = s.Insert(&TeamMember{TeamID: childTeamID, UserID: userID})
	require.NoError(t, err)

	_, err = createTeamRelation(s, parentTeamID, childTeamID)
	require.NoError(t, err)

	_, err = s.Insert(&Project{
		ID:      projectID,
		Title:   "Revocation test",
		OwnerID: 999999,
	})
	require.NoError(t, err)

	now := time.Now()
	_, err = s.Exec(
		`INSERT INTO team_projects
			(team_id, project_id, permission, created, updated)
		 VALUES (?, ?, ?, ?, ?)`,
		parentTeamID, projectID, 1, now, now,
	)
	require.NoError(t, err)

	u, err := user.GetUserByID(s, userID)
	require.NoError(t, err)

	canRead, _, err := (&Project{ID: projectID}).CanRead(s, u)
	require.NoError(t, err)
	require.True(t, canRead)

	_, err = s.
		Where("parent_team_id = ? AND child_team_id = ?", parentTeamID, childTeamID).
		Delete(&TeamRelation{})
	require.NoError(t, err)

	require.NoError(t, s.Commit())
	s.Close()

	// New session represents the next request and avoids the per-session
	// project-access cache.
	s = db.NewSession()
	defer s.Close()

	u, err = user.GetUserByID(s, userID)
	require.NoError(t, err)

	canRead, _, err = (&Project{ID: projectID}).CanRead(s, u)
	require.NoError(t, err)

	assert.False(t, canRead)
}

func TestDirectProjectPermissionSurvivesRelationRemoval(t *testing.T) {
	db.LoadAndAssertFixtures(t)

	const (
		parentTeamID = int64(9701)
		childTeamID  = int64(9702)
		projectID    = int64(9703)
		userID       = int64(1)
	)

	s := db.NewSession()

	_, err := s.Insert(
		&Team{ID: parentTeamID, Name: "Parent", CreatedByID: 1},
		&Team{ID: childTeamID, Name: "Child", CreatedByID: 1},
	)
	require.NoError(t, err)

	_, err = s.Insert(&TeamMember{TeamID: childTeamID, UserID: userID})
	require.NoError(t, err)

	_, err = createTeamRelation(s, parentTeamID, childTeamID)
	require.NoError(t, err)

	_, err = s.Insert(&Project{
		ID:      projectID,
		Title:   "Direct permission survives",
		OwnerID: 999999,
	})
	require.NoError(t, err)

	now := time.Now()

	// Direct child-team Write permission.
	_, err = s.Exec(
		`INSERT INTO team_projects
			(team_id, project_id, permission, created, updated)
		 VALUES (?, ?, ?, ?, ?)`,
		childTeamID, projectID, 1, now, now,
	)
	require.NoError(t, err)

	// Inherited parent-team Admin permission.
	_, err = s.Exec(
		`INSERT INTO team_projects
			(team_id, project_id, permission, created, updated)
		 VALUES (?, ?, ?, ?, ?)`,
		parentTeamID, projectID, 2, now, now,
	)
	require.NoError(t, err)

	_, err = s.
		Where("parent_team_id = ? AND child_team_id = ?", parentTeamID, childTeamID).
		Delete(&TeamRelation{})
	require.NoError(t, err)

	require.NoError(t, s.Commit())
	s.Close()

	s = db.NewSession()
	defer s.Close()

	u, err := user.GetUserByID(s, userID)
	require.NoError(t, err)

	project := &Project{ID: projectID}

	canRead, permission, err := project.CanRead(s, u)
	require.NoError(t, err)

	canWrite, err := project.CanWrite(s, u)
	require.NoError(t, err)

	isAdmin, err := project.IsAdmin(s, u)
	require.NoError(t, err)

	assert.True(t, canRead)
	assert.Equal(t, 1, permission)
	assert.True(t, canWrite)
	assert.False(t, isAdmin)
}

func TestNestedTeamMembershipGrantsParentMembership(t *testing.T) {
	db.LoadAndAssertFixtures(t)

	s := db.NewSession()
	defer s.Close()

	const (
		parentTeamID = int64(9801)
		childTeamID  = int64(9802)
		userID       = int64(1)
	)

	_, err := s.Insert(
		&Team{ID: parentTeamID, Name: "Parent Team", CreatedByID: 1},
		&Team{ID: childTeamID, Name: "Child Team", CreatedByID: 1},
	)
	require.NoError(t, err)

	_, err = s.Insert(&TeamMember{
		TeamID: childTeamID,
		UserID: userID,
		Admin:  true,
	})
	require.NoError(t, err)

	_, err = createTeamRelation(s, parentTeamID, childTeamID)
	require.NoError(t, err)

	u, err := user.GetUserByID(s, userID)
	require.NoError(t, err)

	parentReadable, maxPermission, err := (&Team{ID: parentTeamID}).CanRead(s, u)
	require.NoError(t, err)
	assert.True(t, parentReadable)
	assert.Equal(t, 0, maxPermission)

	parentAdmin, err := (&Team{ID: parentTeamID}).IsAdmin(s, u)
	require.NoError(t, err)
	assert.False(t, parentAdmin)
}

func TestNestedTeamAdminRelationGrantsParentAdministration(t *testing.T) {
	db.LoadAndAssertFixtures(t)

	s := db.NewSession()
	defer s.Close()

	const (
		parentTeamID = int64(9811)
		childTeamID  = int64(9812)
		userID       = int64(1)
	)

	_, err := s.Insert(
		&Team{ID: parentTeamID, Name: "Parent Team", CreatedByID: 1},
		&Team{ID: childTeamID, Name: "Child Team", CreatedByID: 1},
	)
	require.NoError(t, err)

	_, err = s.Insert(&TeamMember{
		TeamID: childTeamID,
		UserID: userID,
	})
	require.NoError(t, err)

	_, err = createTeamRelationWithAdmin(s, parentTeamID, childTeamID, true)
	require.NoError(t, err)

	u, err := user.GetUserByID(s, userID)
	require.NoError(t, err)

	parentReadable, maxPermission, err := (&Team{ID: parentTeamID}).CanRead(s, u)
	require.NoError(t, err)
	assert.True(t, parentReadable)
	assert.Equal(t, int(PermissionAdmin), maxPermission)

	parentAdmin, err := (&Team{ID: parentTeamID}).IsAdmin(s, u)
	require.NoError(t, err)
	assert.True(t, parentAdmin)

	memberAdmin, err := (&TeamMember{TeamID: parentTeamID}).IsAdmin(s, u)
	require.NoError(t, err)
	assert.True(t, memberAdmin)
}

func TestNestedTeamAdminUsesEffectiveChildMembership(t *testing.T) {
	db.LoadAndAssertFixtures(t)

	s := db.NewSession()
	defer s.Close()

	const (
		parentTeamID     = int64(9821)
		childTeamID      = int64(9822)
		grandchildTeamID = int64(9823)
		userID           = int64(1)
	)

	_, err := s.Insert(
		&Team{ID: parentTeamID, Name: "Parent", CreatedByID: 1},
		&Team{ID: childTeamID, Name: "Child", CreatedByID: 1},
		&Team{ID: grandchildTeamID, Name: "Grandchild", CreatedByID: 1},
	)
	require.NoError(t, err)

	_, err = s.Insert(&TeamMember{
		TeamID: grandchildTeamID,
		UserID: userID,
	})
	require.NoError(t, err)

	_, err = createTeamRelation(s, childTeamID, grandchildTeamID)
	require.NoError(t, err)

	_, err = createTeamRelationWithAdmin(s, parentTeamID, childTeamID, true)
	require.NoError(t, err)

	u, err := user.GetUserByID(s, userID)
	require.NoError(t, err)

	childReadable, _, err := (&Team{ID: childTeamID}).CanRead(s, u)
	require.NoError(t, err)
	assert.True(t, childReadable)

	parentAdmin, err := (&Team{ID: parentTeamID}).IsAdmin(s, u)
	require.NoError(t, err)
	assert.True(t, parentAdmin)
}
