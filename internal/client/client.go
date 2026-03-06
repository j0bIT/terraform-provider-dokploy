package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DokployClient holds connection details.
type DokployClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func NewDokployClient(baseURL, apiKey string) *DokployClient {
	return &DokployClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *DokployClient) doRequest(method, endpoint string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBytes)
	}

	url := fmt.Sprintf("%s/%s", c.BaseURL, endpoint)

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// fmt.Fprintf(os.Stderr, "DEBUG RESPONSE [%s]: %s\n", endpoint, string(respBytes))

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(respBytes))
	}

	return respBytes, nil
}

// --- Settings / Traefik ---

func (c *DokployClient) ReadTraefikConfig(serverID *string) (string, error) {
	return c.ReadScopedTraefikConfig("main", serverID)
}

func (c *DokployClient) UpdateTraefikConfig(serverID *string, config string) error {
	return c.UpdateScopedTraefikConfig("main", serverID, config)
}

func (c *DokployClient) ReadWebServerTraefikConfig(serverID *string) (string, error) {
	return c.ReadScopedTraefikConfig("web_server", serverID)
}

func (c *DokployClient) UpdateWebServerTraefikConfig(serverID *string, config string) error {
	return c.UpdateScopedTraefikConfig("web_server", serverID, config)
}

func (c *DokployClient) ReadMiddlewareTraefikConfig(serverID *string) (string, error) {
	return c.ReadScopedTraefikConfig("middleware", serverID)
}

func (c *DokployClient) UpdateMiddlewareTraefikConfig(serverID *string, config string) error {
	return c.UpdateScopedTraefikConfig("middleware", serverID, config)
}

func (c *DokployClient) ReadScopedTraefikConfig(scope string, serverID *string) (string, error) {
	normalizedScope, err := normalizeTraefikConfigScope(scope)
	if err != nil {
		return "", err
	}

	endpoint, err := traefikReadEndpointForScope(normalizedScope)
	if err != nil {
		return "", err
	}

	if serverID != nil && strings.TrimSpace(*serverID) != "" {
		endpoint = fmt.Sprintf("%s?serverId=%s", endpoint, url.QueryEscape(strings.TrimSpace(*serverID)))
	}

	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return "", err
	}

	return parseTraefikConfigResponse(resp)
}

func (c *DokployClient) UpdateScopedTraefikConfig(scope string, serverID *string, config string) error {
	normalizedScope, err := normalizeTraefikConfigScope(scope)
	if err != nil {
		return err
	}

	endpoint, payloadKeys, err := traefikUpdateEndpointAndPayloadKeysForScope(normalizedScope)
	if err != nil {
		return err
	}

	basePayload := map[string]interface{}{}
	if serverID != nil && strings.TrimSpace(*serverID) != "" {
		basePayload["serverId"] = strings.TrimSpace(*serverID)
	}

	var lastErr error
	for _, key := range payloadKeys {
		payload := cloneMap(basePayload)
		payload[key] = config
		_, err := c.doRequest("POST", endpoint, payload)
		if err == nil {
			return nil
		}
		lastErr = err
	}

	return lastErr
}

func (c *DokployClient) ReloadTraefik(serverID *string) error {
	payload := map[string]interface{}{}
	if serverID != nil && strings.TrimSpace(*serverID) != "" {
		payload["serverId"] = strings.TrimSpace(*serverID)
	}

	_, err := c.doRequest("POST", "settings.reloadTraefik", payload)
	return err
}

func parseTraefikConfigResponse(resp []byte) (string, error) {
	trimmed := strings.TrimSpace(string(resp))
	if trimmed == "" || trimmed == "null" {
		return "", nil
	}

	var asString string
	if err := json.Unmarshal(resp, &asString); err == nil {
		return asString, nil
	}

	var wrapper struct {
		TraefikConfig           string `json:"traefikConfig"`
		WebServerTraefikConfig  string `json:"webServerTraefikConfig"`
		MiddlewareTraefikConfig string `json:"middlewareTraefikConfig"`
		Config                  string `json:"config"`
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil {
		if wrapper.TraefikConfig != "" {
			return wrapper.TraefikConfig, nil
		}
		if wrapper.WebServerTraefikConfig != "" {
			return wrapper.WebServerTraefikConfig, nil
		}
		if wrapper.MiddlewareTraefikConfig != "" {
			return wrapper.MiddlewareTraefikConfig, nil
		}
		if wrapper.Config != "" {
			return wrapper.Config, nil
		}
	}

	var generic map[string]interface{}
	if err := json.Unmarshal(resp, &generic); err == nil {
		if cfg, found := extractTraefikConfigValue(generic); found {
			return cfg, nil
		}
	}

	return string(resp), nil
}

func extractTraefikConfigValue(input map[string]interface{}) (string, bool) {
	priorityKeys := []string{
		"traefikConfig",
		"webServerTraefikConfig",
		"middlewareTraefikConfig",
		"config",
	}
	for _, key := range priorityKeys {
		if value, ok := input[key]; ok {
			switch typed := value.(type) {
			case string:
				return typed, true
			case map[string]interface{}:
				if cfg, found := extractTraefikConfigValue(typed); found {
					return cfg, true
				}
			default:
				serialized, err := json.Marshal(typed)
				if err == nil {
					return string(serialized), true
				}
			}
		}
	}

	if data, ok := input["data"]; ok {
		if nested, ok := data.(map[string]interface{}); ok {
			if cfg, found := extractTraefikConfigValue(nested); found {
				return cfg, true
			}
		}
	}

	return "", false
}

func normalizeTraefikConfigScope(scope string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(scope))
	switch normalized {
	case "", "main":
		return "main", nil
	case "web_server", "webserver", "web-server":
		return "web_server", nil
	case "middleware":
		return "middleware", nil
	default:
		return "", fmt.Errorf("invalid Traefik config scope %q (supported: main, web_server, middleware)", scope)
	}
}

func traefikReadEndpointForScope(scope string) (string, error) {
	switch scope {
	case "main":
		return "settings.readTraefikConfig", nil
	case "web_server":
		return "settings.readWebServerTraefikConfig", nil
	case "middleware":
		return "settings.readMiddlewareTraefikConfig", nil
	default:
		return "", fmt.Errorf("unsupported Traefik config scope: %s", scope)
	}
}

func traefikUpdateEndpointAndPayloadKeysForScope(scope string) (string, []string, error) {
	switch scope {
	case "main":
		return "settings.updateTraefikConfig", []string{"traefikConfig", "config"}, nil
	case "web_server":
		return "settings.updateWebServerTraefikConfig", []string{"webServerTraefikConfig", "traefikConfig", "config"}, nil
	case "middleware":
		return "settings.updateMiddlewareTraefikConfig", []string{"middlewareTraefikConfig", "traefikConfig", "config"}, nil
	default:
		return "", nil, fmt.Errorf("unsupported Traefik config scope: %s", scope)
	}
}

