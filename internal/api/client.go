package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
)

var DefaultBaseURL = "https://api.pachim.sh"

const WorkspaceContextHeader = "X-Workspace-Id"

type Client struct {
	BaseURL     string
	Token       string
	WorkspaceID string
	HTTPClient  *http.Client
}

type APIResponse struct {
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message"`
	Status  int             `json:"status"`
	Errors  json.RawMessage `json:"errors"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"user"`
}

type Site struct {
	ID            string `json:"id"`
	Domain        string `json:"domain"`
	AppType       string `json:"app_type"`
	SetupType     string `json:"setup_type"`
	RepoStatus    string `json:"repo_status"`
	DeployBranch  string `json:"deploy_branch"`
	GitMerge      bool   `json:"git_merge"`
	WorkspaceID   string `json:"workspace_id,omitempty"`
	WorkspaceName string `json:"workspace_name,omitempty"`
	Server        struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		IP   string `json:"ip"`
	} `json:"server"`
}

type CatalogWorkspace struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Slug            string  `json:"slug"`
	Type            string  `json:"type"`
	IsPersonal      bool    `json:"is_personal"`
	CurrentUserRole string  `json:"current_user_role"`
	Sites           []Site  `json:"sites"`
}

type Catalog struct {
	Workspaces []CatalogWorkspace `json:"workspaces"`
}

type ActiveDeployment struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	StartedAt   string `json:"started_at"`
	InitiatedBy string `json:"initiated_by"`
}

type SiteInfo struct {
	ID               string            `json:"id"`
	Domain           string            `json:"domain"`
	SetupType        string            `json:"setup_type"`
	DeployBranch     string            `json:"deploy_branch"`
	GitMerge         bool              `json:"git_merge"`
	ActiveDeployment *ActiveDeployment `json:"active_deployment"`
}

type DeployResponse struct {
	DeploymentID string `json:"deployment_id"`
	Status       string `json:"status"`
}

type DeployUploadOptions struct {
	Branch              string
	CommitHash          string
	SaveBranchAsDefault bool
}

type DeploymentStatus struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	Type       string `json:"type"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at"`
	Output     string `json:"output"`
}

type DeploymentListItem struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Type        string `json:"type"`
	InitiatedBy string `json:"initiated_by"`
	StartedAt   string `json:"started_at"`
	FinishedAt  string `json:"finished_at"`
	CreatedAt   string `json:"created_at"`
}

type DeploymentsListResponse struct {
	SiteID      string               `json:"site_id"`
	Domain      string               `json:"domain"`
	Deployments []DeploymentListItem `json:"deployments"`
}

type Workspace struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	Type            string `json:"type"`
	IsPersonal      bool   `json:"is_personal"`
	CurrentUserRole string `json:"current_user_role"`
}

func NewClient(baseURL, token string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 20000 * time.Second,
		},
	}
}

func (c *Client) WithWorkspace(workspaceID string) *Client {
	clone := *c
	clone.WorkspaceID = strings.TrimSpace(workspaceID)

	return &clone
}

func (c *Client) applyAuthHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if strings.TrimSpace(c.WorkspaceID) != "" {
		req.Header.Set(WorkspaceContextHeader, strings.TrimSpace(c.WorkspaceID))
	}
}

func (c *Client) Login(email, password string) (*LoginResponse, error) {
	body := map[string]string{
		"email":    email,
		"password": password,
	}

	resp, err := c.postJSON("/cli/auth/login", body)
	if err != nil {
		return nil, err
	}

	var result LoginResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse login response: %w", err)
	}

	return &result, nil
}

func (c *Client) Logout() error {
	_, err := c.postJSON("/cli/auth/logout", nil)
	return err
}

func (c *Client) ListWorkspaces() ([]Workspace, error) {
	resp, err := c.get("/cli/workspaces")
	if err != nil {
		return nil, err
	}

	var workspaces []Workspace
	if err := json.Unmarshal(resp.Data, &workspaces); err != nil {
		return nil, fmt.Errorf("failed to parse workspaces: %w", err)
	}

	return workspaces, nil
}

func (c *Client) GetCurrentWorkspace() (*Workspace, error) {
	resp, err := c.get("/cli/workspaces/current")
	if err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 || string(resp.Data) == "null" {
		return nil, nil
	}

	var workspace Workspace
	if err := json.Unmarshal(resp.Data, &workspace); err != nil {
		return nil, fmt.Errorf("failed to parse current workspace: %w", err)
	}

	return &workspace, nil
}

func (c *Client) ListCatalog() (*Catalog, error) {
	resp, err := c.get("/cli/catalog")
	if err != nil {
		return nil, err
	}

	var catalog Catalog
	if err := json.Unmarshal(resp.Data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse catalog: %w", err)
	}

	return &catalog, nil
}

