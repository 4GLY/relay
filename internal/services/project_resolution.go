package services

import (
	"context"
	"path/filepath"
	"strings"

	"relay/internal/domain"
	"relay/internal/lib"
)

func (s Service) resolveProject(ctx context.Context, name string, id string) (domain.Project, error) {
	if auth, ok := AuthInfoFromContext(ctx); ok && NormalizeAPIKeyScope(auth.Scope) == APIKeyScopeProject {
		return s.resolveBoundProject(ctx, auth, name, id)
	}
	if id != "" {
		project, err := s.deps.Projects.GetByID(ctx, id)
		if err != nil {
			return domain.Project{}, err
		}
		if name != "" && project.Name != name {
			return domain.Project{}, lib.AppError{
				Code:      "PROJECT_MISMATCH",
				Message:   "project and project_id do not match",
				Retryable: false,
			}
		}
		return project, nil
	}
	if name == "" {
		return domain.Project{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "project")
	}
	return s.deps.Projects.GetByName(ctx, name)
}

func (s Service) resolveCaptureProject(ctx context.Context, name string, repoPath string) (domain.Project, error) {
	if auth, ok := AuthInfoFromContext(ctx); ok && NormalizeAPIKeyScope(auth.Scope) == APIKeyScopeProject {
		if auth.ProjectID == "" {
			return domain.Project{}, lib.Forbidden("FORBIDDEN", "project-scoped api key is missing a project binding")
		}
		project, err := s.deps.Projects.GetByID(ctx, auth.ProjectID)
		if err != nil {
			return domain.Project{}, err
		}
		if name != "" && project.Name != name {
			return domain.Project{}, lib.Forbidden("FORBIDDEN", "api key is not authorized for this project")
		}
		if repoPath != "" && !projectRootPathMatches(repoPath, project.RootPath) {
			return domain.Project{}, lib.Forbidden("FORBIDDEN", "api key is not authorized for this project")
		}
		return project, nil
	}

	project, err := s.deps.Projects.EnsureProject(ctx, domain.Project{
		ID:       lib.ProjectID(name),
		Name:     name,
		RootPath: repoPath,
		Status:   "active",
	})
	if err != nil {
		return domain.Project{}, err
	}
	if err := s.enforceProjectAccess(ctx, project.ID); err != nil {
		return domain.Project{}, err
	}
	return project, nil
}

func projectRootPathMatches(repoPath string, rootPath string) bool {
	if strings.TrimSpace(repoPath) == "" || strings.TrimSpace(rootPath) == "" {
		return false
	}
	return filepath.Clean(repoPath) == filepath.Clean(rootPath)
}

func (s Service) resolveBoundProject(ctx context.Context, auth AuthInfo, name string, id string) (domain.Project, error) {
	if auth.ProjectID == "" {
		return domain.Project{}, lib.Forbidden("FORBIDDEN", "project-scoped api key is missing a project binding")
	}

	project, err := s.deps.Projects.GetByID(ctx, auth.ProjectID)
	if err != nil {
		return domain.Project{}, err
	}
	if id != "" && id != auth.ProjectID {
		return domain.Project{}, lib.Forbidden("FORBIDDEN", "api key is not authorized for this project")
	}
	if name != "" && project.Name != name {
		return domain.Project{}, lib.Forbidden("FORBIDDEN", "api key is not authorized for this project")
	}
	return project, nil
}

func (s Service) enforceProjectAccess(ctx context.Context, projectID string) error {
	auth, ok := AuthInfoFromContext(ctx)
	if !ok {
		return nil
	}
	if NormalizeAPIKeyScope(auth.Scope) != APIKeyScopeProject {
		return nil
	}
	if auth.ProjectID == "" {
		return lib.Forbidden("FORBIDDEN", "project-scoped api key is missing a project binding")
	}
	if auth.ProjectID != projectID {
		return lib.Forbidden("FORBIDDEN", "api key is not authorized for this project")
	}
	return nil
}

func requireAdminAuth(ctx context.Context) error {
	auth, ok := AuthInfoFromContext(ctx)
	if !ok {
		return lib.Forbidden("FORBIDDEN", "admin authorization is required")
	}
	if auth.IsAdmin {
		return nil
	}
	return lib.Forbidden("FORBIDDEN", "admin authorization is required")
}
