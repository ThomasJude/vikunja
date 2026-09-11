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
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"code.vikunja.io/api/pkg/events"
	"code.vikunja.io/api/pkg/license"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/utils"
	"xorm.io/builder"
	"xorm.io/xorm"

	"code.vikunja.io/api/pkg/db"
	"github.com/stretchr/testify/require"
)

func TestInviteLinkFixtures(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	links := []UserInviteLink{}
	require.NoError(t, s.Find(&links))
	require.Len(t, links, 4)
	require.Nil(t, links[0].MaxUses)
	require.Nil(t, links[0].ExpiresAt)
	_, err := s.Insert(&UserInviteLinkTeam{InviteLinkID: 1, TeamID: 1})
	require.Error(t, err)
}

func inviteLinkSetup(t *testing.T) (*xorm.Session, *user.User) {
	t.Helper()
	adminActionsSetup(t)
	license.SetForTests([]license.Feature{license.FeatureAdminPanel, license.FeatureUserInvites})
	t.Cleanup(license.ResetForTests)
	s := db.NewSession()
	t.Cleanup(func() { events.CleanupPending(s); _ = s.Close() })
	_, err := s.ID(1).Cols("is_admin").Update(&user.User{IsAdmin: true})
	require.NoError(t, err)
	return s, &user.User{ID: 1}
}

func TestInviteLinkAdminCreate(t *testing.T) {
	s, admin := inviteLinkSetup(t)
	link, err := CreateInviteLinkAsAdmin(s, admin, &CreateInviteLinkBody{Name: "Welcome", TeamIDs: []int64{1, 1, 8}, SkipEmailConfirm: true})
	require.NoError(t, err)
	require.Len(t, link.ClearTextToken, 64)
	require.Equal(t, utils.Sha256Hex(link.ClearTextToken), link.TokenHash)
	require.Len(t, link.Teams, 2)
	stored := &UserInviteLink{}
	found, err := s.ID(link.ID).Get(stored)
	require.NoError(t, err)
	require.True(t, found)
	require.Empty(t, stored.ClearTextToken)
	require.Equal(t, link.TokenHash, stored.TokenHash)
	require.NoError(t, s.Commit())
	events.DispatchPending(context.Background(), s)
	require.Equal(t, link.ID, singleDispatchedEvent[*AdminInviteLinkCreatedEvent](t).Link.ID)
}

func TestInviteLinkAdminValidation(t *testing.T) {
	zero := int64(0)
	past := time.Now().Add(-time.Hour)
	for _, tc := range []struct {
		name string
		body CreateInviteLinkBody
		want error
	}{
		{"empty name", CreateInviteLinkBody{}, ErrInvalidInviteLinkInput{}},
		{"blank name", CreateInviteLinkBody{Name: "  "}, ErrInvalidInviteLinkInput{}},
		{"long name", CreateInviteLinkBody{Name: strings.Repeat("a", 251)}, ErrInvalidInviteLinkInput{}},
		{"zero uses", CreateInviteLinkBody{Name: "test", MaxUses: &zero}, ErrInvalidInviteLinkInput{}},
		{"past expiry", CreateInviteLinkBody{Name: "test", ExpiresAt: &past}, ErrInvalidInviteLinkInput{}},
		{"unknown team", CreateInviteLinkBody{Name: "test", TeamIDs: []int64{999}}, ErrTeamDoesNotExist{TeamID: 999}},
		{"external team", CreateInviteLinkBody{Name: "test", TeamIDs: []int64{14}}, ErrInviteLinkExternalTeam{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, admin := inviteLinkSetup(t)
			_, err := CreateInviteLinkAsAdmin(s, admin, &tc.body)
			require.ErrorIs(t, err, tc.want)
			count, err := s.Count(&UserInviteLink{})
			require.NoError(t, err)
			require.EqualValues(t, 4, count)
		})
	}
}

func TestInviteLinkAdminAuthorization(t *testing.T) {
	for _, mode := range []string{"non-admin", "no invites", "no admin panel"} {
		t.Run(mode, func(t *testing.T) {
			s, admin := inviteLinkSetup(t)
			switch mode {
			case "non-admin":
				admin.ID = 2
			case "no invites":
				license.SetForTests([]license.Feature{license.FeatureAdminPanel})
			case "no admin panel":
				license.SetForTests([]license.Feature{license.FeatureUserInvites})
			}
			_, err := CreateInviteLinkAsAdmin(s, admin, &CreateInviteLinkBody{Name: "test"})
			require.Error(t, err)
			_, _, err = ListInviteLinksAsAdmin(s, admin, 1, 50)
			require.Error(t, err)
			require.Error(t, DeleteInviteLinkAsAdmin(s, admin, 1))
			_, _, err = ListTeamsAsAdmin(s, admin, "", 1, 50)
			require.Error(t, err)
		})
	}
}

func TestInviteLinkAdminListDelete(t *testing.T) {
	s, admin := inviteLinkSetup(t)
	links, total, err := ListInviteLinksAsAdmin(s, admin, 1, 2)
	require.NoError(t, err)
	require.EqualValues(t, 4, total)
	require.Len(t, links, 2)
	require.NotEmpty(t, links[0].Teams)
	data, err := json.Marshal(links)
	require.NoError(t, err)
	require.NotContains(t, string(data), "token")
	require.NoError(t, DeleteInviteLinkAsAdmin(s, admin, 1))
	n, err := s.Where(builder.Eq{"invite_link_id": 1}).Count(&UserInviteLinkTeam{})
	require.NoError(t, err)
	require.Zero(t, n)
	require.ErrorIs(t, DeleteInviteLinkAsAdmin(s, admin, 999), ErrInviteLinkDoesNotExist{})
	require.NoError(t, s.Commit())
	events.DispatchPending(context.Background(), s)
	require.EqualValues(t, 1, singleDispatchedEvent[*AdminInviteLinkDeletedEvent](t).Link.ID)
}

func TestInviteLinkAdminTeams(t *testing.T) {
	s, admin := inviteLinkSetup(t)
	teams, total, err := ListTeamsAsAdmin(s, admin, "testteam", 1, 50)
	require.NoError(t, err)
	require.Greater(t, total, int64(2))
	ids := []int64{}
	for _, team := range teams {
		ids = append(ids, team.ID)
	}
	require.Contains(t, ids, int64(8))
	require.NotContains(t, ids, int64(14))
	require.NotContains(t, ids, int64(15))
	teams, _, err = ListTeamsAsAdmin(s, admin, "testteam8", 1, 1)
	require.NoError(t, err)
	require.Len(t, teams, 1)
	require.EqualValues(t, 8, teams[0].ID)
	require.NoError(t, (&Team{ID: 1}).Delete(s, admin))
	n, err := s.Where(builder.Eq{"team_id": 1}).Count(&UserInviteLinkTeam{})
	require.NoError(t, err)
	require.Zero(t, n)
}
