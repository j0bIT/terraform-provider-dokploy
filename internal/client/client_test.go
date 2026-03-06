package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func boolPointer(v bool) *bool {
	return &v
}

func int64Pointer(v int64) *int64 {
	return &v
}

func TestReadTraefikConfig_ParsesRawStringResponse(t *testing.T) {
	expected := "entryPoints:\n  web:\n    address: :80\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/settings.readTraefikConfig" {
			t.Fatalf("unexpected endpoint: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(expected))
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")
	got, err := c.ReadTraefikConfig(nil)
	if err != nil {
		t.Fatalf("ReadTraefikConfig returned error: %v", err)
	}
	if got != expected {
		t.Fatalf("unexpected config: got %q want %q", got, expected)
	}
}

func TestReadTraefikConfig_ParsesWrappedResponse(t *testing.T) {
	expected := "http:\n  routers: {}\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/settings.readTraefikConfig" {
			t.Fatalf("unexpected endpoint: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"traefikConfig":"http:\n  routers: {}\n"}}`))
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")
	got, err := c.ReadTraefikConfig(nil)
	if err != nil {
		t.Fatalf("ReadTraefikConfig returned error: %v", err)
	}
	if got != expected {
		t.Fatalf("unexpected config: got %q want %q", got, expected)
	}
}

func TestReadTraefikConfig_IncludesServerIDInQuery(t *testing.T) {
	serverID := "srv_123"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/settings.readTraefikConfig" {
			t.Fatalf("unexpected endpoint: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("serverId"); got != serverID {
			t.Fatalf("unexpected serverId query: got %q want %q", got, serverID)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"traefikConfig":"http:\n  routers: {}\n"}`))
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")
	if _, err := c.ReadTraefikConfig(&serverID); err != nil {
		t.Fatalf("ReadTraefikConfig returned error: %v", err)
	}
}

func TestUpdateTraefikConfig_SendsExpectedPayload(t *testing.T) {
	serverID := "srv_321"
	expectedConfig := "http:\n  routers:\n    test: {}\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/settings.updateTraefikConfig" {
			t.Fatalf("unexpected endpoint: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}

		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode payload: %v", err)
		}

		if got := payload["traefikConfig"]; got != expectedConfig {
			t.Fatalf("unexpected traefikConfig: %#v", got)
		}
		if got := payload["serverId"]; got != serverID {
			t.Fatalf("unexpected serverId: %#v", got)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")
	if err := c.UpdateTraefikConfig(&serverID, expectedConfig); err != nil {
		t.Fatalf("UpdateTraefikConfig returned error: %v", err)
	}
}

func TestReadWebServerTraefikConfig_UsesScopedEndpointAndKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/settings.readWebServerTraefikConfig" {
			t.Fatalf("unexpected endpoint: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"webServerTraefikConfig":"http:\n  routers: {}\n"}`))
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")
	got, err := c.ReadWebServerTraefikConfig(nil)
	if err != nil {
		t.Fatalf("ReadWebServerTraefikConfig returned error: %v", err)
	}
	if got != "http:\n  routers: {}\n" {
		t.Fatalf("unexpected config: %q", got)
	}
}

func TestUpdateWebServerTraefikConfig_UsesScopedEndpointAndPayloadKey(t *testing.T) {
	expectedConfig := "http:\n  routers:\n    dokploy-router-app: {}\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/settings.updateWebServerTraefikConfig" {
			t.Fatalf("unexpected endpoint: %s", r.URL.Path)
		}

		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode payload: %v", err)
		}

		if got := payload["webServerTraefikConfig"]; got != expectedConfig {
			t.Fatalf("unexpected webServerTraefikConfig payload: %#v", got)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")
	if err := c.UpdateWebServerTraefikConfig(nil, expectedConfig); err != nil {
		t.Fatalf("UpdateWebServerTraefikConfig returned error: %v", err)
	}
}

func TestUpdateMiddlewareTraefikConfig_UsesScopedEndpointAndPayloadKey(t *testing.T) {
	expectedConfig := "http:\n  middlewares:\n    dokploy-tailnet-only: {}\n"
	serverID := "srv_abc"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/settings.updateMiddlewareTraefikConfig" {
			t.Fatalf("unexpected endpoint: %s", r.URL.Path)
		}

		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode payload: %v", err)
		}

		if got := payload["middlewareTraefikConfig"]; got != expectedConfig {
			t.Fatalf("unexpected middlewareTraefikConfig payload: %#v", got)
		}
		if got := payload["serverId"]; got != serverID {
			t.Fatalf("unexpected serverId: %#v", got)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")
	if err := c.UpdateMiddlewareTraefikConfig(&serverID, expectedConfig); err != nil {
		t.Fatalf("UpdateMiddlewareTraefikConfig returned error: %v", err)
	}
}

func TestReloadTraefik_SendsExpectedPayload(t *testing.T) {
	serverID := "srv_654"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/settings.reloadTraefik" {
			t.Fatalf("unexpected endpoint: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}

		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode payload: %v", err)
		}

		if got := payload["serverId"]; got != serverID {
			t.Fatalf("unexpected serverId: %#v", got)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")
	if err := c.ReloadTraefik(&serverID); err != nil {
		t.Fatalf("ReloadTraefik returned error: %v", err)
	}
}

func TestDeleteApplication_UsesDeleteEndpoint(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/application.stop", "/application.delete":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	if err := c.DeleteApplication("app-123"); err != nil {
		t.Fatalf("DeleteApplication returned error: %v", err)
	}

	expected := []string{"/application.stop", "/application.delete"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("unexpected call order: got %v want %v", calls, expected)
	}
}

func TestDeleteApplication_FallsBackToRemove(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/application.stop":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`stop failed`))
		case "/application.delete":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`not found`))
		case "/application.remove":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	if err := c.DeleteApplication("app-123"); err != nil {
		t.Fatalf("DeleteApplication returned error: %v", err)
	}

	expected := []string{"/application.stop", "/application.delete", "/application.remove"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("unexpected call order: got %v want %v", calls, expected)
	}
}

