package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

var DefaultBaseURL = "https://api.pachim.sh"

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
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
	ID         string `json:"id"`
	Domain     string `json:"domain"`
	AppType    string `json:"app_type"`
	SetupType  string `json:"setup_type"`
	RepoStatus string `json:"repo_status"`
	GitMerge   bool   `json:"git_merge"`
	Server     struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		IP   string `json:"ip"`
	} `json:"server"`
}

type SiteInfo struct {
	ID        string `json:"id"`
	Domain    string `json:"domain"`
	SetupType string `json:"setup_type"`
	GitMerge  bool   `json:"git_merge"`
}

type DeployResponse struct {
	DeploymentID string `json:"deployment_id"`
	Status       string `json:"status"`
}

type DeploymentStatus struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	Type       string `json:"type"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at"`
	Output     string `json:"output"`
}

func NewClient(baseURL, token string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
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

func (c *Client) Deploy(siteID, zipPath string) (*DeployResponse, error) {
	file, err := os.Open(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file: %w", err)
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("project_zip", "project.zip")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	writer.Close()

	url := fmt.Sprintf("%s/cli/sites/%s/deploy", c.BaseURL, siteID)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")

	httpResp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, _ := io.ReadAll(httpResp.Body)

	if httpResp.StatusCode != 200 {
		return nil, fmt.Errorf("deploy failed (HTTP %d): %s", httpResp.StatusCode, string(respBody))
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var result DeployResponse
	if err := json.Unmarshal(apiResp.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse deploy response: %w", err)
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

func (c *Client) get(path string) (*APIResponse, error) {
	url := c.BaseURL + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("authentication failed. Please run: pachim login")
	}

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limited. Please wait and try again")
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed (HTTP %d): %s", resp.StatusCode, string(body))
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
	req.Header.Set("Accept", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("authentication failed. Please run: pachim login")
	}

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limited. Please wait and try again")
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &apiResp, nil
}