func cloneMap(input map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

// --- User ---

type User struct {
	ID             string `json:"userId"`
	Email          string `json:"email"`
	OrganizationID string `json:"organizationId"`
}

func (c *DokployClient) GetUser() (*User, error) {
	resp, err := c.doRequest("GET", "user.get", nil)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		User User `json:"user"` // Assuming wrapper based on other endpoints
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil && wrapper.User.ID != "" {
		// If org ID is missing on user, maybe we check roles/orgs?
		// For now assuming simple case.
		return &wrapper.User, nil
	}

	// Try direct
	var user User
	if err := json.Unmarshal(resp, &user); err == nil && user.ID != "" {
		return &user, nil
	}

	return nil, fmt.Errorf("failed to parse user response")
}

// --- Project ---

type Project struct {
	ID           string        `json:"projectId"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Env          string        `json:"env"`
	Environments []Environment `json:"environments"`
}

type projectResponse struct {
	Project Project `json:"project"`
}

func (c *DokployClient) CreateProject(name, description string) (*Project, error) {
	payload := map[string]string{
		"name":        name,
		"description": description,
	}
	resp, err := c.doRequest("POST", "project.create", payload)
	if err != nil {
		return nil, err
	}

	var result projectResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result.Project, nil
}

func (c *DokployClient) GetProject(id string) (*Project, error) {
	endpoint := fmt.Sprintf("project.one?projectId=%s", id)
	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var result Project
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *DokployClient) DeleteProject(id string) error {
	payload := map[string]string{
		"projectId": id,
	}
	_, err := c.doRequest("POST", "project.remove", payload)
	return err
}

func (c *DokployClient) UpdateProject(id, name, description string) (*Project, error) {
	payload := map[string]string{
		"projectId":   id,
		"name":        name,
		"description": description,
	}
	resp, err := c.doRequest("POST", "project.update", payload)
	if err != nil {
		return nil, err
	}

	var result Project
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *DokployClient) UpdateProjectEnv(projectID string, updateFn func(envMap map[string]string)) error {
	var lastErr error

	for i := 0; i < 5; i++ {
		project, err := c.GetProject(projectID)
		if err != nil {
			return err
		}

		envMap := ParseEnv(project.Env)
		originalEnvStr := project.Env

		updateFn(envMap)

		newEnvStr := formatEnv(envMap)
		if newEnvStr == originalEnvStr {
			return nil
		}

		payload := map[string]interface{}{
			"projectId":   projectID,
			"name":        project.Name,
			"description": project.Description,
			"env":         newEnvStr,
		}

		_, err = c.doRequest("POST", "project.update", payload)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
			continue
		}

		verifyProject, err := c.GetProject(projectID)
		if err != nil {
			lastErr = fmt.Errorf("failed to verify environment update: %w", err)
			time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
			continue
		}

		if verifyProject.Env == newEnvStr {
			return nil
		}

		lastErr = fmt.Errorf("environment update conflict, retrying")
	}

	return lastErr
}

// --- Environment ---

type Environment struct {
	ID          string     `json:"environmentId"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	ProjectID   string     `json:"projectId"`
	Postgres    []Database `json:"postgres"`
	Mysql       []Database `json:"mysql"`
	Mariadb     []Database `json:"mariadb"`
	Mongo       []Database `json:"mongo"`
	Redis       []Database `json:"redis"`
}

func (c *DokployClient) CreateEnvironment(projectID, name, description string) (*Environment, error) {
	payload := map[string]string{
		"projectId":   projectID,
		"name":        name,
		"description": description,
	}
	resp, err := c.doRequest("POST", "environment.create", payload)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Environment Environment `json:"environment"`
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil && wrapper.Environment.ID != "" {
		return &wrapper.Environment, nil
	}

	var result Environment
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *DokployClient) UpdateEnvironment(env Environment) (*Environment, error) {
	payload := map[string]interface{}{
		"environmentId": env.ID,
		"name":          env.Name,
		"description":   env.Description,
		"projectId":     env.ProjectID,
	}
	resp, err := c.doRequest("POST", "environment.update", payload)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Environment Environment `json:"environment"`
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil && wrapper.Environment.ID != "" {
		return &wrapper.Environment, nil
	}

	var result Environment
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *DokployClient) DeleteEnvironment(id string) error {
	payload := map[string]string{
		"environmentId": id,
	}
	_, err := c.doRequest("POST", "environment.remove", payload)
	return err
}

// --- Application ---

type Application struct {
	ID                string   `json:"applicationId"`
	Name              string   `json:"name"`
	ProjectID         string   `json:"projectId"`
	EnvironmentID     string   `json:"environmentId"`
	RepositoryURL     string   `json:"repository"`
	Branch            string   `json:"branch"`
	BuildType         string   `json:"buildType"`
	DockerfilePath    string   `json:"dockerfile"`
	DockerContextPath string   `json:"dockerContextPath"`
	DockerBuildStage  string   `json:"dockerBuildStage"`
	Env               string   `json:"env"`
	Domains           []Domain `json:"domains"`
	Ports             []Port   `json:"ports"`
	Mounts            []Mount  `json:"mounts"`
	AutoDeploy        bool     `json:"autoDeploy"`
	// Enhanced fields
	SourceType         string `json:"sourceType"`
	CustomGitUrl       string `json:"customGitUrl"`
	CustomGitBranch    string `json:"customGitBranch"`
	CustomGitSSHKeyId  string `json:"customGitSSHKeyId"`
	CustomGitBuildPath string `json:"customGitBuildPath"`
	Username           string `json:"username"`
	Password           string `json:"password"`
	// GitHub Provider fields
	GithubRepository string            `json:"githubRepository"`
	GithubOwner      string            `json:"owner"`
	GithubBranch     string            `json:"githubBranch"`
	GithubBuildPath  string            `json:"buildPath"`
	GithubID         string            `json:"githubId"`
	GithubWatchPaths []string          `json:"watchPaths"`
	EnableSubmodules bool              `json:"enableSubmodules"`
	TriggerType      string            `json:"triggerType"`
	LabelsSwarm      map[string]string `json:"labelsSwarm"`
	// Preview deployment fields
	IsPreviewDeploymentsActive            *bool    `json:"isPreviewDeploymentsActive"`
	PreviewWildcard                       string   `json:"previewWildcard"`
	PreviewPort                           *int64   `json:"previewPort"`
	PreviewPath                           string   `json:"previewPath"`
	PreviewHTTPS                          *bool    `json:"previewHttps"`
	PreviewCertificateType                string   `json:"previewCertificateType"`
	PreviewCustomCertResolver             string   `json:"previewCustomCertResolver"`
	PreviewLimit                          *int64   `json:"previewLimit"`
	PreviewRequireCollaboratorPermissions *bool    `json:"previewRequireCollaboratorPermissions"`
	PreviewEnv                            string   `json:"previewEnv"`
	PreviewBuildArgs                      string   `json:"previewBuildArgs"`
	PreviewLabels                         []string `json:"previewLabels"`
}

func (c *DokployClient) CreateApplication(app Application) (*Application, error) {
	// 1. Create minimal application
	createPayload := map[string]string{
		"name":          app.Name,
		"environmentId": app.EnvironmentID,
	}

	resp, err := c.doRequest("POST", "application.create", createPayload)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Application Application `json:"application"`
	}
	if err := json.Unmarshal(resp, &wrapper); err != nil {
		return nil, err
	}

	createdApp := wrapper.Application
	if createdApp.ID == "" {
		if err := json.Unmarshal(resp, &createdApp); err != nil {
			return nil, err
		}
	}

	// 2. Update with full configuration
	updatePayload := map[string]interface{}{
		"applicationId": createdApp.ID,
		"name":          app.Name,
		"branch":        app.Branch,
		"buildType":     app.BuildType,
		"sourceType":    app.SourceType,
		"autoDeploy":    app.AutoDeploy,
	}

	if app.RepositoryURL != "" {
		updatePayload["repository"] = app.RepositoryURL
	}
	if app.DockerfilePath != "" {
		updatePayload["dockerfile"] = app.DockerfilePath
	}
	if app.DockerContextPath != "" {
		updatePayload["dockerContextPath"] = app.DockerContextPath
	}
	if app.DockerBuildStage != "" {
		updatePayload["dockerBuildStage"] = app.DockerBuildStage
	}
	if app.CustomGitUrl != "" {
		updatePayload["customGitUrl"] = app.CustomGitUrl
	}
	if app.CustomGitBranch != "" {
		updatePayload["customGitBranch"] = app.CustomGitBranch
	}
	if app.CustomGitSSHKeyId != "" {
		updatePayload["customGitSSHKeyId"] = app.CustomGitSSHKeyId
	}
	if app.CustomGitBuildPath != "" {
		updatePayload["customGitBuildPath"] = app.CustomGitBuildPath
	}
	if app.Username != "" {
		updatePayload["username"] = app.Username
	}
	if app.Password != "" {
		updatePayload["password"] = app.Password
	}
	if app.LabelsSwarm != nil {
		updatePayload["labelsSwarm"] = app.LabelsSwarm
	}
	addPreviewApplicationPayload(updatePayload, app)

	// Ensure defaults
	if app.SourceType == "" {
		if app.CustomGitUrl != "" {
			updatePayload["sourceType"] = "git"
		} else {
			updatePayload["sourceType"] = "github"
		}
	}

	respUpdate, err := c.doRequest("POST", "application.update", updatePayload)
	if err != nil {
		return nil, fmt.Errorf("created application %s but failed to update config: %w", createdApp.ID, err)
	}

	if string(respUpdate) == "true" {
		return c.GetApplication(createdApp.ID)
	}

	var updateResult Application
	if err := json.Unmarshal(respUpdate, &wrapper); err == nil && wrapper.Application.ID != "" {
		return &wrapper.Application, nil
	}
	if err := json.Unmarshal(respUpdate, &updateResult); err == nil {
		return &updateResult, nil
	}

	return &createdApp, nil
}