func TestDeleteApplication_ReturnsErrorWhenDeleteAndRemoveFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/application.stop":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`stop failed`))
		case "/application.delete":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`delete failed`))
		case "/application.remove":
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`remove failed`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	err := c.DeleteApplication("app-123")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "application.delete failed") {
		t.Fatalf("expected delete failure in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "application.remove fallback failed") {
		t.Fatalf("expected remove fallback failure in error, got: %v", err)
	}
}

func TestCreateApplication_IncludesPreviewPayload(t *testing.T) {
	var updatePayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/application.create":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"application":{"applicationId":"app-123"}}`))
		case "/application.update":
			if err := json.NewDecoder(r.Body).Decode(&updatePayload); err != nil {
				t.Fatalf("failed to decode application.update payload: %v", err)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"application":{"applicationId":"app-123","name":"rssmate"}}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")
	_, err := c.CreateApplication(Application{
		Name:                                  "rssmate",
		EnvironmentID:                         "env-123",
		Branch:                                "main",
		BuildType:                             "dockerfile",
		SourceType:                            "github",
		AutoDeploy:                            true,
		IsPreviewDeploymentsActive:            boolPointer(true),
		PreviewWildcard:                       "*.vanoorschot.dev",
		PreviewPort:                           int64Pointer(8084),
		PreviewPath:                           "/",
		PreviewHTTPS:                          boolPointer(true),
		PreviewCertificateType:                "letsencrypt",
		PreviewLimit:                          int64Pointer(3),
		PreviewRequireCollaboratorPermissions: boolPointer(true),
		PreviewEnv:                            "FEATURE_FLAG=1",
		PreviewBuildArgs:                      "VERSION=preview",
		PreviewLabels:                         []string{"preview=true"},
		LabelsSwarm: map[string]string{
			"traefik.enable": "true",
		},
	})
	if err != nil {
		t.Fatalf("CreateApplication returned error: %v", err)
	}

	if updatePayload["isPreviewDeploymentsActive"] != true {
		t.Fatalf("missing isPreviewDeploymentsActive in payload: %#v", updatePayload["isPreviewDeploymentsActive"])
	}
	if updatePayload["previewWildcard"] != "*.vanoorschot.dev" {
		t.Fatalf("missing previewWildcard in payload: %#v", updatePayload["previewWildcard"])
	}
	if updatePayload["previewPort"] != float64(8084) {
		t.Fatalf("missing previewPort in payload: %#v", updatePayload["previewPort"])
	}
	if updatePayload["previewPath"] != "/" {
		t.Fatalf("missing previewPath in payload: %#v", updatePayload["previewPath"])
	}
	if updatePayload["previewHttps"] != true {
		t.Fatalf("missing previewHttps in payload: %#v", updatePayload["previewHttps"])
	}
	if updatePayload["previewCertificateType"] != "letsencrypt" {
		t.Fatalf("missing previewCertificateType in payload: %#v", updatePayload["previewCertificateType"])
	}
	if updatePayload["previewLimit"] != float64(3) {
		t.Fatalf("missing previewLimit in payload: %#v", updatePayload["previewLimit"])
	}
	if updatePayload["previewRequireCollaboratorPermissions"] != true {
		t.Fatalf("missing previewRequireCollaboratorPermissions in payload: %#v", updatePayload["previewRequireCollaboratorPermissions"])
	}
	if updatePayload["previewEnv"] != "FEATURE_FLAG=1" {
		t.Fatalf("missing previewEnv in payload: %#v", updatePayload["previewEnv"])
	}
	if updatePayload["previewBuildArgs"] != "VERSION=preview" {
		t.Fatalf("missing previewBuildArgs in payload: %#v", updatePayload["previewBuildArgs"])
	}
	previewLabels, ok := updatePayload["previewLabels"].([]interface{})
	if !ok || len(previewLabels) != 1 || previewLabels[0] != "preview=true" {
		t.Fatalf("missing previewLabels in payload: %#v", updatePayload["previewLabels"])
	}
	labelsSwarm, ok := updatePayload["labelsSwarm"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing labelsSwarm in payload: %#v", updatePayload["labelsSwarm"])
	}
	if labelsSwarm["traefik.enable"] != "true" {
		t.Fatalf("unexpected labelsSwarm payload: %#v", updatePayload["labelsSwarm"])
	}
}

