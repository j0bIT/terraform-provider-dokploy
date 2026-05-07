package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNormalizeApplicationMountPlan_DefaultsMountTypeToVolume(t *testing.T) {
	mount := normalizeApplicationMountPlan(ApplicationMountResourceModel{
		MountType:  types.StringNull(),
		MountPath:  types.StringValue("/data"),
		VolumeName: types.StringValue("rssmate-sqlite-data"),
	})

	if mount.MountType != "volume" {
		t.Fatalf("unexpected mount type: got %q want %q", mount.MountType, "volume")
	}
	if mount.MountPath != "/data" {
		t.Fatalf("unexpected mount path: got %q want %q", mount.MountPath, "/data")
	}
	if mount.VolumeName != "rssmate-sqlite-data" {
		t.Fatalf("unexpected volume name: got %q want %q", mount.VolumeName, "rssmate-sqlite-data")
	}
}

func TestNormalizeApplicationMountPlan_TrimsValues(t *testing.T) {
	mount := normalizeApplicationMountPlan(ApplicationMountResourceModel{
		MountType:  types.StringValue(" volume "),
		MountPath:  types.StringValue(" /data "),
		VolumeName: types.StringValue(" rssmate-sqlite-data "),
	})

	if mount.MountType != "volume" {
		t.Fatalf("unexpected mount type: got %q want %q", mount.MountType, "volume")
	}
	if mount.MountPath != "/data" {
		t.Fatalf("unexpected mount path: got %q want %q", mount.MountPath, "/data")
	}
	if mount.VolumeName != "rssmate-sqlite-data" {
		t.Fatalf("unexpected volume name: got %q want %q", mount.VolumeName, "rssmate-sqlite-data")
	}
}

func TestOptionalBoolPointerFromPlan(t *testing.T) {
	if got := optionalBoolPointerFromPlan(types.BoolNull()); got != nil {
		t.Fatalf("expected nil pointer for null bool, got %#v", got)
	}
	if got := optionalBoolPointerFromPlan(types.BoolUnknown()); got != nil {
		t.Fatalf("expected nil pointer for unknown bool, got %#v", got)
	}

	got := optionalBoolPointerFromPlan(types.BoolValue(false))
	if got == nil || *got {
		t.Fatalf("expected pointer to false, got %#v", got)
	}
}

func TestOptionalInt64PointerFromPlan(t *testing.T) {
	if got := optionalInt64PointerFromPlan(types.Int64Null()); got != nil {
		t.Fatalf("expected nil pointer for null int64, got %#v", got)
	}
	if got := optionalInt64PointerFromPlan(types.Int64Unknown()); got != nil {
		t.Fatalf("expected nil pointer for unknown int64, got %#v", got)
	}

	got := optionalInt64PointerFromPlan(types.Int64Value(0))
	if got == nil || *got != 0 {
		t.Fatalf("expected pointer to 0, got %#v", got)
	}
}

func TestBuildDockerProviderConfig_OnlyImage(t *testing.T) {
	cfg := buildDockerProviderConfig(ApplicationResourceModel{
		DockerImage: types.StringValue("nginx:latest"),
		RegistryURL: types.StringNull(),
		Username:    types.StringNull(),
		Password:    types.StringNull(),
	})

	if got, want := cfg["dockerImage"], "nginx:latest"; got != want {
		t.Fatalf("dockerImage: got %v want %q", got, want)
	}
	if _, ok := cfg["registryUrl"]; ok {
		t.Fatalf("registryUrl should be omitted when null, got %#v", cfg["registryUrl"])
	}
	if _, ok := cfg["username"]; ok {
		t.Fatalf("username should be omitted when null, got %#v", cfg["username"])
	}
	if _, ok := cfg["password"]; ok {
		t.Fatalf("password should be omitted when null, got %#v", cfg["password"])
	}
}

func TestBuildDockerProviderConfig_OmitsEmptyStrings(t *testing.T) {
	cfg := buildDockerProviderConfig(ApplicationResourceModel{
		DockerImage: types.StringValue("nginx:latest"),
		RegistryURL: types.StringValue(""),
		Username:    types.StringValue(""),
		Password:    types.StringValue(""),
	})

	if _, ok := cfg["registryUrl"]; ok {
		t.Fatalf("registryUrl should be omitted when empty string")
	}
	if _, ok := cfg["username"]; ok {
		t.Fatalf("username should be omitted when empty string")
	}
	if _, ok := cfg["password"]; ok {
		t.Fatalf("password should be omitted when empty string")
	}
}

func TestBuildDockerProviderConfig_OmitsUnknown(t *testing.T) {
	cfg := buildDockerProviderConfig(ApplicationResourceModel{
		DockerImage: types.StringValue("nginx:latest"),
		RegistryURL: types.StringUnknown(),
		Username:    types.StringUnknown(),
		Password:    types.StringUnknown(),
	})

	if _, ok := cfg["registryUrl"]; ok {
		t.Fatalf("registryUrl should be omitted when unknown")
	}
	if _, ok := cfg["username"]; ok {
		t.Fatalf("username should be omitted when unknown")
	}
	if _, ok := cfg["password"]; ok {
		t.Fatalf("password should be omitted when unknown")
	}
}

func TestBuildDockerProviderConfig_AllFieldsSet(t *testing.T) {
	cfg := buildDockerProviderConfig(ApplicationResourceModel{
		DockerImage: types.StringValue("ghcr.io/acme/api:1.2.3"),
		RegistryURL: types.StringValue("https://ghcr.io"),
		Username:    types.StringValue("acme-bot"),
		Password:    types.StringValue("s3cret"),
	})

	if got, want := cfg["dockerImage"], "ghcr.io/acme/api:1.2.3"; got != want {
		t.Fatalf("dockerImage: got %v want %q", got, want)
	}
	if got, want := cfg["registryUrl"], "https://ghcr.io"; got != want {
		t.Fatalf("registryUrl: got %v want %q", got, want)
	}
	if got, want := cfg["username"], "acme-bot"; got != want {
		t.Fatalf("username: got %v want %q", got, want)
	}
	if got, want := cfg["password"], "s3cret"; got != want {
		t.Fatalf("password: got %v want %q", got, want)
	}
}
