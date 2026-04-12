package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v69/github"

	"scuffinger/internal/i18n"
	"scuffinger/internal/logging"
	"scuffinger/internal/metrics"
)

// ClientProvider is any type that can return the current GitHub API client.
type ClientProvider interface {
	Client() *github.Client
}

// GitHubHandler handles all /api/github/* HTTP endpoints.
type GitHubHandler struct {
	provider ClientProvider
	org      string
	log      *logging.Logger
}

// NewGitHubHandler creates a new GitHubHandler.
func NewGitHubHandler(provider ClientProvider, org string, log *logging.Logger) *GitHubHandler {
	return &GitHubHandler{provider: provider, org: org, log: log}
}

// RegisterRoutes implements RouteRegistrar.
func (h *GitHubHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/github")
	{
		api.GET("/users/:username", h.GetUser)
		api.GET("/orgs/:org", h.GetOrg)
		api.GET("/repos/:owner/:repo", h.GetRepo)
		api.GET("/repos/:owner/:repo/branches", h.ListBranches)
		api.GET("/repos/:owner/:repo/workflows", h.ListWorkflows)
		api.GET("/repos/:owner/:repo/workflows/:workflow_id/runs", h.ListWorkflowRuns)
		api.GET("/rate-limit", h.GetRateLimit)
	}
}

// ── Handlers ─────────────────────────────────────────────────────────────────