func TestUpdateApplication_SendsExplicitFalseAndZeroPreviewValues(t *testing.T) {
	var updatePayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/application.update":
			if err := json.NewDecoder(r.Body).Decode(&updatePayload); err != nil {
				t.Fatalf("failed to decode application.update payload: %v", err)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"application":{"applicationId":"app-123","name":"rssmate","autoDeploy":false}}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")
	_, err := c.UpdateApplication(Application{
		ID:                                    "app-123",
		Name:                                  "rssmate",
		Branch:                                "main",
		BuildType:                             "dockerfile",
		SourceType:                            "github",
		AutoDeploy:                            false,
		IsPreviewDeploymentsActive:            boolPointer(false),
		PreviewPort:                           int64Pointer(0),
		PreviewHTTPS:                          boolPointer(false),
		PreviewLimit:                          int64Pointer(0),
		PreviewRequireCollaboratorPermissions: boolPointer(false),
		LabelsSwarm: map[string]string{
			"traefik.enable": "true",
		},
	})
	if err != nil {
		t.Fatalf("UpdateApplication returned error: %v", err)
	}

	if updatePayload["isPreviewDeploymentsActive"] != false {
		t.Fatalf("expected false isPreviewDeploymentsActive, got: %#v", updatePayload["isPreviewDeploymentsActive"])
	}
	if updatePayload["previewPort"] != float64(0) {
		t.Fatalf("expected zero previewPort, got: %#v", updatePayload["previewPort"])
	}
	if updatePayload["previewHttps"] != false {
		t.Fatalf("expected false previewHttps, got: %#v", updatePayload["previewHttps"])
	}
	if updatePayload["previewLimit"] != float64(0) {
		t.Fatalf("expected zero previewLimit, got: %#v", updatePayload["previewLimit"])
	}
	if updatePayload["previewRequireCollaboratorPermissions"] != false {
		t.Fatalf("expected false previewRequireCollaboratorPermissions, got: %#v", updatePayload["previewRequireCollaboratorPermissions"])
	}
	labelsSwarm, ok := updatePayload["labelsSwarm"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing labelsSwarm in payload: %#v", updatePayload["labelsSwarm"])
	}
	if labelsSwarm["traefik.enable"] != "true" {
		t.Fatalf("unexpected labelsSwarm payload: %#v", updatePayload["labelsSwarm"])
	}
}

