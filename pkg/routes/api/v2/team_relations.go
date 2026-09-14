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

package apiv2

import (
	"context"
	"fmt"
	"net/http"

	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/web/handler"

	"github.com/danielgtaylor/huma/v2"
)

type teamChildListBody struct {
	Body Paginated[*models.Team]
}

func RegisterTeamRelationRoutes(api huma.API) {
	tags := []string{"teams"}

	Register(api, huma.Operation{
		OperationID: "teams-children-list",
		Summary:     "List a team's child teams",
		Description: "Returns the teams nested directly below this team. Only a team admin may list child teams.",
		Method:      http.MethodGet,
		Path:        "/teams/{team}/children",
		Tags:        tags,
	}, teamChildrenList)

	Register(api, huma.Operation{
		OperationID: "teams-children-add",
		Summary:     "Add a child team",
		Description: "Nests a team below another team. The user must be an admin of both teams.",
		Method:      http.MethodPost,
		Path:        "/teams/{team}/children",
		Tags:        tags,
	}, teamChildrenAdd)

	Register(api, huma.Operation{
		OperationID: "teams-children-remove",
		Summary:     "Remove a child team",
		Description: "Removes a nested team relation. An admin of either team may remove the relation.",
		Method:      http.MethodDelete,
		Path:        "/teams/{team}/children/{child}",
		Tags:        tags,
	}, teamChildrenRemove)
}

func init() { AddRouteRegistrar(RegisterTeamRelationRoutes) }

func teamChildrenList(ctx context.Context, in *struct {
	TeamID int64 `path:"team"`
	ListParams
}) (*teamChildListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	result, _, total, err := handler.DoReadAll(
		ctx,
		&models.TeamRelation{ParentTeamID: in.TeamID},
		a,
		in.Q,
		in.Page,
		in.PerPage,
	)
	if err != nil {
		return nil, translateDomainError(err)
	}

	items, ok := result.([]*models.Team)
	if !ok {
		return nil, fmt.Errorf("teamRelations.ReadAll returned unexpected type %T (expected []*models.Team)", result)
	}

	return &teamChildListBody{
		Body: NewPaginated(items, total, in.Page, in.PerPage),
	}, nil
}

func teamChildrenAdd(ctx context.Context, in *struct {
	TeamID int64 `path:"team"`
	Body   models.TeamRelation
}) (*singleBody[models.TeamRelation], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	in.Body.ParentTeamID = in.TeamID

	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}

	return &singleBody[models.TeamRelation]{Body: &in.Body}, nil
}

func teamChildrenRemove(ctx context.Context, in *struct {
	TeamID  int64 `path:"team"`
	ChildID int64 `path:"child"`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	relation := &models.TeamRelation{
		ParentTeamID: in.TeamID,
		ChildTeamID:  in.ChildID,
	}

	if err := handler.DoDelete(ctx, relation, a); err != nil {
		return nil, translateDomainError(err)
	}

	return &emptyBody{}, nil
}
