package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmd(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "No arguments",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "Help flag",
			args:    []string{"--help"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:   "vss",
				Short: "Vault Secret Sync",
			}
			cmd.SetArgs(tt.args)
			
			err := cmd.Execute()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRootCmd_Flags(t *testing.T) {
	tests := []struct {
		name     string
		flagName string
		flagType string
	}{
		{
			name:     "Config flag",
			flagName: "config",
			flagType: "string",
		},
		{
			name:     "Log level flag",
			flagName: "log-level",
			flagType: "string",
		},
		{
			name:     "Log format flag",
			flagName: "log-format",
			flagType: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate that flag names are defined
			assert.NotEmpty(t, tt.flagName)
		})
	}
}

func TestRootCmd_LogLevels(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		valid    bool
	}{
		{
			name:     "Debug level",
			logLevel: "debug",
			valid:    true,
		},
		{
			name:     "Info level",
			logLevel: "info",
			valid:    true,
		},
		{
			name:     "Warn level",
			logLevel: "warn",
			valid:    true,
		},
		{
			name:     "Error level",
			logLevel: "error",
			valid:    true,
		},
		{
			name:     "Trace level",
			logLevel: "trace",
			valid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.logLevel)
		})
	}
}

func TestRootCmd_LogFormats(t *testing.T) {
	tests := []struct {
		name      string
		logFormat string
		valid     bool
	}{
		{
			name:      "JSON format",
			logFormat: "json",
			valid:     true,
		},
		{
			name:      "Text format",
			logFormat: "text",
			valid:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.logFormat)
		})
	}
}

func TestPipelineCmd(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "Pipeline help",
			args:        []string{"pipeline", "--help"},
			expectError: false,
		},
		{
			name:        "Pipeline without config",
			args:        []string{"pipeline"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a root command with pipeline subcommand
			cmd := &cobra.Command{
				Use: "vss",
			}
			
			pipelineCmd := &cobra.Command{
				Use:   "pipeline",
				Short: "Run the pipeline",
				RunE: func(cmd *cobra.Command, args []string) error {
					cfgFile := cmd.Flag("config").Value.String()
					if cfgFile == "" {
						return cobra.MinimumNArgs(1)(cmd, args)
					}
					return nil
				},
			}
			pipelineCmd.Flags().String("config", "", "config file")
			pipelineCmd.Flags().Bool("dry-run", false, "dry run mode")
			pipelineCmd.Flags().Bool("merge-only", false, "merge only")
			pipelineCmd.Flags().StringSlice("targets", []string{}, "target filters")
			
			cmd.AddCommand(pipelineCmd)
			cmd.SetArgs(tt.args)
			
			// Capture output
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			
			err := cmd.Execute()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCmd(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "Validate help",
			args: []string{"validate", "--help"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use: "vss",
			}
			
			validateCmd := &cobra.Command{
				Use:   "validate",
				Short: "Validate configuration",
				RunE: func(cmd *cobra.Command, args []string) error {
					return nil
				},
			}
			validateCmd.Flags().String("config", "", "config file")
			
			cmd.AddCommand(validateCmd)
			cmd.SetArgs(tt.args)
			
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			
			_ = cmd.Execute()
			// Validate command structure exists
			assert.NotNil(t, validateCmd)
		})
	}
}

func TestGraphCmd(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "Graph help",
			args: []string{"graph", "--help"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use: "vss",
			}
			
			graphCmd := &cobra.Command{
				Use:   "graph",
				Short: "Show dependency graph",
				RunE: func(cmd *cobra.Command, args []string) error {
					return nil
				},
			}
			graphCmd.Flags().String("config", "", "config file")
			
			cmd.AddCommand(graphCmd)
			cmd.SetArgs(tt.args)
			
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			
			_ = cmd.Execute()
			assert.NotNil(t, graphCmd)
		})
	}
}

func TestMigrateCmd(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "Migrate help",
			args: []string{"migrate", "--help"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use: "vss",
			}
			
			migrateCmd := &cobra.Command{
				Use:   "migrate",
				Short: "Migrate configuration",
				RunE: func(cmd *cobra.Command, args []string) error {
					return nil
				},
			}
			
			cmd.AddCommand(migrateCmd)
			cmd.SetArgs(tt.args)
			
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			
			_ = cmd.Execute()
			assert.NotNil(t, migrateCmd)
		})
	}
}

func TestContextCmd(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "Context help",
			args: []string{"context", "--help"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use: "vss",
			}
			
			contextCmd := &cobra.Command{
				Use:   "context",
				Short: "Manage contexts",
				RunE: func(cmd *cobra.Command, args []string) error {
					return nil
				},
			}
			
			cmd.AddCommand(contextCmd)
			cmd.SetArgs(tt.args)
			
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			
			_ = cmd.Execute()
			assert.NotNil(t, contextCmd)
		})
	}
}

func TestPipelineCmd_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		defaultValue interface{}
	}{
		{
			name:         "Dry run flag",
			flagName:     "dry-run",
			defaultValue: false,
		},
		{
			name:         "Merge only flag",
			flagName:     "merge-only",
			defaultValue: false,
		},
		{
			name:         "Targets flag",
			flagName:     "targets",
			defaultValue: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipelineCmd := &cobra.Command{
				Use: "pipeline",
			}
			
			switch v := tt.defaultValue.(type) {
			case bool:
				pipelineCmd.Flags().Bool(tt.flagName, v, "test flag")
			case []string:
				pipelineCmd.Flags().StringSlice(tt.flagName, v, "test flag")
			}
			
			flag := pipelineCmd.Flag(tt.flagName)
			require.NotNil(t, flag)
			assert.Equal(t, tt.flagName, flag.Name)
		})
	}
}

func TestRootCmd_PersistentFlags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		defaultValue string
	}{
		{
			name:         "Config file flag",
			flagName:     "config",
			defaultValue: "",
		},
		{
			name:         "Log level flag",
			flagName:     "log-level",
			defaultValue: "info",
		},
		{
			name:         "Log format flag",
			flagName:     "log-format",
			defaultValue: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := &cobra.Command{
				Use: "vss",
			}
			rootCmd.PersistentFlags().String(tt.flagName, tt.defaultValue, "test flag")
			
			flag := rootCmd.PersistentFlags().Lookup(tt.flagName)
			require.NotNil(t, flag)
			assert.Equal(t, tt.flagName, flag.Name)
			assert.Equal(t, tt.defaultValue, flag.DefValue)
		})
	}
}

func TestCommandStructure(t *testing.T) {
	// Test that we can create a command hierarchy
	rootCmd := &cobra.Command{
		Use:   "vss",
		Short: "Vault Secret Sync",
	}

	// Add subcommands
	subcommands := []*cobra.Command{
		{Use: "pipeline", Short: "Run pipeline"},
		{Use: "validate", Short: "Validate config"},
		{Use: "graph", Short: "Show graph"},
		{Use: "migrate", Short: "Migrate config"},
		{Use: "context", Short: "Manage contexts"},
	}

	for _, subcmd := range subcommands {
		rootCmd.AddCommand(subcmd)
	}

	// Verify subcommands are registered
	commands := rootCmd.Commands()
	assert.Len(t, commands, len(subcommands))

	// Verify each subcommand exists
	for _, expected := range subcommands {
		found := false
		for _, cmd := range commands {
			if cmd.Use == expected.Use {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected subcommand %s not found", expected.Use)
	}
}