func (c *DokployClient) GetApplication(id string) (*Application, error) {
	endpoint := fmt.Sprintf("application.one?applicationId=%s", id)
	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var result Application
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *DokployClient) UpdateApplication(app Application) (*Application, error) {
	payload := map[string]interface{}{
		"applicationId": app.ID,
		"name":          app.Name,
		"branch":        app.Branch,
		"buildType":     app.BuildType,
		"sourceType":    app.SourceType,
		"autoDeploy":    app.AutoDeploy,
	}
	// Optional fields
	if app.RepositoryURL != "" {
		payload["repository"] = app.RepositoryURL
	}
	if app.DockerfilePath != "" {
		payload["dockerfile"] = app.DockerfilePath
	}
	if app.DockerContextPath != "" {
		payload["dockerContextPath"] = app.DockerContextPath
	}
	if app.DockerBuildStage != "" {
		payload["dockerBuildStage"] = app.DockerBuildStage
	}
	if app.CustomGitUrl != "" {
		payload["customGitUrl"] = app.CustomGitUrl
	}
	if app.CustomGitBranch != "" {
		payload["customGitBranch"] = app.CustomGitBranch
	}
	if app.CustomGitSSHKeyId != "" {
		payload["customGitSSHKeyId"] = app.CustomGitSSHKeyId
	}
	if app.CustomGitBuildPath != "" {
		payload["customGitBuildPath"] = app.CustomGitBuildPath
	}
	if app.Username != "" {
		payload["username"] = app.Username
	}
	if app.Password != "" {
		payload["password"] = app.Password
	}
	if app.LabelsSwarm != nil {
		payload["labelsSwarm"] = app.LabelsSwarm
	}

	if app.EnvironmentID != "" {
		payload["environmentId"] = app.EnvironmentID
	}
	addPreviewApplicationPayload(payload, app)

	resp, err := c.doRequest("POST", "application.update", payload)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(string(resp)) == "true" {
		return c.GetApplication(app.ID)
	}

	var wrapper struct {
		Application Application `json:"application"`
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil && wrapper.Application.ID != "" {
		return &wrapper.Application, nil
	}

	var result Application
	if err := json.Unmarshal(resp, &result); err == nil {
		return &result, nil
	}

	return nil, fmt.Errorf("failed to parse application.update response for application %s: %s", app.ID, string(resp))
}

func addPreviewApplicationPayload(payload map[string]interface{}, app Application) {
	if app.IsPreviewDeploymentsActive != nil {
		payload["isPreviewDeploymentsActive"] = *app.IsPreviewDeploymentsActive
	}
	if app.PreviewWildcard != "" {
		payload["previewWildcard"] = app.PreviewWildcard
	}
	if app.PreviewPort != nil {
		payload["previewPort"] = *app.PreviewPort
	}
	if app.PreviewPath != "" {
		payload["previewPath"] = app.PreviewPath
	}
	if app.PreviewHTTPS != nil {
		payload["previewHttps"] = *app.PreviewHTTPS
	}
	if app.PreviewCertificateType != "" {
		payload["previewCertificateType"] = app.PreviewCertificateType
	}
	if app.PreviewCustomCertResolver != "" {
		payload["previewCustomCertResolver"] = app.PreviewCustomCertResolver
	}
	if app.PreviewLimit != nil {
		payload["previewLimit"] = *app.PreviewLimit
	}
	if app.PreviewRequireCollaboratorPermissions != nil {
		payload["previewRequireCollaboratorPermissions"] = *app.PreviewRequireCollaboratorPermissions
	}
	if app.PreviewEnv != "" {
		payload["previewEnv"] = app.PreviewEnv
	}
	if app.PreviewBuildArgs != "" {
		payload["previewBuildArgs"] = app.PreviewBuildArgs
	}
	if len(app.PreviewLabels) > 0 {
		payload["previewLabels"] = app.PreviewLabels
	}
}

func (c *DokployClient) DeleteApplication(id string) error {
	// Best-effort stop before deletion to make teardown explicit and predictable.
	// Ignore stop errors; delete call should still reconcile the final state.
	_ = c.StopApplication(id)

	payload := map[string]string{
		"applicationId": id,
	}
	_, err := c.doRequest("POST", "application.delete", payload)
	if err == nil {
		return nil
	}

	// Backward compatibility with older Dokploy versions that still expose
	// application.remove instead of application.delete.
	_, removeErr := c.doRequest("POST", "application.remove", payload)
	if removeErr != nil {
		return fmt.Errorf("application.delete failed: %w; application.remove fallback failed: %w", err, removeErr)
	}

	return nil
}

func (c *DokployClient) SaveGithubProvider(appID string, githubConfig map[string]interface{}) error {
	payload := map[string]interface{}{
		"applicationId": appID,
	}

	// Add all github provider fields
	for key, value := range githubConfig {
		payload[key] = value
	}

	_, err := c.doRequest("POST", "application.saveGithubProvider", payload)
	return err
}

func (c *DokployClient) DeployApplication(id string) error {
	payload := map[string]string{
		"applicationId": id,
	}
	_, err := c.doRequest("POST", "application.deploy", payload)
	return err
}

func (c *DokployClient) StopApplication(id string) error {
	payload := map[string]string{
		"applicationId": id,
	}
	_, err := c.doRequest("POST", "application.stop", payload)
	return err
}

// --- Mount ---

type Mount struct {
	ID            string `json:"mountId"`
	ApplicationID string `json:"applicationId"`
	MountType     string `json:"mountType"`
	Type          string `json:"type"`
	MountPath     string `json:"mountPath"`
	VolumeName    string `json:"volumeName"`
	HostPath      string `json:"hostPath"`
	ServiceType   string `json:"serviceType"`
	ServiceID     string `json:"serviceId"`
}

func (c *DokployClient) CreateMount(mount Mount) (*Mount, error) {
	mountType := strings.TrimSpace(mount.MountType)
	if mountType == "" {
		mountType = strings.TrimSpace(mount.Type)
	}
	if mountType == "" {
		mountType = "volume"
	}

	payload := map[string]interface{}{
		"type":        mountType,
		"serviceType": "application",
		"serviceId":   mount.ApplicationID,
		"mountPath":   mount.MountPath,
	}
	if strings.TrimSpace(mount.VolumeName) != "" {
		payload["volumeName"] = mount.VolumeName
	}
	if strings.TrimSpace(mount.HostPath) != "" {
		payload["hostPath"] = mount.HostPath
	}

	resp, err := c.doRequest("POST", "mounts.create", payload)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Mount Mount `json:"mount"`
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil && strings.TrimSpace(wrapper.Mount.ID) != "" {
		return &wrapper.Mount, nil
	}

	var result Mount
	if err := json.Unmarshal(resp, &result); err == nil && strings.TrimSpace(result.ID) != "" {
		return &result, nil
	}

	if strings.TrimSpace(string(resp)) == "true" {
		created, err := c.findMountBySignature(mount.ApplicationID, mountType, mount.MountPath, mount.VolumeName)
		if err == nil {
			return created, nil
		}

		// Some Dokploy versions return true from mounts.create while mount listings
		// are not yet queryable. Return requested shape to avoid false negatives.
		return &Mount{
			ApplicationID: mount.ApplicationID,
			MountType:     mountType,
			Type:          mountType,
			MountPath:     mount.MountPath,
			VolumeName:    mount.VolumeName,
			HostPath:      mount.HostPath,
			ServiceType:   "application",
			ServiceID:     mount.ApplicationID,
		}, nil
	}

	created, err := c.findMountBySignature(mount.ApplicationID, mountType, mount.MountPath, mount.VolumeName)
	if err == nil {
		return created, nil
	}

	// Some Dokploy versions acknowledge writes before mount listing is consistent.
	// Return the requested mount shape so caller can continue without hard-failing.
	return &Mount{
		ApplicationID: mount.ApplicationID,
		MountType:     mountType,
		Type:          mountType,
		MountPath:     mount.MountPath,
		VolumeName:    mount.VolumeName,
		HostPath:      mount.HostPath,
		ServiceType:   "application",
		ServiceID:     mount.ApplicationID,
	}, nil
}

func (c *DokployClient) findMountBySignature(applicationID, mountType, mountPath, volumeName string) (*Mount, error) {
	var lastErr error
	for i := 0; i < 8; i++ {
		mounts, err := c.ListMountsByApplication(applicationID)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
			continue
		}

		for _, existing := range mounts {
			existingType := strings.TrimSpace(existing.MountType)
			if existingType == "" {
				existingType = strings.TrimSpace(existing.Type)
			}
			mountTypeMatches := strings.TrimSpace(mountType) == "" || strings.EqualFold(existingType, mountType)

			existingPath := strings.TrimSuffix(strings.TrimSpace(existing.MountPath), "/")
			requestedPath := strings.TrimSuffix(strings.TrimSpace(mountPath), "/")
			pathMatches := existingPath == requestedPath

			existingVolume := strings.TrimSpace(existing.VolumeName)
			requestedVolume := strings.TrimSpace(volumeName)
			volumeNameMatches := requestedVolume == "" ||
				existingVolume == requestedVolume ||
				strings.HasSuffix(existingVolume, "_"+requestedVolume) ||
				strings.HasPrefix(existingVolume, requestedVolume+"_")

			if mountTypeMatches && pathMatches && volumeNameMatches {
				return &existing, nil
			}
		}

		time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
	}

	if lastErr != nil {
		return nil, fmt.Errorf("mount created but mount lookup failed for application %s: %w", applicationID, lastErr)
	}

	return nil, fmt.Errorf(
		"mount created but not found on application %s (mountType=%q, mountPath=%q, volumeName=%q)",
		applicationID,
		mountType,
		mountPath,
		volumeName,
	)
}

