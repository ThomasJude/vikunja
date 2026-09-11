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
	"strings"
	"time"
	"unicode/utf8"

	"code.vikunja.io/api/pkg/events"
	"code.vikunja.io/api/pkg/license"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/utils"
	"xorm.io/builder"
	"xorm.io/xorm"
)

type InviteLinkTeam struct {
	ID   int64  `json:"id" doc:"Numeric team ID."`
	Name string `json:"name" doc:"Team name."`
}

type UserInviteLink struct {
	ID               int64            `xorm:"bigint autoincr not null unique pk" json:"id" doc:"Numeric link ID."`
	Name             string           `xorm:"varchar(250) not null" json:"name" doc:"Name shown to admins and invitees."`
	TokenHash        string           `xorm:"varchar(64) not null unique" json:"-"`
	MaxUses          *int64           `xorm:"bigint null" json:"max_uses" doc:"Null allows unlimited registrations."`
	Uses             int64            `xorm:"bigint not null default 0" json:"uses" doc:"Completed registrations."`
	ExpiresAt        *time.Time       `xorm:"datetime null" json:"expires_at" doc:"Null means no expiry."`
	SkipEmailConfirm bool             `xorm:"not null default false" json:"skip_email_confirm" doc:"Activate invitees without confirming their email."`
	CreatedByID      int64            `xorm:"bigint not null index" json:"created_by_id" doc:"ID of the admin who created this link."`
	Created          time.Time        `xorm:"created not null" json:"created" doc:"Creation timestamp."`
	Updated          time.Time        `xorm:"updated not null" json:"updated" doc:"Last update timestamp."`
	Teams            []InviteLinkTeam `xorm:"-" json:"teams" doc:"Teams the invitee will join."`
	ClearTextToken   string           `xorm:"-" json:"token,omitempty" doc:"Secret token, returned only on creation."`
}

func (UserInviteLink) TableName() string { return "user_invite_links" }

type UserInviteLinkTeam struct {
	ID           int64     `xorm:"bigint autoincr not null unique pk"`
	InviteLinkID int64     `xorm:"bigint not null unique(invite_team)"`
	TeamID       int64     `xorm:"bigint not null index unique(invite_team)"`
	Created      time.Time `xorm:"created not null"`
}

func (UserInviteLinkTeam) TableName() string { return "user_invite_link_teams" }

type CreateInviteLinkBody struct {
	Name             string     `json:"name" maxLength:"250" doc:"Name shown to admins and invitees."`
	TeamIDs          []int64    `json:"team_ids" doc:"Local teams the invitees will join."`
	MaxUses          *int64     `json:"max_uses" doc:"Null allows unlimited registrations."`
	ExpiresAt        *time.Time `json:"expires_at" doc:"Null means no expiry."`
	SkipEmailConfirm bool       `json:"skip_email_confirm" doc:"Activate accounts without confirming email."`
}

func requireInviteLinkAdmin(s *xorm.Session, doer *user.User) error {
	if doer == nil || !license.IsFeatureEnabled(license.FeatureUserInvites) || !isInstanceAdmin(s, doer) {
		return ErrInviteLinkInvalid{}
	}
	return nil
}

func CreateInviteLinkAsAdmin(s *xorm.Session, doer *user.User, body *CreateInviteLinkBody) (*UserInviteLink, error) {
	if err := requireInviteLinkAdmin(s, doer); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(body.Name)
	if name == "" || utf8.RuneCountInString(name) > 250 || (body.MaxUses != nil && *body.MaxUses < 1) || (body.ExpiresAt != nil && !body.ExpiresAt.After(time.Now())) {
		return nil, ErrInvalidInviteLinkInput{}
	}
	teams := make([]InviteLinkTeam, 0, len(body.TeamIDs))
	seen := make(map[int64]bool, len(body.TeamIDs))
	for _, id := range body.TeamIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		team := &Team{}
		found, err := s.ID(id).Get(team)
		if err != nil {
			return nil, fmt.Errorf("load invite team: %w", err)
		}
		if !found {
			return nil, ErrTeamDoesNotExist{TeamID: id}
		}
		if team.ExternalID != "" {
			return nil, ErrInviteLinkExternalTeam{}
		}
		teams = append(teams, InviteLinkTeam{ID: team.ID, Name: team.Name})
	}
	token, err := utils.CryptoRandomString(64)
	if err != nil {
		return nil, fmt.Errorf("generate invite token: %w", err)
	}
	link := &UserInviteLink{Name: name, TokenHash: utils.Sha256Hex(token), MaxUses: body.MaxUses, ExpiresAt: body.ExpiresAt, SkipEmailConfirm: body.SkipEmailConfirm, CreatedByID: doer.ID, Teams: teams}
	if _, err := s.Insert(link); err != nil {
		return nil, fmt.Errorf("create invite link: %w", err)
	}
	for _, team := range teams {
		if _, err := s.Insert(&UserInviteLinkTeam{InviteLinkID: link.ID, TeamID: team.ID}); err != nil {
			return nil, fmt.Errorf("attach invite team: %w", err)
		}
	}
	// Audit events must never carry the clear token.
	auditLink := *link
	events.DispatchOnCommit(s, &AdminInviteLinkCreatedEvent{Link: &auditLink, Doer: doer})
	link.ClearTextToken = token
	return link, nil
}