func TestCreateMount_UsesMountsCreateEndpointAndPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mounts.create":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode payload: %v", err)
			}

			if payload["serviceId"] != "app-123" {
				t.Fatalf("unexpected serviceId: %#v", payload["serviceId"])
			}
			if payload["type"] != "volume" {
				t.Fatalf("unexpected type: %#v", payload["type"])
			}
			if payload["serviceType"] != "application" {
				t.Fatalf("unexpected serviceType: %#v", payload["serviceType"])
			}
			if payload["mountPath"] != "/data" {
				t.Fatalf("unexpected mountPath: %#v", payload["mountPath"])
			}
			if payload["volumeName"] != "rssmate-sqlite-data" {
				t.Fatalf("unexpected volumeName: %#v", payload["volumeName"])
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"mount":{"mountId":"mount-123","applicationId":"app-123","mountType":"volume","mountPath":"/data","volumeName":"rssmate-sqlite-data"}}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")
	mount, err := c.CreateMount(Mount{
		ApplicationID: "app-123",
		MountType:     "volume",
		MountPath:     "/data",
		VolumeName:    "rssmate-sqlite-data",
	})
	if err != nil {
		t.Fatalf("CreateMount returned error: %v", err)
	}
	if mount.ID != "mount-123" {
		t.Fatalf("unexpected mount ID: got %q want %q", mount.ID, "mount-123")
	}
}

func TestCreateMount_FallbackLookupOnBooleanResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mounts.create":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`true`))
		case "/mounts.allNamedByApplicationId":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"mounts":[{"mountId":"mount-lookup","serviceId":"app-123","serviceType":"application","type":"volume","mountPath":"/data","volumeName":"rssmate-sqlite-data"}]}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")
	mount, err := c.CreateMount(Mount{
		ApplicationID: "app-123",
		MountType:     "volume",
		MountPath:     "/data",
		VolumeName:    "rssmate-sqlite-data",
	})
	if err != nil {
		t.Fatalf("CreateMount returned error: %v", err)
	}
	if mount.ID != "mount-lookup" {
		t.Fatalf("unexpected mount ID: got %q want %q", mount.ID, "mount-lookup")
	}
}

func TestCreateMount_BooleanResponseWithEmptyLookupDoesNotFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mounts.create":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`true`))
		case "/mounts.allNamedByApplicationId":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(``))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")
	mount, err := c.CreateMount(Mount{
		ApplicationID: "app-123",
		MountType:     "volume",
		MountPath:     "/data",
		VolumeName:    "rssmate-sqlite-data",
	})
	if err != nil {
		t.Fatalf("CreateMount returned error: %v", err)
	}
	if mount.ApplicationID != "app-123" {
		t.Fatalf("unexpected application ID: got %q want %q", mount.ApplicationID, "app-123")
	}
	if mount.MountPath != "/data" {
		t.Fatalf("unexpected mount path: got %q want %q", mount.MountPath, "/data")
	}
	if mount.VolumeName != "rssmate-sqlite-data" {
		t.Fatalf("unexpected volume name: got %q want %q", mount.VolumeName, "rssmate-sqlite-data")
	}
}

