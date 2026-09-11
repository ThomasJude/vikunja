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

package user

import (
	"encoding/json"
	"fmt"

	"code.vikunja.io/api/pkg/events"
	"code.vikunja.io/api/pkg/notifications"
	"github.com/ThreeDotsLabs/watermill/message"
)

type EmailConfirmationRequestedEvent struct {
	User  *User  `json:"user"`
	Token string `json:"token"`
}

func (e *EmailConfirmationRequestedEvent) Name() string { return "user.email_confirmation.requested" }

type SendEmailConfirmation struct{}

func (l *SendEmailConfirmation) Name() string { return "user.email_confirmation.send" }

func (l *SendEmailConfirmation) Handle(msg *message.Message) error {
	event := &EmailConfirmationRequestedEvent{}
	if err := json.Unmarshal(msg.Payload, event); err != nil {
		return fmt.Errorf("decode email confirmation: %w", err)
	}
	return notifications.Notify(event.User, &EmailConfirmNotification{User: event.User, IsNew: true, ConfirmToken: event.Token})
}

func init() {
	events.RegisterListener((&EmailConfirmationRequestedEvent{}).Name(), &SendEmailConfirmation{})
}