func (c *DokployClient) ListMountsByApplication(applicationID string) ([]Mount, error) {
	endpoint := fmt.Sprintf("mounts.allNamedByApplicationId?applicationId=%s", applicationID)
	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	if trimmed := strings.TrimSpace(string(resp)); trimmed == "" || trimmed == "null" || trimmed == "true" || trimmed == "false" {
		return []Mount{}, nil
	}

	var wrapper struct {
		Mounts []Mount `json:"mounts"`
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil && wrapper.Mounts != nil {
		return wrapper.Mounts, nil
	}

	var dataWrapper struct {
		Data []Mount `json:"data"`
	}
	if err := json.Unmarshal(resp, &dataWrapper); err == nil && dataWrapper.Data != nil {
		return dataWrapper.Data, nil
	}

	var direct []Mount
	if err := json.Unmarshal(resp, &direct); err == nil {
		return direct, nil
	}

	var named map[string]Mount
	if err := json.Unmarshal(resp, &named); err == nil && named != nil {
		mounts := make([]Mount, 0, len(named))
		for _, mount := range named {
			mounts = append(mounts, mount)
		}
		return mounts, nil
	}

	return nil, fmt.Errorf("failed to parse mounts.allNamedByApplicationId response: %s", string(resp))
}

func (c *DokployClient) DeleteMount(id string) error {
	payload := map[string]string{
		"mountId": id,
	}
	_, err := c.doRequest("POST", "mounts.remove", payload)
	if err == nil {
		return nil
	}

	_, fallbackErr := c.doRequest("POST", "mount.delete", payload)
	if fallbackErr == nil {
		return nil
	}

	_, fallbackDeleteErr := c.doRequest("POST", "mounts.delete", payload)
	if fallbackDeleteErr == nil {
		return nil
	}

	return fmt.Errorf(
		"mounts.remove failed: %w; mount.delete fallback failed: %v; mounts.delete fallback failed: %v",
		err,
		fallbackErr,
		fallbackDeleteErr,
	)
}

// --- Compose ---

type Compose struct {
	ID                string   `json:"composeId"`
	Name              string   `json:"name"`
	AppName           string   `json:"appName"`
	ProjectID         string   `json:"projectId"`
	EnvironmentID     string   `json:"environmentId"`
	ComposeFile       string   `json:"composeFile"`
	SourceType        string   `json:"sourceType"`
	CustomGitUrl      string   `json:"customGitUrl"`
	CustomGitBranch   string   `json:"customGitBranch"`
	CustomGitSSHKeyId string   `json:"customGitSSHKeyId"`
	ComposePath       string   `json:"composePath"`
	AutoDeploy        bool     `json:"autoDeploy"`
	Env               string   `json:"env"`
	Domains           []Domain `json:"domains"`
}

func (c *DokployClient) CreateCompose(comp Compose) (*Compose, error) {
	// 1. Create minimal compose
	payload := map[string]string{
		"environmentId": comp.EnvironmentID,
		"name":          comp.Name,
		"composeType":   "docker-compose",
		"appName":       comp.Name,
	}

	// If raw content provided, include it
	if comp.ComposeFile != "" {
		payload["composeFile"] = comp.ComposeFile
	}

	resp, err := c.doRequest("POST", "compose.create", payload)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Compose Compose `json:"compose"`
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil && wrapper.Compose.ID != "" {
		return &wrapper.Compose, nil
	}

	createdComp := wrapper.Compose
	if createdComp.ID == "" {
		if err := json.Unmarshal(resp, &createdComp); err != nil {
			return nil, err
		}
	}

	// 2. Update with Git configuration if necessary
	updatePayload := map[string]interface{}{
		"composeId":  createdComp.ID,
		"name":       comp.Name,
		"sourceType": comp.SourceType,
		"autoDeploy": comp.AutoDeploy,
	}

	if comp.CustomGitUrl != "" {
		updatePayload["customGitUrl"] = comp.CustomGitUrl
	}
	if comp.CustomGitBranch != "" {
		updatePayload["customGitBranch"] = comp.CustomGitBranch
	}
	if comp.CustomGitSSHKeyId != "" {
		updatePayload["customGitSSHKeyId"] = comp.CustomGitSSHKeyId
	}
	if comp.ComposePath != "" {
		updatePayload["composePath"] = comp.ComposePath
	}
	if comp.ComposeFile != "" {
		updatePayload["composeFile"] = comp.ComposeFile
	}

	if comp.SourceType == "" {
		if comp.CustomGitUrl != "" {
			updatePayload["sourceType"] = "git"
		} else if comp.ComposeFile != "" {
			updatePayload["sourceType"] = "raw"
		} else {
			updatePayload["sourceType"] = "github"
		}
	}

	respUpdate, err := c.doRequest("POST", "compose.update", updatePayload)
	if err != nil {
		return nil, fmt.Errorf("created compose %s but failed to update config: %w", createdComp.ID, err)
	}

	if string(respUpdate) == "true" {
		return c.GetCompose(createdComp.ID)
	}

	var updateResult Compose
	if err := json.Unmarshal(respUpdate, &wrapper); err == nil && wrapper.Compose.ID != "" {
		return &wrapper.Compose, nil
	}
	if err := json.Unmarshal(respUpdate, &updateResult); err == nil {
		return &updateResult, nil
	}

	return &createdComp, nil
}

func (c *DokployClient) GetCompose(id string) (*Compose, error) {
	endpoint := fmt.Sprintf("compose.one?composeId=%s", id)
	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	var result Compose
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *DokployClient) UpdateCompose(comp Compose) (*Compose, error) {
	payload := map[string]interface{}{
		"composeId":  comp.ID,
		"name":       comp.Name,
		"sourceType": comp.SourceType,
		"autoDeploy": comp.AutoDeploy,
	}

	if comp.CustomGitUrl != "" {
		payload["customGitUrl"] = comp.CustomGitUrl
	}
	if comp.CustomGitBranch != "" {
		payload["customGitBranch"] = comp.CustomGitBranch
	}
	if comp.CustomGitSSHKeyId != "" {
		payload["customGitSSHKeyId"] = comp.CustomGitSSHKeyId
	}
	if comp.ComposePath != "" {
		payload["composePath"] = comp.ComposePath
	}
	if comp.ComposeFile != "" {
		payload["composeFile"] = comp.ComposeFile
	}

	if comp.EnvironmentID != "" {
		payload["environmentId"] = comp.EnvironmentID
	}

	resp, err := c.doRequest("POST", "compose.update", payload)
	if err != nil {
		return nil, err
	}

	var result Compose
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *DokployClient) DeleteCompose(id string, deleteVolumes bool) error {
	// Best-effort stop before deletion to make teardown explicit and predictable.
	// Ignore stop errors; delete call should still reconcile the final state.
	_ = c.StopCompose(id)

	deletePayload := map[string]interface{}{
		"composeId":     id,
		"deleteVolumes": deleteVolumes,
	}
	_, err := c.doRequest("POST", "compose.delete", deletePayload)
	if err == nil {
		return nil
	}

	// Backward compatibility with older Dokploy versions that still expose
	// compose.remove instead of compose.delete.
	payload := map[string]string{
		"composeId": id,
	}
	_, removeErr := c.doRequest("POST", "compose.remove", payload)
	if removeErr != nil {
		return fmt.Errorf("compose.delete failed: %w; compose.remove fallback failed: %w", err, removeErr)
	}

	return nil
}

func (c *DokployClient) DeployCompose(id string) error {
	payload := map[string]string{
		"composeId": id,
	}
	_, err := c.doRequest("POST", "compose.deploy", payload)
	return err
}

func (c *DokployClient) StopCompose(id string) error {
	payload := map[string]string{
		"composeId": id,
	}
	_, err := c.doRequest("POST", "compose.stop", payload)
	return err
}

// --- Database ---

type Database struct {
	ID            string `json:"databaseId"`
	Name          string `json:"name"`
	AppName       string `json:"appName"`
	Type          string `json:"type"`
	ProjectID     string `json:"projectId"`
	EnvironmentID string `json:"environmentId"`
	Version       string `json:"version"`
	DockerImage   string `json:"dockerImage"`
	ExternalPort  int64  `json:"externalPort"`
	InternalPort  int64  `json:"internalPort"`
	Password      string `json:"password"`
	PostgresID    string `json:"postgresId"`
	MysqlID       string `json:"mysqlId"`
	MariadbID     string `json:"mariadbId"`
	MongoID       string `json:"mongoId"`
	RedisID       string `json:"redisId"`
}

func databaseTypeSpecificID(db Database, databaseType string) string {
	switch databaseType {
	case "postgres":
		return db.PostgresID
	case "mysql":
		return db.MysqlID
	case "mariadb":
		return db.MariadbID
	case "mongo":
		return db.MongoID
	case "redis":
		return db.RedisID
	default:
		return ""
	}
}

func databaseAnyTypeID(db Database) string {
	if db.PostgresID != "" {
		return db.PostgresID
	}
	if db.MysqlID != "" {
		return db.MysqlID
	}
	if db.MariadbID != "" {
		return db.MariadbID
	}
	if db.MongoID != "" {
		return db.MongoID
	}
	if db.RedisID != "" {
		return db.RedisID
	}
	return ""
}

func normalizeDatabaseID(db *Database, databaseType string) {
	if db == nil {
		return
	}
	if db.ID != "" {
		return
	}
	if id := databaseTypeSpecificID(*db, databaseType); id != "" {
		db.ID = id
		return
	}
	db.ID = databaseAnyTypeID(*db)
}

func (c *DokployClient) CreateDatabase(projectID, environmentID, name, dbType, password, dockerImage string) (*Database, error) {
	var endpoint string
	payload := map[string]string{
		"environmentId":    environmentID,
		"name":             name,
		"appName":          name,
		"databaseName":     name,
		"databasePassword": password,
		"dockerImage":      dockerImage,
	}

	switch dbType {
	case "postgres":
		endpoint = "postgres.create"
		payload["databaseUser"] = "postgres"
	case "mysql":
		endpoint = "mysql.create"
		payload["databaseUser"] = "root"
	case "mariadb":
		endpoint = "mariadb.create"
		payload["databaseUser"] = "root"
	case "mongo":
		endpoint = "mongo.create"
		payload["databaseUser"] = "mongo"
	case "redis":
		endpoint = "redis.create"
		payload["databaseUser"] = "default"
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	resp, err := c.doRequest("POST", endpoint, payload)
	if err != nil {
		return nil, err
	}

	// Try to parse the response as a database object first.
	var directResult Database
	if err := json.Unmarshal(resp, &directResult); err == nil {
		normalizeDatabaseID(&directResult, dbType)
		if directResult.ID != "" {
			if directResult.Type == "" {
				directResult.Type = dbType
			}
			return &directResult, nil
		}
	}

	// Check if response is just "true" (boolean success indicator)
	if string(resp) == "true" {
		project, err := c.GetProject(projectID)
		if err != nil {
			return nil, fmt.Errorf("database created but failed to fetch project: %w", err)
		}

		for _, env := range project.Environments {
			if env.ID == environmentID {
				var dbs []Database
				switch dbType {
				case "postgres":
					dbs = env.Postgres
				case "mysql":
					dbs = env.Mysql
				case "mariadb":
					dbs = env.Mariadb
				case "mongo":
					dbs = env.Mongo
				case "redis":
					dbs = env.Redis
				}

				for _, db := range dbs {
					if db.Name == name || db.AppName == name {
						id := databaseTypeSpecificID(db, dbType)
						if id == "" {
							id = databaseAnyTypeID(db)
						}

						// If no type-specific ID, try the generic databaseId field
						if id == "" && db.ID != "" {
							id = db.ID
						}

						// Create a result database with the ID explicitly set
						result := Database{
							ID:            id,
							Name:          db.Name,
							AppName:       db.AppName,
							Type:          dbType,
							ProjectID:     db.ProjectID,
							EnvironmentID: db.EnvironmentID,
							Version:       db.Version,
							DockerImage:   db.DockerImage,
							ExternalPort:  db.ExternalPort,
							InternalPort:  db.InternalPort,
							Password:      db.Password,
							PostgresID:    db.PostgresID,
							MysqlID:       db.MysqlID,
							MariadbID:     db.MariadbID,
							MongoID:       db.MongoID,
							RedisID:       db.RedisID,
						}
						normalizeDatabaseID(&result, dbType)

						if result.ID == "" {
							return nil, fmt.Errorf("database created but ID not found (name: %s, postgresId: %s, databaseId: %s)", name, db.PostgresID, db.ID)
						}

						return &result, nil
					}
				}
			}
		}
		return nil, fmt.Errorf("database created but not found in project environments")
	}

	var wrapper struct {
		Database Database `json:"database"`
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil {
		normalizeDatabaseID(&wrapper.Database, dbType)
		if wrapper.Database.ID != "" {
			if wrapper.Database.Type == "" {
				wrapper.Database.Type = dbType
			}
			return &wrapper.Database, nil
		}
	}

	var result Database
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	normalizeDatabaseID(&result, dbType)
	if result.Type == "" {
		result.Type = dbType
	}
	if result.ID == "" {
		return nil, fmt.Errorf("database created but ID not found in response for type %s", dbType)
	}
	return &result, nil
}

func (c *DokployClient) GetDatabase(dbID string, databaseType string) (*Database, error) {
	var endpoint string
	switch databaseType {
	case "postgres":
		endpoint = fmt.Sprintf("postgres.one?postgresId=%s", dbID)
	case "mysql":
		endpoint = fmt.Sprintf("mysql.one?mysqlId=%s", dbID)
	case "mariadb":
		endpoint = fmt.Sprintf("mariadb.one?mariadbId=%s", dbID)
	case "mongo":
		endpoint = fmt.Sprintf("mongo.one?mongoId=%s", dbID)
	case "redis":
		endpoint = fmt.Sprintf("redis.one?redisId=%s", dbID)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", databaseType)
	}

	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var db Database
	if err := json.Unmarshal(resp, &db); err == nil {
		normalizeDatabaseID(&db, databaseType)
		if db.ID != "" {
			db.Type = databaseType
			return &db, nil
		}
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	var dbBytes json.RawMessage
	var ok bool

	switch databaseType {
	case "postgres":
		dbBytes, ok = result["postgres"]
	case "mysql":
		dbBytes, ok = result["mysql"]
	case "mariadb":
		dbBytes, ok = result["mariadb"]
	case "mongo":
		dbBytes, ok = result["mongo"]
	case "redis":
		dbBytes, ok = result["redis"]
	}

	if !ok {
		if val, found := result["database"]; found {
			dbBytes = val
		} else {
			return nil, fmt.Errorf("database key not found in response for type %s", databaseType)
		}
	}

	if err := json.Unmarshal(dbBytes, &db); err != nil {
		return nil, err
	}
	normalizeDatabaseID(&db, databaseType)
	db.Type = databaseType

	return &db, nil
}

func (c *DokployClient) DeleteDatabase(id string) error {
	return fmt.Errorf("delete database requires type update")
}

func (c *DokployClient) DeleteDatabaseWithType(id, dbType string) error {
	var endpoint string
	var idKey string
	switch dbType {
	case "postgres":
		endpoint = "postgres.remove"
		idKey = "postgresId"
	case "mysql":
		endpoint = "mysql.remove"
		idKey = "mysqlId"
	case "mariadb":
		endpoint = "mariadb.remove"
		idKey = "mariadbId"
	case "mongo":
		endpoint = "mongo.remove"
		idKey = "mongoId"
	case "redis":
		endpoint = "redis.remove"
		idKey = "redisId"
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}

	payload := map[string]string{
		idKey: id,
	}
	_, err := c.doRequest("POST", endpoint, payload)
	return err
}

// --- Domain ---

type Domain struct {
	ID              string `json:"domainId"`
	ApplicationID   string `json:"applicationId"`
	ComposeID       string `json:"composeId"`
	ServiceName     string `json:"serviceName"`
	Host            string `json:"host"`
	Path            string `json:"path"`
	Port            int64  `json:"port"`
	HTTPS           bool   `json:"https"`
	CertificateType string `json:"certificateType"`
}

func (c *DokployClient) CreateDomain(domain Domain) (*Domain, error) {
	payload := map[string]interface{}{
		"host":            domain.Host,
		"path":            domain.Path,
		"port":            domain.Port,
		"https":           domain.HTTPS,
		"certificateType": domain.CertificateType,
	}
	if domain.ApplicationID != "" {
		payload["applicationId"] = domain.ApplicationID
	}
	if domain.ComposeID != "" {
		payload["composeId"] = domain.ComposeID
	}
	if domain.ServiceName != "" {
		payload["serviceName"] = domain.ServiceName
	}

	resp, err := c.doRequest("POST", "domain.create", payload)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Domain Domain `json:"domain"`
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil && wrapper.Domain.ID != "" {
		return &wrapper.Domain, nil
	}

	var result Domain
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *DokployClient) GetDomainsByApplication(appID string) ([]Domain, error) {
	app, err := c.GetApplication(appID)
	if err != nil {
		return nil, err
	}
	return app.Domains, nil
}

func (c *DokployClient) GetDomainsByCompose(composeID string) ([]Domain, error) {
	comp, err := c.GetCompose(composeID)
	if err != nil {
		return nil, err
	}
	return comp.Domains, nil
}

func (c *DokployClient) DeleteDomain(id string) error {
	payload := map[string]string{
		"domainId": id,
	}
	_, err := c.doRequest("POST", "domain.remove", payload)
	return err
}

func (c *DokployClient) GenerateDomain(appName string) (string, error) {
	payload := map[string]string{
		"appName": appName,
	}
	resp, err := c.doRequest("POST", "domain.generateDomain", payload)
	if err != nil {
		return "", err
	}

	// Try to parse as JSON wrapper
	var wrapper struct {
		Domain string `json:"domain"`
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil && wrapper.Domain != "" {
		return wrapper.Domain, nil
	}

	// Fallback: maybe it returns just the string in quotes or raw?
	// If it is a simple string "foo.bar", Unmarshal might fail or we just return string(resp) trimmed.
	return strings.Trim(string(resp), "\""), nil
}

func (c *DokployClient) UpdateDomain(domain Domain) (*Domain, error) {
	payload := map[string]interface{}{
		"domainId":        domain.ID,
		"host":            domain.Host,
		"path":            domain.Path,
		"port":            domain.Port,
		"https":           domain.HTTPS,
		"certificateType": domain.CertificateType,
		"serviceName":     domain.ServiceName,
	}
	resp, err := c.doRequest("POST", "domain.update", payload)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Domain Domain `json:"domain"`
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil && wrapper.Domain.ID != "" {
		return &wrapper.Domain, nil
	}

	var result Domain
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// --- Port ---

type Port struct {
	ID            string `json:"portId"`
	ApplicationID string `json:"applicationId"`
	PublishedPort int64  `json:"publishedPort"`
	TargetPort    int64  `json:"targetPort"`
	Protocol      string `json:"protocol"`
	PublishMode   string `json:"publishMode"`
}

func (c *DokployClient) CreatePort(port Port) (*Port, error) {
	payload := map[string]interface{}{
		"applicationId": port.ApplicationID,
		"publishedPort": port.PublishedPort,
		"targetPort":    port.TargetPort,
	}
	if strings.TrimSpace(port.Protocol) != "" {
		payload["protocol"] = port.Protocol
	}
	if strings.TrimSpace(port.PublishMode) != "" {
		payload["publishMode"] = port.PublishMode
	}

	resp, err := c.doRequest("POST", "port.create", payload)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Port Port `json:"port"`
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil && wrapper.Port.ID != "" {
		return &wrapper.Port, nil
	}

	var result Port
	if err := json.Unmarshal(resp, &result); err == nil && result.ID != "" {
		return &result, nil
	}

	// Dokploy sometimes returns a bare boolean on successful writes.
	// In that case resolve the newly created port by matching its signature.
	if string(resp) == "true" {
		return c.findPortBySignature(port.ApplicationID, port.PublishedPort, port.TargetPort, port.Protocol, port.PublishMode)
	}

	return c.findPortBySignature(port.ApplicationID, port.PublishedPort, port.TargetPort, port.Protocol, port.PublishMode)
}

func (c *DokployClient) findPortBySignature(applicationID string, publishedPort, targetPort int64, protocol, publishMode string) (*Port, error) {
	app, err := c.GetApplication(applicationID)
	if err != nil {
		return nil, fmt.Errorf("port created but failed to fetch application: %w", err)
	}

	for _, existing := range app.Ports {
		protocolMatches := strings.TrimSpace(protocol) == "" || strings.EqualFold(existing.Protocol, protocol)
		publishModeMatches := strings.TrimSpace(publishMode) == "" || strings.EqualFold(existing.PublishMode, publishMode)

		if existing.PublishedPort == publishedPort &&
			existing.TargetPort == targetPort &&
			protocolMatches &&
			publishModeMatches {
			return &existing, nil
		}
	}

	return nil, fmt.Errorf(
		"port created but not found on application %s (publishedPort=%d, targetPort=%d, protocol=%q, publishMode=%q)",
		applicationID,
		publishedPort,
		targetPort,
		protocol,
		publishMode,
	)
}

func (c *DokployClient) GetPort(id string) (*Port, error) {
	endpoint := fmt.Sprintf("port.one?portId=%s", id)
	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Port Port `json:"port"`
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil && wrapper.Port.ID != "" {
		return &wrapper.Port, nil
	}

	var result Port
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	if result.ID == "" {
		return nil, fmt.Errorf("failed to parse port.one response: missing portId")
	}
	return &result, nil
}

func (c *DokployClient) UpdatePort(port Port) (*Port, error) {
	payload := map[string]interface{}{
		"portId":        port.ID,
		"publishedPort": port.PublishedPort,
		"targetPort":    port.TargetPort,
	}
	if strings.TrimSpace(port.Protocol) != "" {
		payload["protocol"] = port.Protocol
	}
	if strings.TrimSpace(port.PublishMode) != "" {
		payload["publishMode"] = port.PublishMode
	}

	resp, err := c.doRequest("POST", "port.update", payload)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Port Port `json:"port"`
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil && wrapper.Port.ID != "" {
		return &wrapper.Port, nil
	}

	var result Port
	if err := json.Unmarshal(resp, &result); err == nil && result.ID != "" {
		return &result, nil
	}

	if string(resp) == "true" {
		return c.GetPort(port.ID)
	}

	return c.GetPort(port.ID)
}

func (c *DokployClient) DeletePort(id string) error {
	payload := map[string]string{
		"portId": id,
	}
	_, err := c.doRequest("POST", "port.delete", payload)
	if err != nil {
		return err
	}
	return nil
}

// --- Environment Variable ---

type EnvironmentVariable struct {
	ID            string `json:"id"`
	ApplicationID string `json:"applicationId"`
	Key           string `json:"key"`
	Value         string `json:"value"`
	Scope         string `json:"scope"`
}

func (c *DokployClient) UpdateApplicationEnv(appID string, updateFn func(envMap map[string]string), createEnvFile *bool) error {
	var lastErr error
	for i := 0; i < 5; i++ { // Retry up to 5 times
		app, err := c.GetApplication(appID)
		if err != nil {
			return err
		}

		envMap := ParseEnv(app.Env)
		originalEnvStr := app.Env

		updateFn(envMap) // Modify the map

		newEnvStr := formatEnv(envMap)

		if newEnvStr == originalEnvStr {
			return nil // No changes to be made
		}

		payload := map[string]interface{}{
			"applicationId": appID,
			"env":           newEnvStr,
		}
		if createEnvFile != nil {
			payload["createEnvFile"] = *createEnvFile
		}

		_, err = c.doRequest("POST", "application.saveEnvironment", payload)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(100*(i+1)) * time.Millisecond) // Backoff
			continue
		}

		// Verify write
		verifyApp, err := c.GetApplication(appID)
		if err != nil {
			// If we can't verify, we have to assume it worked or retry
			lastErr = fmt.Errorf("failed to verify environment update: %w", err)
			time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
			continue
		}
		if verifyApp.Env == newEnvStr {
			return nil // Success
		}
		lastErr = fmt.Errorf("environment update conflict, retrying")
	}
	return lastErr
}

func (c *DokployClient) UpdateComposeEnv(composeID string, updateFn func(envMap map[string]string), _ *bool) error {
	var lastErr error
	for i := 0; i < 5; i++ { // Retry up to 5 times
		comp, err := c.GetCompose(composeID)
		if err != nil {
			return err
		}

		envMap := ParseEnv(comp.Env)
		originalEnvStr := comp.Env

		updateFn(envMap) // Modify the map

		newEnvStr := formatEnv(envMap)

		if newEnvStr == originalEnvStr {
			return nil // No changes to be made
		}

		payload := map[string]interface{}{
			"composeId": composeID,
			"env":       newEnvStr,
		}

		_, err = c.doRequest("POST", "compose.update", payload)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(100*(i+1)) * time.Millisecond) // Backoff
			continue
		}

		// Verify write
		verifyComp, err := c.GetCompose(composeID)
		if err != nil {
			// If we can't verify, we have to assume it worked or retry
			lastErr = fmt.Errorf("failed to verify environment update: %w", err)
			time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
			continue
		}
		if verifyComp.Env == newEnvStr {
			return nil // Success
		}
		lastErr = fmt.Errorf("environment update conflict, retrying")
	}
	return lastErr
}

func (c *DokployClient) CreateVariable(appID, key, value, scope string, createEnvFile *bool) (*EnvironmentVariable, error) {
	err := c.UpdateApplicationEnv(appID, func(envMap map[string]string) {
		envMap[key] = value
	}, createEnvFile)

	if err != nil {
		return nil, err
	}

	return &EnvironmentVariable{
		ID:            appID + "_" + key,
		ApplicationID: appID,
		Key:           key,
		Value:         value,
		Scope:         scope,
	}, nil
}

func (c *DokployClient) GetVariablesByApplication(appID string) ([]EnvironmentVariable, error) {
	app, err := c.GetApplication(appID)
	if err != nil {
		return nil, err
	}
	envMap := ParseEnv(app.Env)
	var vars []EnvironmentVariable
	for k, v := range envMap {
		vars = append(vars, EnvironmentVariable{
			ID:            appID + "_" + k,
			ApplicationID: appID,
			Key:           k,
			Value:         v,
			Scope:         "runtime",
		})
	}
	return vars, nil
}

func (c *DokployClient) DeleteVariable(id string, createEnvFile *bool) error {
	parts := strings.SplitN(id, "_", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid variable ID format")
	}
	appID, key := parts[0], parts[1]

	return c.UpdateApplicationEnv(appID, func(envMap map[string]string) {
		delete(envMap, key)
	}, createEnvFile)
}

func ParseEnv(env string) map[string]string {
	m := make(map[string]string)
	if env == "" {
		return m
	}
	lines := strings.Split(env, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	return m
}

func formatEnv(m map[string]string) string {
	var lines []string
	for k, v := range m {
		lines = append(lines, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(lines, "\n")
}

// --- SSH Key ---

type SSHKey struct {
	ID          string `json:"sshKeyId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PrivateKey  string `json:"privateKey"`
	PublicKey   string `json:"publicKey"`
}

func (c *DokployClient) CreateSSHKey(name, description, privateKey, publicKey string) (*SSHKey, error) {
	// Fetch user to get Organization ID
	user, err := c.GetUser()
	if err != nil {
		return nil, fmt.Errorf("failed to get user for organization ID: %w", err)
	}

	payload := map[string]string{
		"name":           name,
		"description":    description,
		"privateKey":     privateKey,
		"publicKey":      publicKey,
		"organizationId": user.OrganizationID,
	}

	resp, err := c.doRequest("POST", "sshKey.create", payload)
	if err != nil {
		return nil, err
	}

	// Handle empty response or boolean by fetching list
	if len(resp) == 0 || string(resp) == "true" {
		return c.findSSHKeyByName(name)
	}

	var wrapper struct {
		SSHKey SSHKey `json:"sshKey"`
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil && wrapper.SSHKey.ID != "" {
		return &wrapper.SSHKey, nil
	}

	var result SSHKey
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	if result.ID == "" {
		return c.findSSHKeyByName(name)
	}

	// Fallback to list lookup if unmarshal failed to produce ID
	return &result, nil
}

func (c *DokployClient) ListSSHKeys() ([]SSHKey, error) {
	resp, err := c.doRequest("GET", "sshKey.all", nil)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		SSHKeys []SSHKey `json:"sshKeys"` // Guessing wrapper
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil && wrapper.SSHKeys != nil {
		return wrapper.SSHKeys, nil
	}

	var list []SSHKey
	if err := json.Unmarshal(resp, &list); err == nil {
		return list, nil
	}

	return nil, fmt.Errorf("failed to parse sshKey.all response")
}

func (c *DokployClient) findSSHKeyByName(name string) (*SSHKey, error) {
	keys, err := c.ListSSHKeys()
	if err != nil {
		return nil, fmt.Errorf("ssh key created but failed to list keys: %w", err)
	}
	for _, key := range keys {
		if key.Name == name {
			return &key, nil
		}
	}
	return nil, fmt.Errorf("ssh key created but not found in list by name: %s", name)
}

func (c *DokployClient) GetSSHKey(id string) (*SSHKey, error) {
	endpoint := fmt.Sprintf("sshKey.one?sshKeyId=%s", id)
	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	var result SSHKey
	if err := json.Unmarshal(resp, &result); err != nil {
		// Try wrapper?
		var wrapper struct {
			SSHKey SSHKey `json:"sshKey"`
		}
		if err2 := json.Unmarshal(resp, &wrapper); err2 == nil {
			return &wrapper.SSHKey, nil
		}
		return nil, err
	}
	return &result, nil
}

func (c *DokployClient) DeleteSSHKey(id string) error {
	payload := map[string]string{
		"sshKeyId": id,
	}
	_, err := c.doRequest("POST", "sshKey.remove", payload)
	return err
}

// --- Volume Backup ---

type VolumeBackup struct {
	ID              string `json:"volumeBackupId"`
	Name            string `json:"name"`
	ServiceType     string `json:"serviceType"`
	ComposeID       string `json:"composeId"`
	AppName         string `json:"appName"`
	ServiceName     string `json:"serviceName"`
	VolumeName      string `json:"volumeName"`
	DestinationID   string `json:"destinationId"`
	CronExpression  string `json:"cronExpression"`
	Prefix          string `json:"prefix"`
	TurnOff         bool   `json:"turnOff"`
	Enabled         bool   `json:"enabled"`
	KeepLatestCount int64  `json:"keepLatestCount"`
}

type BackupDestination struct {
	ID              string `json:"destinationId"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	Provider        string `json:"provider"`
	Bucket          string `json:"bucket"`
	Region          string `json:"region"`
	Endpoint        string `json:"endpoint"`
	AccessKey       string `json:"accessKey"`
	SecretKey       string `json:"secretKey"`
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
}

func (c *DokployClient) CreateVolumeBackup(backup VolumeBackup) (*VolumeBackup, error) {
	payload := map[string]interface{}{
		"name":            backup.Name,
		"serviceType":     backup.ServiceType,
		"composeId":       backup.ComposeID,
		"serviceName":     backup.ServiceName,
		"volumeName":      backup.VolumeName,
		"destinationId":   backup.DestinationID,
		"cronExpression":  backup.CronExpression,
		"turnOff":         backup.TurnOff,
		"enabled":         backup.Enabled,
		"keepLatestCount": backup.KeepLatestCount,
	}
	if payload["serviceType"] == "" {
		payload["serviceType"] = "compose"
	}
	if backup.Prefix != "" {
		payload["prefix"] = backup.Prefix
	}
	if backup.AppName != "" {
		payload["appName"] = backup.AppName
	}

	resp, err := c.doRequest("POST", "volumeBackups.create", payload)
	if err != nil {
		return nil, err
	}

	created, parseErr := parseVolumeBackupResponse(resp)
	if parseErr == nil && created.ID != "" {
		return created, nil
	}

	found, findErr := c.findVolumeBackupByTarget(backup.ComposeID, backup.Name, backup.ServiceName, backup.VolumeName)
	if findErr != nil {
		return nil, fmt.Errorf("volume backup created but response was not parseable (%v) and lookup failed: %w", parseErr, findErr)
	}
	return found, nil
}

func (c *DokployClient) GetVolumeBackup(id string) (*VolumeBackup, error) {
	endpoint := fmt.Sprintf("volumeBackups.one?volumeBackupId=%s", id)
	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	return parseVolumeBackupResponse(resp)
}

func (c *DokployClient) UpdateVolumeBackup(backup VolumeBackup) (*VolumeBackup, error) {
	payload := map[string]interface{}{
		"volumeBackupId":  backup.ID,
		"name":            backup.Name,
		"serviceType":     backup.ServiceType,
		"composeId":       backup.ComposeID,
		"serviceName":     backup.ServiceName,
		"volumeName":      backup.VolumeName,
		"destinationId":   backup.DestinationID,
		"cronExpression":  backup.CronExpression,
		"turnOff":         backup.TurnOff,
		"enabled":         backup.Enabled,
		"keepLatestCount": backup.KeepLatestCount,
	}
	if payload["serviceType"] == "" {
		payload["serviceType"] = "compose"
	}
	if backup.Prefix != "" {
		payload["prefix"] = backup.Prefix
	}
	if backup.AppName != "" {
		payload["appName"] = backup.AppName
	}

	resp, err := c.doRequest("POST", "volumeBackups.update", payload)
	if err != nil {
		return nil, err
	}

	updated, parseErr := parseVolumeBackupResponse(resp)
	if parseErr == nil && updated.ID != "" {
		return updated, nil
	}
	return c.GetVolumeBackup(backup.ID)
}

func (c *DokployClient) DeleteVolumeBackup(id string) error {
	payload := map[string]string{
		"volumeBackupId": id,
	}
	_, err := c.doRequest("POST", "volumeBackups.delete", payload)
	if err == nil {
		return nil
	}

	// Backward compatibility with older Dokploy versions.
	_, removeErr := c.doRequest("POST", "volumeBackups.remove", payload)
	if removeErr != nil {
		return fmt.Errorf("volumeBackups.delete failed: %w; volumeBackups.remove fallback failed: %w", err, removeErr)
	}
	return nil
}

func (c *DokployClient) ListVolumeBackups(composeID string) ([]VolumeBackup, error) {
	endpoint := fmt.Sprintf("volumeBackups.list?id=%s&volumeBackupType=compose", composeID)
	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		legacyEndpoint := fmt.Sprintf("volumeBackups.all?id=%s&type=compose", composeID)
		legacyResp, legacyErr := c.doRequest("GET", legacyEndpoint, nil)
		if legacyErr != nil {
			return nil, fmt.Errorf("volumeBackups.list failed: %w; volumeBackups.all fallback failed: %w", err, legacyErr)
		}
		return parseVolumeBackupListResponse(legacyResp)
	}
	return parseVolumeBackupListResponse(resp)
}

func (c *DokployClient) ListBackupDestinations() ([]BackupDestination, error) {
	resp, err := c.doRequest("GET", "destination.all", nil)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Destinations []BackupDestination `json:"destinations"`
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil && wrapper.Destinations != nil {
		return wrapper.Destinations, nil
	}

	var list []BackupDestination
	if err := json.Unmarshal(resp, &list); err == nil {
		return list, nil
	}

	return nil, fmt.Errorf("failed to parse destination.all response")
}

func (c *DokployClient) CreateBackupDestination(destination BackupDestination) (*BackupDestination, error) {
	provider := destination.Provider
	if provider == "" {
		provider = destination.Type
	}

	accessKey := destination.AccessKey
	if accessKey == "" {
		accessKey = destination.AccessKeyID
	}

	secretKey := destination.SecretKey
	if secretKey == "" {
		secretKey = destination.SecretAccessKey
	}

	payload := map[string]interface{}{
		"name":            destination.Name,
		"provider":        provider,
		"accessKey":       accessKey,
		"bucket":          destination.Bucket,
		"region":          destination.Region,
		"endpoint":        destination.Endpoint,
		"secretAccessKey": secretKey,
	}
	if payload["provider"] == "" {
		payload["provider"] = "s3"
	}
	// destination.create requires region and endpoint; allow empty values to pass
	// through so Dokploy can return explicit validation diagnostics.

	resp, err := c.doRequest("POST", "destination.create", payload)
	if err != nil {
		return nil, err
	}

	created, parseErr := parseBackupDestinationResponse(resp)
	if parseErr == nil && created.ID != "" {
		return created, nil
	}

	found, findErr := c.FindBackupDestinationByName(destination.Name)
	if findErr != nil {
		return nil, fmt.Errorf("destination created but response was not parseable (%v) and lookup failed: %w", parseErr, findErr)
	}
	return found, nil
}

func (c *DokployClient) GetBackupDestination(id string) (*BackupDestination, error) {
	endpoint := fmt.Sprintf("destination.one?destinationId=%s", id)
	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	return parseBackupDestinationResponse(resp)
}

func (c *DokployClient) UpdateBackupDestination(destination BackupDestination) (*BackupDestination, error) {
	provider := destination.Provider
	if provider == "" {
		provider = destination.Type
	}

	accessKey := destination.AccessKey
	if accessKey == "" {
		accessKey = destination.AccessKeyID
	}

	secretKey := destination.SecretKey
	if secretKey == "" {
		secretKey = destination.SecretAccessKey
	}

	payload := map[string]interface{}{
		"destinationId":   destination.ID,
		"name":            destination.Name,
		"provider":        provider,
		"accessKey":       accessKey,
		"bucket":          destination.Bucket,
		"region":          destination.Region,
		"endpoint":        destination.Endpoint,
		"secretAccessKey": secretKey,
	}
	if payload["provider"] == "" {
		payload["provider"] = "s3"
	}
	// destination.update requires region and endpoint; allow empty values to pass
	// through so Dokploy can return explicit validation diagnostics.

	resp, err := c.doRequest("POST", "destination.update", payload)
	if err != nil {
		return nil, err
	}

	updated, parseErr := parseBackupDestinationResponse(resp)
	if parseErr == nil && updated.ID != "" {
		return updated, nil
	}
	return c.GetBackupDestination(destination.ID)
}

func (c *DokployClient) DeleteBackupDestination(id string) error {
	payload := map[string]string{
		"destinationId": id,
	}
	_, err := c.doRequest("POST", "destination.remove", payload)
	return err
}

func (c *DokployClient) FindBackupDestinationByName(name string) (*BackupDestination, error) {
	destinations, err := c.ListBackupDestinations()
	if err != nil {
		return nil, err
	}

	target := strings.TrimSpace(name)
	for _, destination := range destinations {
		if strings.EqualFold(strings.TrimSpace(destination.Name), target) {
			return &destination, nil
		}
	}

	return nil, fmt.Errorf("backup destination not found by name: %s", name)
}

func parseBackupDestinationResponse(resp []byte) (*BackupDestination, error) {
	var wrapper struct {
		Destination BackupDestination `json:"destination"`
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil && wrapper.Destination.ID != "" {
		return &wrapper.Destination, nil
	}

	var direct BackupDestination
	if err := json.Unmarshal(resp, &direct); err == nil && direct.ID != "" {
		return &direct, nil
	}

	return nil, fmt.Errorf("failed to parse destination response")
}

func (c *DokployClient) findVolumeBackupByTarget(composeID, name, serviceName, volumeName string) (*VolumeBackup, error) {
	backups, err := c.ListVolumeBackups(composeID)
	if err != nil {
		return nil, err
	}

	for _, backup := range backups {
		if backup.Name == name && backup.ServiceName == serviceName && backup.VolumeName == volumeName {
			return &backup, nil
		}
	}

	return nil, fmt.Errorf("volume backup not found by target (name=%s, service=%s, volume=%s)", name, serviceName, volumeName)
}

func parseVolumeBackupResponse(resp []byte) (*VolumeBackup, error) {
	var wrapper struct {
		VolumeBackup VolumeBackup `json:"volumeBackup"`
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil && wrapper.VolumeBackup.ID != "" {
		return &wrapper.VolumeBackup, nil
	}

	var direct VolumeBackup
	if err := json.Unmarshal(resp, &direct); err == nil && direct.ID != "" {
		return &direct, nil
	}

	return nil, fmt.Errorf("failed to parse volume backup response")
}

func parseVolumeBackupListResponse(resp []byte) ([]VolumeBackup, error) {
	var wrapper struct {
		VolumeBackups []VolumeBackup `json:"volumeBackups"`
	}
	if err := json.Unmarshal(resp, &wrapper); err == nil && wrapper.VolumeBackups != nil {
		return wrapper.VolumeBackups, nil
	}

	var dataWrapper struct {
		Data []VolumeBackup `json:"data"`
	}
	if err := json.Unmarshal(resp, &dataWrapper); err == nil && dataWrapper.Data != nil {
		return dataWrapper.Data, nil
	}

	var direct []VolumeBackup
	if err := json.Unmarshal(resp, &direct); err == nil {
		return direct, nil
	}

	return nil, fmt.Errorf("failed to parse volumeBackups.all response")
}
