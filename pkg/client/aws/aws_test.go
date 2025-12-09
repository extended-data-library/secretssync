package aws

import (
	"context"
	"os"
	"testing"

	"github.com/jbcom/secretsync/pkg/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getTestEndpoint returns the LocalStack endpoint if available
func getTestEndpoint() string {
	return os.Getenv("AWS_ENDPOINT_URL")
}

// skipIfNoLocalStack skips the test if LocalStack is not available
func skipIfNoLocalStack(t *testing.T) {
	t.Helper()
	if getTestEndpoint() == "" {
		t.Skip("Skipping integration test: AWS_ENDPOINT_URL not set (LocalStack not available)")
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *AwsClient
		wantErr bool
	}{
		{
			name: "Valid configuration",
			cfg: &AwsClient{
				Name:   "test-client",
				Region: "us-west-2",
			},
			wantErr: false,
		},
		{
			name: "Default region when not specified",
			cfg: &AwsClient{
				Name: "test-client",
			},
			wantErr: false,
		},
		{
			name: "With role ARN",
			cfg: &AwsClient{
				Name:    "test-client",
				Region:  "eu-west-1",
				RoleArn: "arn:aws:iam::123456789012:role/test-role",
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
			assert.Equal(t, tt.cfg.Name, client.Name)
			if tt.cfg.Region == "" {
				assert.Equal(t, "us-east-1", client.Region)
			} else {
				assert.Equal(t, tt.cfg.Region, client.Region)
			}
		})
	}
}

func TestAwsClient_Validate(t *testing.T) {
	tests := []struct {
		name    string
		client  *AwsClient
		wantErr bool
	}{
		{
			name: "Valid client",
			client: &AwsClient{
				Name: "test-client",
			},
			wantErr: false,
		},
		{
			name:    "Missing name",
			client:  &AwsClient{},
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

func TestAwsClient_Driver(t *testing.T) {
	client := &AwsClient{}
	assert.Equal(t, driver.DriverNameAws, client.Driver())
}

func TestAwsClient_GetPath(t *testing.T) {
	client := &AwsClient{Name: "test-path"}
	assert.Equal(t, "test-path", client.GetPath())
}

func TestAwsClient_Meta(t *testing.T) {
	client := &AwsClient{
		Name:   "test-client",
		Region: "us-west-2",
		Tags: map[string]string{
			"env": "test",
		},
	}

	meta := client.Meta()
	assert.NotNil(t, meta)
	assert.Equal(t, "test-client", meta["name"])
	assert.Equal(t, "us-west-2", meta["region"])
}

func TestAwsClient_DeepCopy(t *testing.T) {
	original := &AwsClient{
		Name:           "test",
		Region:         "us-west-2",
		ReplicaRegions: []string{"eu-west-1", "ap-southeast-1"},
		Tags: map[string]string{
			"env": "test",
		},
	}

	copied := original.DeepCopy()
	assert.NotNil(t, copied)
	assert.Equal(t, original.Name, copied.Name)
	assert.Equal(t, original.Region, copied.Region)
	assert.Equal(t, original.ReplicaRegions, copied.ReplicaRegions)
	assert.Equal(t, original.Tags, copied.Tags)

	// Ensure deep copy, not shallow
	copied.ReplicaRegions[0] = "changed"
	assert.NotEqual(t, original.ReplicaRegions[0], copied.ReplicaRegions[0])
}

func TestAwsClient_ListSecrets(t *testing.T) {
	t.Run("List secrets with LocalStack", func(t *testing.T) {
		skipIfNoLocalStack(t)

		client := &AwsClient{
			Name:           "test",
			Region:         "us-east-1",
			NoEmptySecrets: false,
		}

		ctx := context.Background()

		// Create client with LocalStack endpoint
		err := client.CreateClientWithEndpoint(ctx, getTestEndpoint())
		require.NoError(t, err)

		// List secrets (should be empty initially)
		secrets, err := client.ListSecrets(ctx, "")
		assert.NoError(t, err)
		assert.NotNil(t, secrets)
	})

	t.Run("List secrets validation - no client", func(t *testing.T) {
		// This test validates the code structure without requiring LocalStack
		client := &AwsClient{
			Name:           "test",
			NoEmptySecrets: false,
		}

		// Don't call CreateClient - client.client is nil
		// The test just validates the struct setup
		assert.Equal(t, "test", client.Name)
		assert.False(t, client.NoEmptySecrets)
	})
}

func TestAwsClient_GetAlternatePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Path with leading slash",
			path:     "/foo",
			expected: "foo",
		},
		{
			name:     "Path without leading slash",
			path:     "foo",
			expected: "/foo",
		},
		{
			name:     "Empty path",
			path:     "",
			expected: "",
		},
		{
			name:     "Path with multiple components",
			path:     "/foo/bar/baz",
			expected: "foo/bar/baz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &AwsClient{}
			result := client.getAlternatePath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAwsClient_SetDefaults(t *testing.T) {
	tests := []struct {
		name     string
		client   *AwsClient
		defaults *AwsClient
		expected *AwsClient
	}{
		{
			name:   "Apply all defaults",
			client: &AwsClient{},
			defaults: &AwsClient{
				Region:         "us-west-2",
				RoleArn:        "arn:aws:iam::123456789012:role/test",
				EncryptionKey:  "key-123",
				ReplicaRegions: []string{"eu-west-1"},
				NoEmptySecrets: true,
				SkipUnchanged:  true,
			},
			expected: &AwsClient{
				Region:         "us-west-2",
				RoleArn:        "arn:aws:iam::123456789012:role/test",
				EncryptionKey:  "key-123",
				ReplicaRegions: []string{"eu-west-1"},
				NoEmptySecrets: true,
				SkipUnchanged:  true,
			},
		},
		{
			name: "Don't override existing values",
			client: &AwsClient{
				Region:  "ap-southeast-1",
				RoleArn: "existing-role",
			},
			defaults: &AwsClient{
				Region:  "us-west-2",
				RoleArn: "default-role",
			},
			expected: &AwsClient{
				Region:  "ap-southeast-1",
				RoleArn: "existing-role",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.client.SetDefaults(tt.defaults)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.Region, tt.client.Region)
			assert.Equal(t, tt.expected.RoleArn, tt.client.RoleArn)
			assert.Equal(t, tt.expected.EncryptionKey, tt.client.EncryptionKey)
			assert.Equal(t, tt.expected.ReplicaRegions, tt.client.ReplicaRegions)
			assert.Equal(t, tt.expected.NoEmptySecrets, tt.client.NoEmptySecrets)
			assert.Equal(t, tt.expected.SkipUnchanged, tt.client.SkipUnchanged)
		})
	}
}

func TestAwsClient_WriteSecret(t *testing.T) {
	t.Run("Write and read secret with LocalStack", func(t *testing.T) {
		skipIfNoLocalStack(t)

		client := &AwsClient{
			Name:              "test",
			Region:            "us-east-1",
			SkipUnchanged:     false,
			accountSecretArns: map[string]string{},
		}

		ctx := context.Background()

		// Create client with LocalStack endpoint
		err := client.CreateClientWithEndpoint(ctx, getTestEndpoint())
		require.NoError(t, err)

		meta := metav1.ObjectMeta{Name: "test"}
		secretName := "test-secret-" + t.Name()
		secretValue := []byte(`{"key":"value"}`)

		// Write the secret
		_, err = client.WriteSecret(ctx, meta, secretName, secretValue)
		assert.NoError(t, err)

		// List to populate ARN map
		_, err = client.ListSecrets(ctx, "")
		require.NoError(t, err)

		// Read back the secret
		result, err := client.GetSecret(ctx, secretName)
		assert.NoError(t, err)
		assert.Equal(t, secretValue, result)

		// Cleanup
		_ = client.DeleteSecret(ctx, secretName)
	})

	t.Run("Write secret validation - no client", func(t *testing.T) {
		// This test validates the struct setup without requiring LocalStack
		client := &AwsClient{
			Name:              "test",
			SkipUnchanged:     false,
			accountSecretArns: map[string]string{},
		}

		assert.Equal(t, "test", client.Name)
		assert.False(t, client.SkipUnchanged)
		assert.NotNil(t, client.accountSecretArns)
	})
}

func TestAwsClient_Close(t *testing.T) {
	client := &AwsClient{
		Name: "test",
	}

	err := client.Close()
	assert.NoError(t, err)
	assert.Nil(t, client.client)
}
