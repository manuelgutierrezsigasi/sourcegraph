package batches

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	apiclient "github.com/sourcegraph/sourcegraph/enterprise/internal/executor"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	batcheslib "github.com/sourcegraph/sourcegraph/lib/batches"
	"github.com/sourcegraph/sourcegraph/lib/batches/execution/cache"
	"github.com/sourcegraph/sourcegraph/lib/batches/template"
)

type BatchesStore interface {
	GetBatchSpecWorkspace(context.Context, store.GetBatchSpecWorkspaceOpts) (*btypes.BatchSpecWorkspace, error)
	GetBatchSpec(context.Context, store.GetBatchSpecOpts) (*btypes.BatchSpec, error)
	SetBatchSpecWorkspaceExecutionJobAccessToken(ctx context.Context, jobID, tokenID int64) error
	ListBatchSpecExecutionCacheEntries(ctx context.Context, opts store.ListBatchSpecExecutionCacheEntriesOpts) ([]*btypes.BatchSpecExecutionCacheEntry, error)

	DatabaseDB() database.DB
}

// transformRecord transforms a *btypes.BatchSpecWorkspaceExecutionJob into an apiclient.Job.
func transformRecord(ctx context.Context, s BatchesStore, job *btypes.BatchSpecWorkspaceExecutionJob, accessToken string) (apiclient.Job, error) {
	// MAYBE: We could create a view in which batch_spec and repo are joined
	// against the batch_spec_workspace_job so we don't have to load them
	// separately.
	workspace, err := s.GetBatchSpecWorkspace(ctx, store.GetBatchSpecWorkspaceOpts{ID: job.BatchSpecWorkspaceID})
	if err != nil {
		return apiclient.Job{}, errors.Wrapf(err, "fetching workspace %d", job.BatchSpecWorkspaceID)
	}

	batchSpec, err := s.GetBatchSpec(ctx, store.GetBatchSpecOpts{ID: workspace.BatchSpecID})
	if err != nil {
		return apiclient.Job{}, errors.Wrap(err, "fetching batch spec")
	}

	// 🚨 SECURITY: Set the actor on the context so we check for permissions
	// when loading the repository.
	ctx = actor.WithActor(ctx, actor.FromUser(batchSpec.UserID))

	repo, err := s.DatabaseDB().Repos().Get(ctx, workspace.RepoID)
	if err != nil {
		return apiclient.Job{}, errors.Wrap(err, "fetching repo")
	}

	// Create an internal access token that will get cleaned up when the job
	// finishes.
	token, err := createAndAttachInternalAccessToken(ctx, s, job.ID, batchSpec.UserID)
	if err != nil {
		return apiclient.Job{}, errors.Wrap(err, "creating internal access token")
	}

	executionInput := batcheslib.WorkspacesExecutionInput{
		Spec: batchSpec.Spec,
		Workspace: batcheslib.Workspace{
			Repository: batcheslib.WorkspaceRepo{
				ID:   string(graphqlbackend.MarshalRepositoryID(repo.ID)),
				Name: string(repo.Name),
			},
			Branch: batcheslib.WorkspaceBranch{
				Name:   workspace.Branch,
				Target: batcheslib.Commit{OID: workspace.Commit},
			},
			Path:               workspace.Path,
			OnlyFetchWorkspace: workspace.OnlyFetchWorkspace,
			Steps:              workspace.Steps,
			SearchResultPaths:  workspace.FileMatches,
		},
	}

	frontendURL := conf.Get().ExternalURL

	srcEndpoint, err := makeURL(frontendURL, accessToken)
	if err != nil {
		return apiclient.Job{}, err
	}

	redactedSrcEndpoint, err := makeURL(frontendURL, "PASSWORD_REMOVED")
	if err != nil {
		return apiclient.Job{}, err
	}

	cliEnv := []string{
		fmt.Sprintf("SRC_ENDPOINT=%s", srcEndpoint),
		fmt.Sprintf("SRC_ACCESS_TOKEN=%s", token),
	}

	marshaledInput, err := json.Marshal(executionInput)
	if err != nil {
		return apiclient.Job{}, err
	}

	files := map[string]string{"input.json": string(marshaledInput)}

	if !batchSpec.NoCache {
		// We start at the back so that we can find the _last_ cached step,
		// then restart execution on the following step.
		taskKey := cache.KeyForWorkspace(
			&template.BatchChangeAttributes{
				Name:        batchSpec.Spec.Name,
				Description: batchSpec.Spec.Description,
			},
			batcheslib.Repository{
				ID:          string(graphqlbackend.MarshalRepositoryID(workspace.RepoID)),
				Name:        string(repo.Name),
				BaseRef:     workspace.Branch,
				BaseRev:     workspace.Commit,
				FileMatches: workspace.FileMatches,
			},
			workspace.Path,
			workspace.OnlyFetchWorkspace,
			workspace.Steps,
		)

		for i := len(workspace.Steps) - 1; i > -1; i-- {
			key := cache.StepsCacheKey{ExecutionKey: &taskKey, StepIndex: i}
			rawKey, err := key.Key()
			if err != nil {
				return apiclient.Job{}, nil
			}
			// TODO: Once implemented, enforce ownership of cache entries here.
			entries, err := s.ListBatchSpecExecutionCacheEntries(ctx, store.ListBatchSpecExecutionCacheEntriesOpts{
				Keys: []string{rawKey},
			})
			if err != nil {
				return apiclient.Job{}, err
			}
			if len(entries) != 1 {
				continue
			}

			// Add file to virtualMachineFiles.
			files[rawKey+`.json`] = entries[0].Value
			// And break after. src-cli only needs the most recent cache entry.
			break
		}
	}

	return apiclient.Job{
		ID:                  int(job.ID),
		VirtualMachineFiles: files,
		CliSteps: []apiclient.CliStep{
			{
				Commands: []string{"batch", "exec", "-f", "input.json"},
				Dir:      ".",
				Env:      cliEnv,
			},
		},
		RedactedValues: map[string]string{
			// 🚨 SECURITY: Catch leak of upload endpoint. This is necessary in addition
			// to the below in case the username or password contains illegal URL characters,
			// which are then urlencoded and are not replaceable via byte comparison.
			srcEndpoint: redactedSrcEndpoint,

			// 🚨 SECURITY: Catch uses of fragments pulled from URL to construct another target
			// (in src-cli). We only pass the constructed URL to src-cli, which we trust not to
			// ship the values to a third party, but not to trust to ensure the values are absent
			// from the command's stdout or stderr streams.
			accessToken: "PASSWORD_REMOVED",

			// 🚨 SECURITY: Redact the access token used for src-cli to talk to
			// Sourcegraph instance.
			token: "SRC_ACCESS_TOKEN_REMOVED",
		},
	}, nil
}

const (
	accessTokenNote  = "batch-spec-execution"
	accessTokenScope = "user:all"
)

func createAndAttachInternalAccessToken(ctx context.Context, s BatchesStore, jobID int64, userID int32) (string, error) {
	tokenID, token, err := s.DatabaseDB().AccessTokens().CreateInternal(ctx, userID, []string{accessTokenScope}, accessTokenNote, userID)
	if err != nil {
		return "", err
	}
	if err := s.SetBatchSpecWorkspaceExecutionJobAccessToken(ctx, jobID, tokenID); err != nil {
		return "", err
	}
	return token, nil
}

func makeURL(base, password string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	u.User = url.UserPassword("sourcegraph", password)
	return u.String(), nil
}