// GetUser returns information about a GitHub user.
func (h *GitHubHandler) GetUser(c *gin.Context) {
	username := c.Param("username")
	h.log.Debug("Fetching GitHub user", "username", username)

	start := time.Now()
	user, _, err := h.provider.Client().Users.Get(c.Request.Context(), username)
	metrics.ObserveGitHubCall("get_user", time.Since(start), err)
	if err != nil {
		h.log.Error(i18n.Get(i18n.ErrGhFetchUser), "username", username, "error", err)
		ghError(c, i18n.ErrGhFetchUser, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

// GetOrg returns information about a GitHub organization.
func (h *GitHubHandler) GetOrg(c *gin.Context) {
	org := c.Param("org")
	h.log.Debug("Fetching GitHub organization", "org", org)

	start := time.Now()
	organization, _, err := h.provider.Client().Organizations.Get(c.Request.Context(), org)
	metrics.ObserveGitHubCall("get_org", time.Since(start), err)
	if err != nil {
		h.log.Error(i18n.Get(i18n.ErrGhFetchOrg), "org", org, "error", err)
		ghError(c, i18n.ErrGhFetchOrg, err)
		return
	}
	c.JSON(http.StatusOK, organization)
}

// GetRepo returns information about a GitHub repository.
func (h *GitHubHandler) GetRepo(c *gin.Context) {
	owner, repo := c.Param("owner"), c.Param("repo")
	h.log.Debug("Fetching GitHub repo", "owner", owner, "repo", repo)

	start := time.Now()
	repository, _, err := h.provider.Client().Repositories.Get(c.Request.Context(), owner, repo)
	metrics.ObserveGitHubCall("get_repo", time.Since(start), err)
	if err != nil {
		h.log.Error(i18n.Get(i18n.ErrGhFetchRepo), "owner", owner, "repo", repo, "error", err)
		ghError(c, i18n.ErrGhFetchRepo, err)
		return
	}
	c.JSON(http.StatusOK, repository)
}

// ListBranches returns all branches of a repository (paginated).
func (h *GitHubHandler) ListBranches(c *gin.Context) {
	owner, repo := c.Param("owner"), c.Param("repo")
	h.log.Debug("Fetching branches", "owner", owner, "repo", repo)

	var all []*github.Branch
	opts := &github.BranchListOptions{ListOptions: github.ListOptions{PerPage: 100}}

	start := time.Now()
	for {
		branches, resp, err := h.provider.Client().Repositories.ListBranches(c.Request.Context(), owner, repo, opts)
		if err != nil {
			metrics.ObserveGitHubCall("list_branches", time.Since(start), err)
			h.log.Error(i18n.Get(i18n.ErrGhFetchBranches), "owner", owner, "repo", repo, "error", err)
			ghError(c, i18n.ErrGhFetchBranches, err)
			return
		}
		all = append(all, branches...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	metrics.ObserveGitHubCall("list_branches", time.Since(start), nil)

	c.JSON(http.StatusOK, gin.H{"total_count": len(all), "branches": all})
}

// ListWorkflows returns all workflows of a repository (paginated).
func (h *GitHubHandler) ListWorkflows(c *gin.Context) {
	owner, repo := c.Param("owner"), c.Param("repo")
	h.log.Debug("Fetching workflows", "owner", owner, "repo", repo)

	var all []*github.Workflow
	opts := &github.ListOptions{PerPage: 100}

	start := time.Now()
	for {
		workflows, resp, err := h.provider.Client().Actions.ListWorkflows(c.Request.Context(), owner, repo, opts)
		if err != nil {
			metrics.ObserveGitHubCall("list_workflows", time.Since(start), err)
			h.log.Error(i18n.Get(i18n.ErrGhFetchWorkflows), "owner", owner, "repo", repo, "error", err)
			ghError(c, i18n.ErrGhFetchWorkflows, err)
			return
		}
		all = append(all, workflows.Workflows...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	metrics.ObserveGitHubCall("list_workflows", time.Since(start), nil)

	c.JSON(http.StatusOK, gin.H{"total_count": len(all), "workflows": all})
}

// ListWorkflowRuns returns runs for a specific workflow (paginated).
func (h *GitHubHandler) ListWorkflowRuns(c *gin.Context) {
	owner, repo := c.Param("owner"), c.Param("repo")
	wfIDStr := c.Param("workflow_id")

	wfID, err := strconv.ParseInt(wfIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   i18n.Get(i18n.ErrGhInvalidWorkflow),
			"details": err.Error(),
		})
		return
	}

	h.log.Debug("Fetching workflow runs", "owner", owner, "repo", repo, "workflow_id", wfID)

	var all []*github.WorkflowRun
	opts := &github.ListWorkflowRunsOptions{ListOptions: github.ListOptions{PerPage: 100}}

	start := time.Now()
	for {
		runs, resp, err := h.provider.Client().Actions.ListWorkflowRunsByID(c.Request.Context(), owner, repo, wfID, opts)
		if err != nil {
			metrics.ObserveGitHubCall("list_workflow_runs", time.Since(start), err)
			h.log.Error(i18n.Get(i18n.ErrGhFetchRuns), "workflow_id", wfID, "error", err)
			ghError(c, i18n.ErrGhFetchRuns, err)
			return
		}
		all = append(all, runs.WorkflowRuns...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	metrics.ObserveGitHubCall("list_workflow_runs", time.Since(start), nil)

	c.JSON(http.StatusOK, gin.H{"total_count": len(all), "workflow_runs": all})
}

// GetRateLimit returns the current GitHub API rate limit status.
func (h *GitHubHandler) GetRateLimit(c *gin.Context) {
	start := time.Now()
	limits, _, err := h.provider.Client().RateLimit.Get(c.Request.Context())
	metrics.ObserveGitHubCall("get_rate_limit", time.Since(start), err)
	if err != nil {
		h.log.Error(i18n.Get(i18n.ErrGhFetchRateLimit), "error", err)
		ghError(c, i18n.ErrGhFetchRateLimit, err)
		return
	}
	metrics.SetGitHubRateLimit("api", limits.Core.Remaining)
	c.JSON(http.StatusOK, limits)
}

// ── helpers ──────────────────────────────────────────────────────────────────

// ghError writes a translated error response, mapping GitHub 404s to HTTP 404.
func ghError(c *gin.Context, key i18n.Key, err error) {
	status := http.StatusBadGateway
	if ghErr, ok := err.(*github.ErrorResponse); ok {
		status = ghErr.Response.StatusCode
	}
	c.JSON(status, gin.H{
		"error":   i18n.Get(key),
		"details": err.Error(),
	})
}