func TestDeleteMount_UsesRemoveEndpoint(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/mounts.remove":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`true`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")
	if err := c.DeleteMount("mount-123"); err != nil {
		t.Fatalf("DeleteMount returned error: %v", err)
	}

	expected := []string{"/mounts.remove"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("unexpected call order: got %v want %v", calls, expected)
	}
}

func TestDeleteMount_FallsBackToLegacyEndpoint(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/mounts.remove":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`not found`))
		case "/mount.delete":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`true`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")
	if err := c.DeleteMount("mount-123"); err != nil {
		t.Fatalf("DeleteMount returned error: %v", err)
	}

	expected := []string{"/mounts.remove", "/mount.delete"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("unexpected call order: got %v want %v", calls, expected)
	}
}

func TestUpdateProjectEnv_UpdatesProjectEnvironment(t *testing.T) {
	projectEnv := "A=1\nB=2"
	updateCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/project.one":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"projectId":"proj-1","name":"Project One","description":"Test project","env":"` + strings.ReplaceAll(projectEnv, "\n", `\n`) + `"}`))
		case "/project.update":
			updateCalls++

			var payload struct {
				ProjectID   string `json:"projectId"`
				Name        string `json:"name"`
				Description string `json:"description"`
				Env         string `json:"env"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode project.update payload: %v", err)
			}

			if payload.ProjectID != "proj-1" {
				t.Fatalf("unexpected project ID: got %q want %q", payload.ProjectID, "proj-1")
			}
			if payload.Name != "Project One" {
				t.Fatalf("unexpected project name: got %q want %q", payload.Name, "Project One")
			}
			if payload.Description != "Test project" {
				t.Fatalf("unexpected project description: got %q want %q", payload.Description, "Test project")
			}

			projectEnv = payload.Env
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	err := c.UpdateProjectEnv("proj-1", func(envMap map[string]string) {
		envMap["C"] = "3"
	})
	if err != nil {
		t.Fatalf("UpdateProjectEnv returned error: %v", err)
	}

	if updateCalls == 0 {
		t.Fatal("expected project.update to be called at least once")
	}

	finalEnv := ParseEnv(projectEnv)
	if finalEnv["A"] != "1" || finalEnv["B"] != "2" || finalEnv["C"] != "3" {
		t.Fatalf("unexpected final env map: %#v", finalEnv)
	}
}

func TestUpdateProjectEnv_NoChangesSkipsUpdate(t *testing.T) {
	updateCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/project.one":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"projectId":"proj-1","name":"Project One","description":"Test project","env":"A=1"}`))
		case "/project.update":
			updateCalls++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	err := c.UpdateProjectEnv("proj-1", func(envMap map[string]string) {
		envMap["A"] = "1"
	})
	if err != nil {
		t.Fatalf("UpdateProjectEnv returned error: %v", err)
	}

	if updateCalls != 0 {
		t.Fatalf("expected no project.update call, got %d", updateCalls)
	}
}

func TestCreateDatabase_MySQLDirectResponseUsesMysqlIDAsID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mysql.create":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"mysqlId":"mysql-123","name":"test-db","appName":"test-db"}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	db, err := c.CreateDatabase("project-1", "env-1", "test-db", "mysql", "secret", "mysql:8")
	if err != nil {
		t.Fatalf("CreateDatabase returned error: %v", err)
	}
	if db.ID != "mysql-123" {
		t.Fatalf("unexpected database ID: got %q want %q", db.ID, "mysql-123")
	}
	if db.Type != "mysql" {
		t.Fatalf("unexpected database type: got %q want %q", db.Type, "mysql")
	}
}

