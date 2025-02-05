package stage

import (
	"context"
	"fmt"

	"github.com/werf/werf/pkg/container_runtime"
	"github.com/werf/werf/pkg/image"
	"github.com/werf/werf/pkg/util"
)

func NewGitLatestPatchStage(gitPatchStageOptions *NewGitPatchStageOptions, baseStageOptions *NewBaseStageOptions) *GitLatestPatchStage {
	s := &GitLatestPatchStage{}
	s.GitPatchStage = newGitPatchStage(GitLatestPatch, gitPatchStageOptions, baseStageOptions)
	return s
}

type GitLatestPatchStage struct {
	*GitPatchStage
}

func (s *GitLatestPatchStage) IsEmpty(ctx context.Context, c Conveyor, prevBuiltImage container_runtime.LegacyImageInterface) (bool, error) {
	if empty, err := s.GitPatchStage.IsEmpty(ctx, c, prevBuiltImage); err != nil {
		return false, err
	} else if empty {
		return true, nil
	}

	isEmpty := true
	for _, gitMapping := range s.gitMappings {
		commit, err := gitMapping.GetBaseCommitForPrevBuiltImage(ctx, c, prevBuiltImage)
		if err != nil {
			return false, fmt.Errorf("unable to get base commit for git mapping %s: %s", gitMapping.GitRepo().GetName(), err)
		}

		if exist, err := gitMapping.GitRepo().IsCommitExists(ctx, commit); err != nil {
			return false, fmt.Errorf("unable to check existence of commit %q in the repo %s: %s", commit, gitMapping.GitRepo().GetName(), err)
		} else if !exist {
			return false, fmt.Errorf("commit %q is not exist in the repo %s", commit, gitMapping.GitRepo().GetName())
		}

		if empty, err := gitMapping.IsPatchEmpty(ctx, c, prevBuiltImage); err != nil {
			return false, err
		} else if !empty {
			isEmpty = false
			break
		}
	}

	return isEmpty, nil
}

func (s *GitLatestPatchStage) GetDependencies(ctx context.Context, c Conveyor, _, prevBuiltImage container_runtime.LegacyImageInterface) (string, error) {
	var args []string

	for _, gitMapping := range s.gitMappings {
		patchContent, err := gitMapping.GetPatchContent(ctx, c, prevBuiltImage)
		if err != nil {
			return "", fmt.Errorf("error getting patch between previous built image %s and current commit for git mapping %s: %s", prevBuiltImage.Name(), gitMapping.Name, err)
		}

		args = append(args, patchContent)
	}

	return util.Sha256Hash(args...), nil
}

func (s *GitLatestPatchStage) SelectSuitableStage(ctx context.Context, c Conveyor, stages []*image.StageDescription) (*image.StageDescription, error) {
	ancestorsImages, err := s.selectStagesAncestorsByGitMappings(ctx, c, stages)
	if err != nil {
		return nil, fmt.Errorf("unable to select cache images ancestors by git mappings: %s", err)
	}
	return s.selectStageByOldestCreationTimestamp(ancestorsImages)
}

func (s *GitLatestPatchStage) GetNextStageDependencies(ctx context.Context, c Conveyor) (string, error) {
	return s.BaseStage.getNextStageGitDependencies(ctx, c)
}
