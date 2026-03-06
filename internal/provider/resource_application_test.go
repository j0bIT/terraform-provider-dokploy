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