func TestCreateDatabase_MySQLWrappedResponseUsesMysqlIDAsID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mysql.create":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"database":{"mysqlId":"mysql-456","name":"test-db","appName":"test-db"}}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	db, err := c.CreateDatabase("project-1", "env-1", "test-db", "mysql", "secret", "mysql:8")
	if err != nil {
		t.Fatalf("CreateDatabase returned error: %v", err)
	}
	if db.ID != "mysql-456" {
		t.Fatalf("unexpected database ID: got %q want %q", db.ID, "mysql-456")
	}
	if db.Type != "mysql" {
		t.Fatalf("unexpected database type: got %q want %q", db.Type, "mysql")
	}
}

func TestCreateVolumeBackup_UsesComposeEndpointAndPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/volumeBackups.create":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode payload: %v", err)
			}

			if payload["serviceType"] != "compose" {
				t.Fatalf("unexpected serviceType: %#v", payload["serviceType"])
			}
			if payload["composeId"] != "compose-123" {
				t.Fatalf("unexpected composeId: %#v", payload["composeId"])
			}
			if payload["destinationId"] != "dest-123" {
				t.Fatalf("unexpected destinationId: %#v", payload["destinationId"])
			}
			if payload["appName"] != "fca-ghost-kqlble" {
				t.Fatalf("unexpected appName: %#v", payload["appName"])
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"volumeBackup":{"volumeBackupId":"vb-123","name":"ghost-content","composeId":"compose-123","serviceName":"ghost","volumeName":"ghost-content-data","destinationId":"dest-123","cronExpression":"0 3 * * *","enabled":true,"keepLatestCount":14}}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	backup, err := c.CreateVolumeBackup(VolumeBackup{
		Name:            "ghost-content",
		ComposeID:       "compose-123",
		AppName:         "fca-ghost-kqlble",
		ServiceName:     "ghost",
		VolumeName:      "ghost-content-data",
		DestinationID:   "dest-123",
		CronExpression:  "0 3 * * *",
		Enabled:         true,
		TurnOff:         false,
		KeepLatestCount: 14,
	})
	if err != nil {
		t.Fatalf("CreateVolumeBackup returned error: %v", err)
	}
	if backup.ID != "vb-123" {
		t.Fatalf("unexpected backup ID: got %q want %q", backup.ID, "vb-123")
	}
}

