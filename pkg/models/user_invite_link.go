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

import "time"

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
