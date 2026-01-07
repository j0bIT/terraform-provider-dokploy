package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	AutoDeploy        bool     `json:"autoDeploy"`
	// Enhanced fields
	SourceType         string `json:"sourceType"`
	CustomGitUrl       string `json:"customGitUrl"`
	CustomGitBranch    string `json:"customGitBranch"`
	CustomGitSSHKeyId  string `json:"customGitSSHKeyId"`
	CustomGitBuildPath string `json:"customGitBuildPath"`
	Username           string `json:"username"`
	Password           string `json:"password"`
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

	if app.EnvironmentID != "" {
		payload["environmentId"] = app.EnvironmentID
	}

	resp, err := c.doRequest("POST", "application.update", payload)
	if err != nil {
		return nil, err
	}

	var result Application
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *DokployClient) DeleteApplication(id string) error {
	payload := map[string]string{
		"applicationId": id,
	}
	_, err := c.doRequest("POST", "application.remove", payload)
	return err
}

func (c *DokployClient) DeployApplication(id string) error {
	payload := map[string]string{
		"applicationId": id,
	}
	_, err := c.doRequest("POST", "application.deploy", payload)
	return err
}

// --- Compose ---

type Compose struct {
	ID                string   `json:"composeId"`
	Name              string   `json:"name"`
	ProjectID         string   `json:"projectId"`
	EnvironmentID     string   `json:"environmentId"`
	ComposeFile       string   `json:"composeFile"`
	SourceType        string   `json:"sourceType"`
	CustomGitUrl      string   `json:"customGitUrl"`
	CustomGitBranch   string   `json:"customGitBranch"`
	CustomGitSSHKeyId string   `json:"customGitSSHKeyId"`
	ComposePath       string   `json:"composePath"`
	AutoDeploy        bool     `json:"autoDeploy"`
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

func (c *DokployClient) DeleteCompose(id string) error {
	payload := map[string]string{
		"composeId": id,
	}
	_, err := c.doRequest("POST", "compose.remove", payload)
	return err
}

func (c *DokployClient) DeployCompose(id string) error {
	payload := map[string]string{
		"composeId": id,
	}
	_, err := c.doRequest("POST", "compose.deploy", payload)
	return err
}

// --- Database ---

type Database struct {
	ID                   string                   `json:"databaseId"`
	Name                 string                   `json:"name"`
	AppName              string                   `json:"appName"`
	Type                 string                   `json:"type"`
	ProjectID            string                   `json:"projectId"`
	EnvironmentID        string                   `json:"environmentId"`
	Description          string                   `json:"description"`
	DatabaseName         string                   `json:"databaseName"`
	DatabaseUser         string                   `json:"databaseUser"`
	DatabaseRootPassword string                   `json:"databaseRootPassword"`
	DockerImage          string                   `json:"dockerImage"`
	ExternalPort         int64                    `json:"externalPort"`
	InternalPort         int64                    `json:"internalPort"`
	ServerID             string                   `json:"serverId"`
	ApplicationStatus    string                   `json:"applicationStatus"`
	ReplicaSets          bool                     `json:"replicaSets"`
	Env                  string                   `json:"env"`
	MemoryReservation    string                   `json:"memoryReservation"`
	MemoryLimit          string                   `json:"memoryLimit"`
	CPUReservation       string                   `json:"cpuReservation"`
	CPULimit             string                   `json:"cpuLimit"`
	Command              string                   `json:"command"`
	Args                 []string                 `json:"args"`
	Replicas             int64                    `json:"replicas"`
	HealthCheckSwarm     map[string]interface{}   `json:"healthCheckSwarm"`
	RestartPolicySwarm   map[string]interface{}   `json:"restartPolicySwarm"`
	PlacementSwarm       map[string]interface{}   `json:"placementSwarm"`
	UpdateConfigSwarm    map[string]interface{}   `json:"updateConfigSwarm"`
	RollbackConfigSwarm  map[string]interface{}   `json:"rollbackConfigSwarm"`
	ModeSwarm            map[string]interface{}   `json:"modeSwarm"`
	LabelsSwarm          map[string]string        `json:"labelsSwarm"`
	NetworkSwarm         []map[string]interface{} `json:"networkSwarm"`
	StopGracePeriodSwarm int64                    `json:"stopGracePeriodSwarm"`
	EndpointSpecSwarm    map[string]interface{}   `json:"endpointSpecSwarm"`
	PostgresID           string                   `json:"postgresId"`
	MysqlID              string                   `json:"mysqlId"`
	MariadbID            string                   `json:"mariadbId"`
	MongoID              string                   `json:"mongoId"`
	RedisID              string                   `json:"redisId"`
}

func (c *DokployClient) CreateDatabase(environmentID, dbType, appName, name, description, databaseName, databaseUser, databasePassword, databaseRootPassword, dockerImage, serverID string, replicaSets bool, args []string) (*Database, error) {
	var endpoint string
	payload := map[string]interface{}{
		"environmentId":    environmentID,
		"name":             name,
		"appName":          appName,
		"databaseName":     databaseName,
		"databaseUser":     databaseUser,
		"databasePassword": databasePassword,
		"dockerImage":      dockerImage,
	}

	if description != "" {
		payload["description"] = description
	}
	if serverID != "" {
		payload["serverId"] = serverID
	}
	if args != nil && len(args) > 0 {
		payload["args"] = args
	}

	switch dbType {
	case "postgres":
		endpoint = "postgres.create"
		if dockerImage == "" {
			payload["dockerImage"] = "postgres:15"
		}
	case "mysql":
		endpoint = "mysql.create"
		if databaseRootPassword == "" {
			return nil, fmt.Errorf("databaseRootPassword is required for mysql")
		}
		payload["databaseRootPassword"] = databaseRootPassword
		if dockerImage == "" {
			payload["dockerImage"] = "mysql:8"
		}
	case "mariadb":
		endpoint = "mariadb.create"
		if databaseRootPassword == "" {
			return nil, fmt.Errorf("databaseRootPassword is required for mariadb")
		}
		payload["databaseRootPassword"] = databaseRootPassword
		if dockerImage == "" {
			payload["dockerImage"] = "mariadb:6"
		}
	case "mongo":
		endpoint = "mongo.create"
		if replicaSets {
			payload["replicaSets"] = true
		}
		if dockerImage == "" {
			payload["dockerImage"] = "mongo:15"
		}
	case "redis":
		endpoint = "redis.create"
		if dockerImage == "" {
			payload["dockerImage"] = "redis:8"
		}
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	resp, err := c.doRequest("POST", endpoint, payload)
	if err != nil {
		return nil, err
	}

	var result Database
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	if result.Type == "" {
		result.Type = dbType
	}
	if result.AppName == "" {
		result.AppName = appName
	}
	if result.DatabaseName == "" {
		result.DatabaseName = databaseName
	}
	return &result, nil
}

func (c *DokployClient) UpdateDatabase(dbID, dbType, appName, name, description, databaseName, databaseUser, databasePassword, databaseRootPassword, dockerImage, serverID string, replicaSets bool, env, memoryReservation, memoryLimit, cpuReservation, cpuLimit, command, applicationStatus string, replicas, stopGracePeriod int64, args []string) (*Database, error) {
	var endpoint string
	var idKey string
	payload := map[string]interface{}{
		idKey:  dbID,
		"name": name,
	}

	if appName != "" {
		payload["appName"] = appName
	}
	if description != "" {
		payload["description"] = description
	}
	if databaseName != "" {
		payload["databaseName"] = databaseName
	}
	if databaseUser != "" {
		payload["databaseUser"] = databaseUser
	}
	if databasePassword != "" {
		payload["databasePassword"] = databasePassword
	}
	if databaseRootPassword != "" {
		payload["databaseRootPassword"] = databaseRootPassword
	}
	if dockerImage != "" {
		payload["dockerImage"] = dockerImage
	}
	if serverID != "" {
		payload["serverId"] = serverID
	}
	if env != "" {
		payload["env"] = env
	}
	if memoryReservation != "" {
		payload["memoryReservation"] = memoryReservation
	}
	if memoryLimit != "" {
		payload["memoryLimit"] = memoryLimit
	}
	if cpuReservation != "" {
		payload["cpuReservation"] = cpuReservation
	}
	if cpuLimit != "" {
		payload["cpuLimit"] = cpuLimit
	}
	if command != "" {
		payload["command"] = command
	}
	if applicationStatus != "" {
		payload["applicationStatus"] = applicationStatus
	}
	if replicas > 0 {
		payload["replicas"] = replicas
	}
	if stopGracePeriod > 0 {
		payload["stopGracePeriod"] = stopGracePeriod
	}
	if args != nil && len(args) > 0 {
		payload["args"] = args
	}

	switch dbType {
	case "postgres":
		endpoint = "postgres.update"
		idKey = "postgresId"
	case "mysql":
		endpoint = "mysql.update"
		idKey = "mysqlId"
	case "mariadb":
		endpoint = "mariadb.update"
		idKey = "mariadbId"
	case "mongo":
		endpoint = "mongo.update"
		idKey = "mongoId"
		if replicaSets {
			payload["replicaSets"] = true
		}
	case "redis":
		endpoint = "redis.update"
		idKey = "redisId"
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	payload[idKey] = dbID
	resp, err := c.doRequest("POST", endpoint, payload)
	if err != nil {
		return nil, err
	}

	var result Database
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *DokployClient) DeployDatabase(dbID, dbType string) error {
	var endpoint string
	var idKey string
	switch dbType {
	case "postgres":
		endpoint = "postgres.deploy"
		idKey = "postgresId"
	case "mysql":
		endpoint = "mysql.deploy"
		idKey = "mysqlId"
	case "mariadb":
		endpoint = "mariadb.deploy"
		idKey = "mariadbId"
	case "mongo":
		endpoint = "mongo.deploy"
		idKey = "mongoId"
	case "redis":
		endpoint = "redis.deploy"
		idKey = "redisId"
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}

	payload := map[string]string{
		idKey: dbID,
	}
	_, err := c.doRequest("POST", endpoint, payload)
	return err
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
		valid := false
		if db.ID != "" {
			valid = true
		}
		if db.PostgresID != "" {
			valid = true
		}
		if db.MysqlID != "" {
			valid = true
		}
		if db.MariadbID != "" {
			valid = true
		}
		if db.MongoID != "" {
			valid = true
		}
		if db.RedisID != "" {
			valid = true
		}

		if valid {
			if db.ID == "" {
				if db.PostgresID != "" {
					db.ID = db.PostgresID
				}
				if db.MysqlID != "" {
					db.ID = db.MysqlID
				}
				if db.MariadbID != "" {
					db.ID = db.MariadbID
				}
				if db.MongoID != "" {
					db.ID = db.MongoID
				}
				if db.RedisID != "" {
					db.ID = db.RedisID
				}
			}
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

	if db.ID == "" {
		if db.PostgresID != "" {
			db.ID = db.PostgresID
		}
		if db.MysqlID != "" {
			db.ID = db.MysqlID
		}
		if db.MariadbID != "" {
			db.ID = db.MariadbID
		}
		if db.MongoID != "" {
			db.ID = db.MongoID
		}
		if db.RedisID != "" {
			db.ID = db.RedisID
		}
	}
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
	ID            string `json:"domainId"`
	ApplicationID string `json:"applicationId"`
	ComposeID     string `json:"composeId"`
	ServiceName   string `json:"serviceName"`
	Host          string `json:"host"`
	Path          string `json:"path"`
	Port          int64  `json:"port"`
	HTTPS         bool   `json:"https"`
}

func (c *DokployClient) CreateDomain(domain Domain) (*Domain, error) {
	payload := map[string]interface{}{
		"host":  domain.Host,
		"path":  domain.Path,
		"port":  domain.Port,
		"https": domain.HTTPS,
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
		"domainId":    domain.ID,
		"host":        domain.Host,
		"path":        domain.Path,
		"port":        domain.Port,
		"https":       domain.HTTPS,
		"serviceName": domain.ServiceName,
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