func TestCreateVolumeBackup_FallbackLookupOnBooleanResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/volumeBackups.create":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`true`))
		case "/volumeBackups.list":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`not found`))
		case "/volumeBackups.all":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"volumeBackupId":"vb-lookup","name":"ghost-content","composeId":"compose-123","serviceName":"ghost","volumeName":"ghost-content-data","destinationId":"dest-123"}]`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	backup, err := c.CreateVolumeBackup(VolumeBackup{
		Name:          "ghost-content",
		ComposeID:     "compose-123",
		ServiceName:   "ghost",
		VolumeName:    "ghost-content-data",
		DestinationID: "dest-123",
	})
	if err != nil {
		t.Fatalf("CreateVolumeBackup returned error: %v", err)
	}
	if backup.ID != "vb-lookup" {
		t.Fatalf("unexpected backup ID: got %q want %q", backup.ID, "vb-lookup")
	}
}

func TestDeleteVolumeBackup_UsesDeleteEndpoint(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/volumeBackups.delete":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`true`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")
	if err := c.DeleteVolumeBackup("vb-123"); err != nil {
		t.Fatalf("DeleteVolumeBackup returned error: %v", err)
	}

	expected := []string{"/volumeBackups.delete"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("unexpected call order: got %v want %v", calls, expected)
	}
}

func TestFindBackupDestinationByName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/destination.all":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"destinations":[{"destinationId":"dest-hetzner","name":"Hetzner backup s3 bucket","type":"s3"}]}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	destination, err := c.FindBackupDestinationByName("hetzner backup s3 bucket")
	if err != nil {
		t.Fatalf("FindBackupDestinationByName returned error: %v", err)
	}
	if destination.ID != "dest-hetzner" {
		t.Fatalf("unexpected destination ID: got %q want %q", destination.ID, "dest-hetzner")
	}
}

func TestCreateBackupDestination_UsesDestinationCreateEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/destination.create":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode payload: %v", err)
			}

			if payload["name"] != "Hetzner backup s3 bucket" {
				t.Fatalf("unexpected name: %#v", payload["name"])
			}
			if payload["provider"] != "s3" {
				t.Fatalf("unexpected provider: %#v", payload["provider"])
			}
			if payload["bucket"] != "backups-2f6fe75d" {
				t.Fatalf("unexpected bucket: %#v", payload["bucket"])
			}
			if payload["accessKey"] != "access-key" {
				t.Fatalf("unexpected accessKey: %#v", payload["accessKey"])
			}
			if payload["secretAccessKey"] != "secret-key" {
				t.Fatalf("unexpected secretAccessKey: %#v", payload["secretAccessKey"])
			}
			if _, ok := payload["accessKeyId"]; ok {
				t.Fatalf("unexpected alias field accessKeyId in payload")
			}
			if _, ok := payload["secretKey"]; ok {
				t.Fatalf("unexpected alias field secretKey in payload")
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"destination":{"destinationId":"dest-123","name":"Hetzner backup s3 bucket","type":"s3","bucket":"backups-2f6fe75d","region":"nbg1","endpoint":"https://nbg1.your-objectstorage.com"}}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	destination, err := c.CreateBackupDestination(BackupDestination{
		Name:            "Hetzner backup s3 bucket",
		Type:            "s3",
		Bucket:          "backups-2f6fe75d",
		Region:          "nbg1",
		Endpoint:        "https://nbg1.your-objectstorage.com",
		AccessKeyID:     "access-key",
		SecretAccessKey: "secret-key",
	})
	if err != nil {
		t.Fatalf("CreateBackupDestination returned error: %v", err)
	}
	if destination.ID != "dest-123" {
		t.Fatalf("unexpected destination ID: got %q want %q", destination.ID, "dest-123")
	}
}

func TestUpdateBackupDestination_FallsBackToGetDestination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/destination.update":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode payload: %v", err)
			}
			if payload["provider"] != "s3" {
				t.Fatalf("unexpected provider: %#v", payload["provider"])
			}
			if payload["accessKey"] != "access-key" {
				t.Fatalf("unexpected accessKey: %#v", payload["accessKey"])
			}
			if payload["secretAccessKey"] != "secret-key" {
				t.Fatalf("unexpected secretAccessKey: %#v", payload["secretAccessKey"])
			}
			if _, ok := payload["accessKeyId"]; ok {
				t.Fatalf("unexpected alias field accessKeyId in payload")
			}
			if _, ok := payload["secretKey"]; ok {
				t.Fatalf("unexpected alias field secretKey in payload")
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`true`))
		case "/destination.one":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"destinationId":"dest-123","name":"Hetzner backup s3 bucket","type":"s3","bucket":"backups-2f6fe75d","region":"nbg1","endpoint":"https://nbg1.your-objectstorage.com"}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	destination, err := c.UpdateBackupDestination(BackupDestination{
		ID:              "dest-123",
		Name:            "Hetzner backup s3 bucket",
		Type:            "s3",
		Bucket:          "backups-2f6fe75d",
		Region:          "nbg1",
		Endpoint:        "https://nbg1.your-objectstorage.com",
		AccessKeyID:     "access-key",
		SecretAccessKey: "secret-key",
	})
	if err != nil {
		t.Fatalf("UpdateBackupDestination returned error: %v", err)
	}
	if destination.ID != "dest-123" {
		t.Fatalf("unexpected destination ID: got %q want %q", destination.ID, "dest-123")
	}
}
