package provider

import "testing"

func TestResolveComposeVolumeName(t *testing.T) {
	tests := []struct {
		name       string
		appName    string
		volumeName string
		expected   string
	}{
		{
			name:       "adds app prefix",
			appName:    "ghost-6bj1z0",
			volumeName: "ghost-mysql-data",
			expected:   "ghost-6bj1z0_ghost-mysql-data",
		},
		{
			name:       "keeps already prefixed",
			appName:    "ghost-6bj1z0",
			volumeName: "ghost-6bj1z0_ghost-mysql-data",
			expected:   "ghost-6bj1z0_ghost-mysql-data",
		},
		{
			name:       "no app name",
			appName:    "",
			volumeName: "ghost-mysql-data",
			expected:   "ghost-mysql-data",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := resolveComposeVolumeName(test.appName, test.volumeName)
			if got != test.expected {
				t.Fatalf("unexpected value: got %q want %q", got, test.expected)
			}
		})
	}
}

func TestStripComposeVolumePrefix(t *testing.T) {
	got := stripComposeVolumePrefix("ghost-6bj1z0", "ghost-6bj1z0_ghost-mysql-data")
	if got != "ghost-mysql-data" {
		t.Fatalf("unexpected value: got %q want %q", got, "ghost-mysql-data")
	}
}