func loadInviteLinkTeams(s *xorm.Session, link *UserInviteLink) error {
	link.Teams = []InviteLinkTeam{}
	err := s.Table("teams").Select("teams.id, teams.name").
		Join("INNER", "user_invite_link_teams", "teams.id = user_invite_link_teams.team_id").
		Where(builder.Eq{"user_invite_link_teams.invite_link_id": link.ID}).OrderBy("teams.id ASC").Find(&link.Teams)
	if err != nil {
		return fmt.Errorf("load invite teams: %w", err)
	}
	return nil
}

func ListInviteLinksAsAdmin(s *xorm.Session, doer *user.User, page, perPage int) ([]*UserInviteLink, int64, error) {
	if err := requireInviteLinkAdmin(s, doer); err != nil {
		return nil, 0, err
	}
	limit, start := getLimitFromPageIndex(page, perPage)
	links := []*UserInviteLink{}
	total, err := s.Limit(limit, start).OrderBy("id DESC").FindAndCount(&links)
	if err != nil {
		return nil, 0, fmt.Errorf("list invite links: %w", err)
	}
	for _, link := range links {
		if err := loadInviteLinkTeams(s, link); err != nil {
			return nil, 0, err
		}
	}
	return links, total, nil
}

func DeleteInviteLinkAsAdmin(s *xorm.Session, doer *user.User, id int64) error {
	if err := requireInviteLinkAdmin(s, doer); err != nil {
		return err
	}
	link := &UserInviteLink{}
	found, err := s.ID(id).Get(link)
	if err != nil {
		return fmt.Errorf("load invite link: %w", err)
	}
	if !found {
		return ErrInviteLinkDoesNotExist{}
	}
	if _, err := s.Where(builder.Eq{"invite_link_id": id}).Delete(&UserInviteLinkTeam{}); err != nil {
		return fmt.Errorf("delete invite teams: %w", err)
	}
	if _, err := s.ID(id).Delete(&UserInviteLink{}); err != nil {
		return fmt.Errorf("delete invite link: %w", err)
	}
	events.DispatchOnCommit(s, &AdminInviteLinkDeletedEvent{Link: link, Doer: doer})
	return nil
}

func usableInviteLinkQuery(s *xorm.Session, token string) *xorm.Session {
	return s.Where(builder.Eq{"token_hash": utils.Sha256Hex(token)}).
		And(builder.Or(builder.IsNull{"expires_at"}, builder.Gt{"expires_at": time.Now()})).
		And("max_uses IS NULL OR uses < max_uses")
}

func GetInviteLinkByToken(s *xorm.Session, token string) (*UserInviteLink, error) {
	if !license.IsFeatureEnabled(license.FeatureUserInvites) {
		return nil, ErrInviteLinkInvalid{}
	}
	link := &UserInviteLink{}
	found, err := usableInviteLinkQuery(s, token).Get(link)
	if err != nil {
		return nil, fmt.Errorf("get invite link: %w", err)
	}
	if !found {
		return nil, ErrInviteLinkInvalid{}
	}
	if err := loadInviteLinkTeams(s, link); err != nil {
		return nil, err
	}
	return link, nil
}

func RegisterUserViaInviteLink(s *xorm.Session, token string, u *user.User) (*user.User, error) {
	if !license.IsFeatureEnabled(license.FeatureUserInvites) {
		return nil, ErrInviteLinkInvalid{}
	}
	claimed, err := usableInviteLinkQuery(s, token).Incr("uses").Update(&UserInviteLink{})
	if err != nil {
		return nil, fmt.Errorf("claim invite link use: %w", err)
	}
	if claimed == 0 {
		return nil, ErrInviteLinkInvalid{}
	}
	link := &UserInviteLink{}
	found, err := s.Where(builder.Eq{"token_hash": utils.Sha256Hex(token)}).Get(link)
	if err != nil {
		return nil, fmt.Errorf("load claimed invite link: %w", err)
	}
	if !found {
		return nil, ErrInviteLinkInvalid{}
	}
	created, err := RegisterUser(s, u, user.CreateUserOptions{SkipEmailConfirm: link.SkipEmailConfirm})
	if err != nil {
		return nil, err
	}
	if err := loadInviteLinkTeams(s, link); err != nil {
		return nil, err
	}
	for _, team := range link.Teams {
		if _, err := s.Insert(&TeamMember{TeamID: team.ID, UserID: created.ID}); err != nil {
			return nil, fmt.Errorf("join invited team: %w", err)
		}
		events.DispatchOnCommit(s, &TeamMemberAddedEvent{Team: &Team{ID: team.ID, Name: team.Name}, Member: created, Doer: created})
	}
	return created, nil
}
