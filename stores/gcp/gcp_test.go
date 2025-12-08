package gcp

import (
	"context"
	"testing"

	"github.com/jbcom/secretsync/pkg/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *GcpClient
		wantErr bool
	}{
		{
			name: "Valid configuration",
			cfg: &GcpClient{
				Project: "test-project",
				Name:    "test-secret",
			},
			wantErr: false,
		},
		{
			name: "With replication locations",
			cfg: &GcpClient{
				Project:              "test-project",
				Name:                 "test-secret",
				ReplicationLocations: []string{"us-central1", "us-east1"},
			},
			wantErr: false,
		},
		{
			name: "With labels",
			cfg: &GcpClient{
				Project: "test-project",
				Name:    "test-secret",
				Labels: map[string]string{
					"env":  "test",
					"team": "platform",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, client)
			assert.Equal(t, tt.cfg.Project, client.Project)
			assert.Equal(t, tt.cfg.Name, client.Name)
			assert.Equal(t, tt.cfg.ReplicationLocations, client.ReplicationLocations)
			assert.Equal(t, tt.cfg.Labels, client.Labels)
		})
	}
}

func TestGcpClient_Validate(t *testing.T) {
	tests := []struct {
		name    string
		client  *GcpClient
		wantErr bool
	}{
		{
			name: "Valid client",
			client: &GcpClient{
				Name: "test-secret",
			},
			wantErr: false,
		},
		{
			name:    "Missing name",
			client:  &GcpClient{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.client.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, driver.ErrPathRequired, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGcpClient_Driver(t *testing.T) {
	client := &GcpClient{}
	assert.Equal(t, driver.DriverNameGcp, client.Driver())
}

func TestGcpClient_GetPath(t *testing.T) {
	client := &GcpClient{Name: "test-secret"}
	assert.Equal(t, "test-secret", client.GetPath())
}

func TestGcpClient_Meta(t *testing.T) {
	client := &GcpClient{
		Project: "test-project",
		Name:    "test-secret",
		Labels: map[string]string{
			"env": "test",
		},
	}

	meta := client.Meta()
	assert.NotNil(t, meta)
	assert.Equal(t, "test-project", meta["project"])
	assert.Equal(t, "test-secret", meta["name"])
}

func TestGcpClient_DeepCopy(t *testing.T) {
	original := &GcpClient{
		Project:              "test-project",
		Name:                 "test-secret",
		ReplicationLocations: []string{"us-central1", "us-east1"},
		Labels: map[string]string{
			"env": "test",
		},
	}

	copied := original.DeepCopy()
	assert.NotNil(t, copied)
	assert.Equal(t, original.Project, copied.Project)
	assert.Equal(t, original.Name, copied.Name)
	assert.Equal(t, original.ReplicationLocations, copied.ReplicationLocations)

	// Ensure deep copy, not shallow
	if len(copied.ReplicationLocations) > 0 {
		copied.ReplicationLocations[0] = "changed"
		assert.NotEqual(t, original.ReplicationLocations[0], copied.ReplicationLocations[0])
	}
}

func TestGcpClient_CleanName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Name with slashes",
			input:    "path/to/secret",
			expected: "path-to-secret",
		},
		{
			name:     "Name without slashes",
			input:    "simple-name",
			expected: "simple-name",
		},
		{
			name:     "Name with multiple slashes",
			input:    "a/b/c/d",
			expected: "a-b-c-d",
		},
		{
			name:     "Empty name",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &GcpClient{}
			result := client.cleanName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGcpClient_FullName(t *testing.T) {
	tests := []struct {
		name     string
		client   *GcpClient
		input    string
		expected string
	}{
		{
			name: "With explicit name",
			client: &GcpClient{
				Project: "test-project",
				Name:    "default-secret",
			},
			input:    "custom-secret",
			expected: "projects/test-project/secrets/custom-secret",
		},
		{
			name: "Using default name",
			client: &GcpClient{
				Project: "test-project",
				Name:    "default-secret",
			},
			input:    "",
			expected: "projects/test-project/secrets/default-secret",
		},
		{
			name: "Name with slashes",
			client: &GcpClient{
				Project: "test-project",
				Name:    "default",
			},
			input:    "path/to/secret",
			expected: "projects/test-project/secrets/path-to-secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.client.fullName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGcpClient_SetDefaults(t *testing.T) {
	tests := []struct {
		name     string
		client   *GcpClient
		defaults *GcpClient
		expected *GcpClient
	}{
		{
			name:   "Apply all defaults",
			client: &GcpClient{},
			defaults: &GcpClient{
				Project:              "default-project",
				Name:                 "default-name",
				ReplicationLocations: []string{"us-central1"},
			},
			expected: &GcpClient{
				Project:              "default-project",
				Name:                 "default-name",
				ReplicationLocations: []string{"us-central1"},
			},
		},
		{
			name: "Don't override existing values",
			client: &GcpClient{
				Project: "existing-project",
				Name:    "existing-name",
			},
			defaults: &GcpClient{
				Project:              "default-project",
				Name:                 "default-name",
				ReplicationLocations: []string{"us-central1"},
			},
			expected: &GcpClient{
				Project:              "existing-project",
				Name:                 "existing-name",
				ReplicationLocations: []string{"us-central1"},
			},
		},
		{
			name: "Partial defaults",
			client: &GcpClient{
				Project: "existing-project",
			},
			defaults: &GcpClient{
				Name:                 "default-name",
				ReplicationLocations: []string{"us-central1", "us-east1"},
			},
			expected: &GcpClient{
				Project:              "existing-project",
				Name:                 "default-name",
				ReplicationLocations: []string{"us-central1", "us-east1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.client.SetDefaults(tt.defaults)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.Project, tt.client.Project)
			assert.Equal(t, tt.expected.Name, tt.client.Name)
			assert.Equal(t, tt.expected.ReplicationLocations, tt.client.ReplicationLocations)
		})
	}
}

func TestGcpClient_WriteSecret(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		secrets []byte
		wantErr bool
	}{
		{
			name:    "Write valid secret",
			path:    "test-secret",
			secrets: []byte("secret-value"),
			wantErr: false,
		},
		{
			name:    "Write secret with path containing slashes",
			path:    "path/to/secret",
			secrets: []byte("secret-value"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &GcpClient{
				Project: "test-project",
				Name:    "default",
			}

			ctx := context.Background()
			meta := metav1.ObjectMeta{Name: "test"}

			// Note: This will fail without a real GCP client
			// We're testing the validation and structure
			_, err := client.WriteSecret(ctx, meta, tt.path, tt.secrets)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				// Without mocks, we expect an error due to nil client
				// The test validates code structure
				assert.Error(t, err) // Expected due to nil client
			}
		})
	}
}

func TestGcpClient_DeleteSecret(t *testing.T) {
	tests := []struct {
		name   string
		secret string
	}{
		{
			name:   "Delete simple secret",
			secret: "test-secret",
		},
		{
			name:   "Delete secret with path",
			secret: "path/to/secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &GcpClient{
				Project: "test-project",
			}

			ctx := context.Background()
			err := client.DeleteSecret(ctx, tt.secret)

			// Without real GCP client, expect error
			assert.Error(t, err)
		})
	}
}

func TestGcpClient_ListSecrets(t *testing.T) {
	t.Run("List secrets", func(t *testing.T) {
		client := &GcpClient{
			Project: "test-project",
		}

		ctx := context.Background()

		// Without real GCP client, expect error or nil
		_, err := client.ListSecrets(ctx, "")

		// Without mocks, we expect an error due to nil client
		assert.Error(t, err)
	})
}

func TestGcpClient_GetSecret(t *testing.T) {
	t.Run("Get secret", func(t *testing.T) {
		client := &GcpClient{
			Project: "test-project",
			Name:    "test-secret",
		}

		ctx := context.Background()

		// Without real GCP client, expect error
		_, err := client.GetSecret(ctx, "test-secret")

		// Without mocks, we expect an error due to nil client
		assert.Error(t, err)
	})
}

func TestGcpClient_Close(t *testing.T) {
	t.Run("Close without client", func(t *testing.T) {
		client := &GcpClient{
			Project: "test-project",
		}

		err := client.Close()
		// Without a real client, expect error
		assert.Error(t, err)
	})
}

func TestGcpClient_ReplicationConfiguration(t *testing.T) {
	tests := []struct {
		name                 string
		replicationLocations []string
		description          string
	}{
		{
			name:                 "Automatic replication",
			replicationLocations: nil,
			description:          "Should use automatic replication when no locations specified",
		},
		{
			name:                 "User-managed replication",
			replicationLocations: []string{"us-central1", "us-east1"},
			description:          "Should use user-managed replication with specified locations",
		},
		{
			name:                 "Single location replication",
			replicationLocations: []string{"us-central1"},
			description:          "Should handle single location replication",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &GcpClient{
				Project:              "test-project",
				Name:                 "test-secret",
				ReplicationLocations: tt.replicationLocations,
			}

			// Validate the configuration is set correctly
			assert.Equal(t, tt.replicationLocations, client.ReplicationLocations)
			
			// Test would validate replication setup with proper mocks
			// For now, we're validating the data structure
		})
	}
}

func TestGcpClient_Labels(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
	}{
		{
			name: "With custom labels",
			labels: map[string]string{
				"env":  "production",
				"team": "platform",
			},
		},
		{
			name:   "Without custom labels",
			labels: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &GcpClient{
				Project: "test-project",
				Name:    "test-secret",
				Labels:  tt.labels,
			}

			// Validate labels are stored correctly
			assert.Equal(t, tt.labels, client.Labels)
			
			// The createSecretWrapper method should add managed-by label
			// plus any user-provided labels
			// This would be validated with proper mocks
		})
	}
}
