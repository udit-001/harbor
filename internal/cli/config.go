package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/udit-001/harbor/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and manage harbor configuration",
	Args:  cobra.NoArgs,
	RunE:  runShowHelp,
	Long: `View and update harbor configuration.

The config file (harbor.toml) lives in your platform app config
directory (~/.config/harbor/ on Linux) and points to your data
directory where the database and workspaces live.

Examples:
  harbor config read             # Show current config
  harbor config set data_dir ~/my-harbor  # Change data directory`,
}

var configReadCmd = &cobra.Command{
	Use:   "read",
	Short: "Read current configuration",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("config error: %w", err)
		}

		dataDir := config.DefaultDataDir()
		if cfg != nil && cfg.DataDir != "" {
			dataDir = cfg.DataDir
		}
		dbPath := filepath.Join(dataDir, "harbor.db")

		port := config.DefaultPort
		portLabel := "9090 (default)"
		if cfg != nil && cfg.Port != 0 {
			port = cfg.Port
			portLabel = fmt.Sprintf("%d", port)
		}

		fmt.Println()
		fmt.Printf("  Config file:   %s\n", config.Path())
		fmt.Printf("  data_dir:      %s\n", dataDir)
		fmt.Printf("  Database:      %s\n", dbPath)
		fmt.Printf("  port:          %s\n", portLabel)
		fmt.Println()
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Update a configuration key or DB-backed setting.

Supported keys:
  data_dir            Path to the harbor data directory
  port                HTTP server port for the web dashboard (1-65535)

TOML keys are saved to the config file. Run 'harbor config read' to verify.

Examples:
  harbor config set data_dir ~/my-harbor
  harbor config set port 8080`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("config error: %w", err)
		}
		if cfg == nil {
			return fmt.Errorf("no config found — run 'harbor init' first")
		}

		switch key {
		case "data_dir":
			cfg.DataDir = value
		case "port":
			p, err := strconv.Atoi(value)
			if err != nil || p < 1 || p > 65535 {
				return fmt.Errorf("invalid value %q for %s: use a port number 1-65535", value, key)
			}
			cfg.Port = p
		default:
			return fmt.Errorf("unknown config key: %s", key)
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		// Ensure the new data directory exists
		if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
			return fmt.Errorf("create data directory: %w", err)
		}

		fmt.Println()
		fmt.Printf("  ✓ %s set to %s\n", key, value)
		fmt.Printf("    Config: %s\n", config.Path())
		if key == "port" {
			fmt.Println("    Note: if you've pinned Pharos, re-create the shortcut so it points at the new port.")
		}
		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configReadCmd)
	configCmd.AddCommand(configSetCmd)
}
