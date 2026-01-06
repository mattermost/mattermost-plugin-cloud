package main

import (
	"testing"
)

// TestValidTagIntegration tests the actual Docker Hub API.
// This is an integration test that requires network access
func TestValidTagIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dc := NewDockerClient()

	tests := []struct {
		name        string
		tag         string
		repository  string
		shouldExist bool
	}{
		{
			name:        "existing 11.x tag",
			tag:         "11.2.1",
			repository:  "mattermost/mattermost-enterprise-edition",
			shouldExist: true,
		},
		{
			name:        "existing 10.x tag",
			tag:         "10.12.4",
			repository:  "mattermost/mattermost-enterprise-edition",
			shouldExist: true,
		},
		{
			name:        "non-existent tag",
			tag:         "99.99.99-nonexistent",
			repository:  "mattermost/mattermost-enterprise-edition",
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := dc.ValidTag(tt.tag, tt.repository)
			if err != nil {
				t.Fatalf("ValidTag returned error: %v", err)
			}
			if exists != tt.shouldExist {
				t.Errorf("ValidTag(%s, %s) = %v, want %v",
					tt.tag, tt.repository, exists, tt.shouldExist)
			}
		})
	}
}

// TestGetDigestForTagIntegration tests the actual Docker Hub API
func TestGetDigestForTagIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dc := NewDockerClient()

	digest, err := dc.GetDigestForTag("11.2.1", "mattermost/mattermost-enterprise-edition")
	if err != nil {
		t.Fatalf("GetDigestForTag returned error: %v", err)
	}

	if digest == "" {
		t.Error("GetDigestForTag returned empty digest")
	}

	// Digest should start with sha256:
	if len(digest) < 7 || digest[:7] != "sha256:" {
		t.Errorf("GetDigestForTag returned invalid digest format: %s", digest)
	}

	t.Logf("Got digest: %s", digest)
}
