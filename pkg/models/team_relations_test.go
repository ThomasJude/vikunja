package models

import (
	"testing"

	"code.vikunja.io/api/pkg/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

const (
	testParentTeamID = int64(9101)
	testChildTeamID  = int64(9102)
	testGrandchildID = int64(9103)
)

func createNestedTeamTestTeams(t *testing.T, s *xorm.Session) {
	t.Helper()

	teams := []*Team{
		{
			ID:          testParentTeamID,
			Name:        "Nested Team Parent",
			CreatedByID: 1,
		},
		{
			ID:          testChildTeamID,
			Name:        "Nested Team Child",
			CreatedByID: 1,
		},
		{
			ID:          testGrandchildID,
			Name:        "Nested Team Grandchild",
			CreatedByID: 1,
		},
	}

	for _, team := range teams {
		_, err := s.Insert(team)
		require.NoError(t, err)
	}
}

func TestTeamRelationCreate(t *testing.T) {
	db.LoadAndAssertFixtures(t)

	s := db.NewSession()
	defer s.Close()

	createNestedTeamTestTeams(t, s)

	relation, err := createTeamRelation(
		s,
		testParentTeamID,
		testChildTeamID,
	)

	require.NoError(t, err)
	assert.Equal(t, testParentTeamID, relation.ParentTeamID)
	assert.Equal(t, testChildTeamID, relation.ChildTeamID)

	ancestors, err := getAncestorTeamIDs(
		s,
		[]int64{testChildTeamID},
	)

	require.NoError(t, err)
	assert.Equal(
		t,
		[]int64{testParentTeamID, testChildTeamID},
		ancestors,
	)
}

func TestTeamRelationRejectSelfReference(t *testing.T) {
	db.LoadAndAssertFixtures(t)

	s := db.NewSession()
	defer s.Close()

	createNestedTeamTestTeams(t, s)

	_, err := createTeamRelation(
		s,
		testParentTeamID,
		testParentTeamID,
	)

	assert.Error(t, err)
}

func TestTeamRelationRejectDuplicate(t *testing.T) {
	db.LoadAndAssertFixtures(t)

	s := db.NewSession()
	defer s.Close()

	createNestedTeamTestTeams(t, s)

	_, err := createTeamRelation(
		s,
		testParentTeamID,
		testChildTeamID,
	)
	require.NoError(t, err)

	_, err = createTeamRelation(
		s,
		testParentTeamID,
		testChildTeamID,
	)

	assert.Error(t, err)
}

func TestTeamRelationRejectTwoTeamCycle(t *testing.T) {
	db.LoadAndAssertFixtures(t)

	s := db.NewSession()
	defer s.Close()

	createNestedTeamTestTeams(t, s)

	_, err := createTeamRelation(
		s,
		testParentTeamID,
		testChildTeamID,
	)
	require.NoError(t, err)

	_, err = createTeamRelation(
		s,
		testChildTeamID,
		testParentTeamID,
	)

	assert.Error(t, err)
}

func TestTeamRelationRejectDeepCycle(t *testing.T) {
	db.LoadAndAssertFixtures(t)

	s := db.NewSession()
	defer s.Close()

	createNestedTeamTestTeams(t, s)

	_, err := createTeamRelation(
		s,
		testParentTeamID,
		testChildTeamID,
	)
	require.NoError(t, err)

	_, err = createTeamRelation(
		s,
		testChildTeamID,
		testGrandchildID,
	)
	require.NoError(t, err)

	_, err = createTeamRelation(
		s,
		testGrandchildID,
		testParentTeamID,
	)

	assert.Error(t, err)
}

func TestTeamRelationRejectMissingTeams(t *testing.T) {
	db.LoadAndAssertFixtures(t)

	s := db.NewSession()
	defer s.Close()

	createNestedTeamTestTeams(t, s)

	t.Run("missing parent", func(t *testing.T) {
		_, err := createTeamRelation(
			s,
			999991,
			testChildTeamID,
		)

		assert.Error(t, err)
	})

	t.Run("missing child", func(t *testing.T) {
		_, err := createTeamRelation(
			s,
			testParentTeamID,
			999992,
		)

		assert.Error(t, err)
	})
}

func TestEffectiveTeamIDsForNestedMembership(t *testing.T) {
	db.LoadAndAssertFixtures(t)

	s := db.NewSession()
	defer s.Close()

	createNestedTeamTestTeams(t, s)

	const nestedUserID = int64(99001)

	_, err := s.Insert(&TeamMember{
		TeamID: testGrandchildID,
		UserID: nestedUserID,
	})
	require.NoError(t, err)

	_, err = createTeamRelation(
		s,
		testParentTeamID,
		testChildTeamID,
	)
	require.NoError(t, err)

	_, err = createTeamRelation(
		s,
		testChildTeamID,
		testGrandchildID,
	)
	require.NoError(t, err)

	teamIDs, err := getEffectiveTeamIDsForUser(
		s,
		nestedUserID,
	)

	require.NoError(t, err)
	assert.Equal(
		t,
		[]int64{
			testParentTeamID,
			testChildTeamID,
			testGrandchildID,
		},
		teamIDs,
	)
}
