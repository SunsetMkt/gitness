// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package execution

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/pipeline/canceler"
	"github.com/harness/gitness/app/pipeline/commit"
	"github.com/harness/gitness/app/pipeline/triggerer"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type Controller struct {
	tx             dbtx.Transactor
	authorizer     authz.Authorizer
	executionStore store.ExecutionStore
	checkStore     store.CheckStore
	canceler       canceler.Canceler
	commitService  commit.Service
	triggerer      triggerer.Triggerer
	stageStore     store.StageStore
	pipelineStore  store.PipelineStore
	repoFinder     refcache.RepoFinder
}

func NewController(
	tx dbtx.Transactor,
	authorizer authz.Authorizer,
	executionStore store.ExecutionStore,
	checkStore store.CheckStore,
	canceler canceler.Canceler,
	commitService commit.Service,
	triggerer triggerer.Triggerer,
	stageStore store.StageStore,
	pipelineStore store.PipelineStore,
	repoFinder refcache.RepoFinder,
) *Controller {
	return &Controller{
		tx:             tx,
		authorizer:     authorizer,
		executionStore: executionStore,
		checkStore:     checkStore,
		canceler:       canceler,
		commitService:  commitService,
		triggerer:      triggerer,
		stageStore:     stageStore,
		pipelineStore:  pipelineStore,
		repoFinder:     repoFinder,
	}
}

// getRepoCheckPipelineAccess fetches a repo, checks if the permission is allowed based on the repo state,
// and checks if the current user has permission to access pipelines belong to it.
//
//nolint:unparam
func (c *Controller) getRepoCheckPipelineAccess(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pipelineIdentifier string,
	reqPermission enum.Permission,
	allowedRepoStates ...enum.RepoState,
) (*types.Repository, error) {
	repo, err := c.repoFinder.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo by ref: %w", err)
	}

	if err := apiauth.CheckRepoState(ctx, session, repo, reqPermission, allowedRepoStates...); err != nil {
		return nil, err
	}

	err = apiauth.CheckPipeline(
		ctx,
		c.authorizer,
		session,
		repo.Path,
		pipelineIdentifier,
		reqPermission)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize: %w", err)
	}

	return repo, nil
}