func (c *Client) ListSites() ([]Site, error) {
	resp, err := c.get("/cli/sites")
	if err != nil {
		return nil, err
	}

	var sites []Site
	if err := json.Unmarshal(resp.Data, &sites); err != nil {
		return nil, fmt.Errorf("failed to parse sites: %w", err)
	}

	return sites, nil
}

func (c *Client) GetSiteInfo(siteID string) (*SiteInfo, error) {
	resp, err := c.get(fmt.Sprintf("/cli/sites/%s/info", siteID))
	if err != nil {
		return nil, err
	}

	var info SiteInfo
	if err := json.Unmarshal(resp.Data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse site info: %w", err)
	}

	return &info, nil
}

func (c *Client) ToggleGitMerge(siteID string) (bool, error) {
	resp, err := c.postJSON(fmt.Sprintf("/cli/sites/%s/toggle-git-merge", siteID), nil)
	if err != nil {
		return false, err
	}

	var result struct {
		GitMerge bool `json:"git_merge"`
	}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return false, err
	}

	return result.GitMerge, nil
}

func (c *Client) Deploy(siteID, zipPath string, opts *DeployUploadOptions) (*DeployResponse, error) {
	return c.DeployWithProgress(siteID, zipPath, opts, nil)
}

func (c *Client) DeployWithProgress(siteID, zipPath string, opts *DeployUploadOptions, onProgress func(int)) (*DeployResponse, error) {
	file, err := os.Open(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	errCh := make(chan error, 1)

	go func() {
		defer pw.Close()

		if opts != nil {
			if opts.Branch != "" {
				if err := writer.WriteField("branch", opts.Branch); err != nil {
					errCh <- err
					return
				}
			}
			if opts.CommitHash != "" {
				if err := writer.WriteField("commit_hash", opts.CommitHash); err != nil {
					errCh <- err
					return
				}
			}
			if opts.SaveBranchAsDefault {
				if err := writer.WriteField("save_branch_as_default", "1"); err != nil {
					errCh <- err
					return
				}
			}
		}

		part, err := writer.CreateFormFile("project_zip", "project.zip")
		if err != nil {
			errCh <- err
			return
		}

		reader := io.Reader(file)
		if onProgress != nil {
			reader = &progressReader{
				reader:     file,
				total:      info.Size(),
				onProgress: onProgress,
			}
		}

		if _, err := io.Copy(part, reader); err != nil {
			errCh <- err
			return
		}

		if err := writer.Close(); err != nil {
			errCh <- err
			return
		}

		errCh <- nil
	}()

	url := fmt.Sprintf("%s/cli/sites/%s/deploy", c.BaseURL, siteID)
	req, err := http.NewRequest(http.MethodPost, url, pr)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	c.applyAuthHeaders(req)

	httpResp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if copyErr := <-errCh; copyErr != nil {
		return nil, fmt.Errorf("failed to prepare upload: %w", copyErr)
	}

	respBody, _ := io.ReadAll(httpResp.Body)

	if httpResp.StatusCode != http.StatusOK {
		return nil, parseHTTPError(httpResp.StatusCode, respBody)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var result DeployResponse
	if err := json.Unmarshal(apiResp.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse deploy response: %w", err)
	}

	if onProgress != nil {
		onProgress(100)
	}

	return &result, nil
}

func (c *Client) GetDeploymentStatus(siteID, deploymentID string) (*DeploymentStatus, error) {
	url := fmt.Sprintf("/cli/sites/%s/deployments/%s/status", siteID, deploymentID)

	resp, err := c.get(url)
	if err != nil {
		return nil, err
	}

	var status DeploymentStatus
	if err := json.Unmarshal(resp.Data, &status); err != nil {
		return nil, fmt.Errorf("failed to parse status: %w", err)
	}

	return &status, nil
}

func (c *Client) ListDeployments(siteID string, limit int) (*DeploymentsListResponse, error) {
	url := fmt.Sprintf("/cli/sites/%s/deployments?limit=%d", siteID, limit)

	resp, err := c.get(url)
	if err != nil {
		return nil, err
	}

	var result DeploymentsListResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse deployments: %w", err)
	}

	return &result, nil
}

func (c *Client) get(path string) (*APIResponse, error) {
	url := c.BaseURL + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	c.applyAuthHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 401 {
		return nil, parseHTTPError(resp.StatusCode, body)
	}

	if resp.StatusCode == 429 {
		return nil, parseHTTPError(resp.StatusCode, body)
	}

	if resp.StatusCode >= 400 {
		return nil, parseHTTPError(resp.StatusCode, body)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &apiResp, nil
}

func (c *Client) postJSON(path string, payload interface{}) (*APIResponse, error) {
	var reqBody io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}

	url := c.BaseURL + path
	req, err := http.NewRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	c.applyAuthHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 401 {
		return nil, parseHTTPError(resp.StatusCode, body)
	}

	if resp.StatusCode == 429 {
		return nil, parseHTTPError(resp.StatusCode, body)
	}

	if resp.StatusCode >= 400 {
		return nil, parseHTTPError(resp.StatusCode, body)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &apiResp, nil
}
