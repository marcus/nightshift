// config.go implements the config command for viewing and modifying configuration.

package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/marcus/nightshift/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long: `View and modify nightshift configuration.

Shows current configuration merged from global and project configs.
Use subcommands to get/set specific values or validate the config.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigShow()
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get KEY",
	Short: "Get configuration value",
	Long: `Get a specific configuration value by key path.

Examples:
  nightshift config get budget.max_percent
  nightshift config get providers.claude.enabled
  nightshift config get logging.level`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigGet(args[0])
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set KEY VALUE",
	Short: "Set configuration value",
	Long: `Set a configuration value by key path.

Writes to the project config if it exists, otherwise to global config.
Use --global to always write to global config.

Examples:
  nightshift config set budget.max_percent 15
  nightshift config set logging.level debug
  nightshift config set providers.claude.enabled false`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		global, _ := cmd.Flags().GetBool("global")
		return runConfigSet(args[0], args[1], global)
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	Long: `Validate the current configuration.

Checks both global and project configs for errors.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigValidate()
	},
}

func init() {
	configSetCmd.Flags().BoolP("global", "g", false, "Write to global config instead of project config")
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configValidateCmd)
	rootCmd.AddCommand(configCmd)
}

// runConfigShow displays the current merged configuration.
func runConfigShow() error {
	// Show config source paths
	globalPath := config.GlobalConfigPath()
	projectPath := findProjectConfigPath()

	fmt.Println("Configuration Sources")
	fmt.Println("=====================")
	fmt.Printf("Global:  %s", globalPath)
	if fileExists(globalPath) {
		fmt.Println(" (exists)")
	} else {
		fmt.Println(" (not found)")
	}
	fmt.Printf("Project: %s", projectPath)
	if fileExists(projectPath) {
		fmt.Println(" (exists)")
	} else {
		fmt.Println(" (not found)")
	}
	fmt.Println()

	// Load and display merged config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	fmt.Println("Current Configuration")
	fmt.Println("=====================")
	printConfigYAML(cfg)

	return nil
}

// runConfigGet retrieves a specific config value by key path.
func runConfigGet(key string) error {
	v := viper.New()

	// Load configs into viper
	if err := loadViperConfig(v); err != nil {
		return err
	}

	value := v.Get(key)
	if value == nil {
		return fmt.Errorf("key not found: %s", key)
	}

	// Format output based on type
	switch val := value.(type) {
	case map[string]interface{}:
		printMap(val, 0)
	case []interface{}:
		printSlice(val, 0)
	default:
		fmt.Println(value)
	}

	return nil
}

// runConfigSet sets a config value and writes it back.
func runConfigSet(key, value string, useGlobal bool) error {
	// Determine which config file to write to
	var configPath string
	if useGlobal {
		configPath = config.GlobalConfigPath()
	} else {
		projectPath := findProjectConfigPath()
		if fileExists(projectPath) {
			configPath = projectPath
		} else {
			configPath = config.GlobalConfigPath()
		}
	}

	// Ensure directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Load existing config or create new viper instance
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Read existing config if it exists
	if fileExists(configPath) {
		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("reading config: %w", err)
		}
	}

	// Parse and set the value
	parsedValue := parseValue(value)
	v.Set(key, parsedValue)

	// Write back to file
	if err := v.WriteConfig(); err != nil {
		// If file doesn't exist, use SafeWriteConfig
		if os.IsNotExist(err) {
			if err := v.SafeWriteConfig(); err != nil {
				return fmt.Errorf("writing config: %w", err)
			}
		} else {
			return fmt.Errorf("writing config: %w", err)
		}
	}

	fmt.Printf("Set %s = %v in %s\n", key, parsedValue, configPath)

	// Validate the new config
	if _, err := config.Load(); err != nil {
		fmt.Printf("Warning: config validation failed: %v\n", err)
	}

	return nil
}

// runConfigValidate validates the configuration files.
func runConfigValidate() error {
	fmt.Println("Validating configuration...")
	fmt.Println()

	globalPath := config.GlobalConfigPath()
	projectPath := findProjectConfigPath()

	hasErrors := false

	// Validate global config if it exists
	if fileExists(globalPath) {
		fmt.Printf("Global config: %s\n", globalPath)
		if err := validateConfigFile(globalPath); err != nil {
			fmt.Printf("  Error: %v\n", err)
			hasErrors = true
		} else {
			fmt.Println("  Valid")
		}
		fmt.Println()
	}

	// Validate project config if it exists
	if fileExists(projectPath) {
		fmt.Printf("Project config: %s\n", projectPath)
		if err := validateConfigFile(projectPath); err != nil {
			fmt.Printf("  Error: %v\n", err)
			hasErrors = true
		} else {
			fmt.Println("  Valid")
		}
		fmt.Println()
	}

	// Validate merged config
	fmt.Println("Merged configuration:")
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		hasErrors = true
	} else {
		if err := config.Validate(cfg); err != nil {
			fmt.Printf("  Error: %v\n", err)
			hasErrors = true
		} else {
			fmt.Println("  Valid")
		}
	}

	if hasErrors {
		return fmt.Errorf("configuration has errors")
	}

	fmt.Println()
	fmt.Println("All configurations are valid.")
	return nil
}

// Helper functions

func findProjectConfigPath() string {
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, config.ProjectConfigName)
}

func fileExists(path string) bool {
	path = expandPath(path)
	_, err := os.Stat(path)
	return err == nil
}

func loadViperConfig(v *viper.Viper) error {
	// Load global config
	globalPath := expandPath(config.GlobalConfigPath())
	if fileExists(globalPath) {
		v.SetConfigFile(globalPath)
		v.SetConfigType("yaml")
		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("reading global config: %w", err)
		}
	}

	// Merge project config
	projectPath := findProjectConfigPath()
	if fileExists(projectPath) {
		v.SetConfigFile(projectPath)
		if err := v.MergeInConfig(); err != nil {
			return fmt.Errorf("merging project config: %w", err)
		}
	}

	// Bind environment variables
	v.SetEnvPrefix("NIGHTSHIFT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	return nil
}

func validateConfigFile(path string) error {
	v := viper.New()
	v.SetConfigFile(expandPath(path))
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	var cfg config.Config
	if err := v.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("parsing: %w", err)
	}

	return config.Validate(&cfg)
}

func parseValue(value string) interface{} {
	// Try to parse as bool
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}

	// Try to parse as int
	if i, err := strconv.Atoi(value); err == nil {
		return i
	}

	// Try to parse as float
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		// Only return float if it has a decimal point
		if strings.Contains(value, ".") {
			return f
		}
	}

	// Return as string
	return value
}

func printConfigYAML(cfg *config.Config) {
	// Use reflection to print config as YAML-like format
	printStruct(reflect.ValueOf(cfg).Elem(), 0)
}

func printStruct(v reflect.Value, indent int) {
	t := v.Type()
	prefix := strings.Repeat("  ", indent)

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		// Get the mapstructure tag for the field name
		tag := field.Tag.Get("mapstructure")
		if tag == "" {
			tag = strings.ToLower(field.Name)
		}

		// Skip empty/zero values for cleaner output
		if isZero(value) {
			continue
		}

		switch value.Kind() {
		case reflect.Struct:
			fmt.Printf("%s%s:\n", prefix, tag)
			printStruct(value, indent+1)
		case reflect.Ptr:
			if !value.IsNil() {
				if value.Elem().Kind() == reflect.Struct {
					fmt.Printf("%s%s:\n", prefix, tag)
					printStruct(value.Elem(), indent+1)
				} else {
					fmt.Printf("%s%s: %v\n", prefix, tag, value.Elem().Interface())
				}
			}
		case reflect.Slice:
			if value.Len() > 0 {
				fmt.Printf("%s%s:\n", prefix, tag)
				for j := 0; j < value.Len(); j++ {
					elem := value.Index(j)
					if elem.Kind() == reflect.Struct {
						fmt.Printf("%s  -\n", prefix)
						printStruct(elem, indent+2)
					} else {
						fmt.Printf("%s  - %v\n", prefix, elem.Interface())
					}
				}
			}
		case reflect.Map:
			if value.Len() > 0 {
				fmt.Printf("%s%s:\n", prefix, tag)
				for _, key := range value.MapKeys() {
					mapVal := value.MapIndex(key)
					fmt.Printf("%s  %v: %v\n", prefix, key.Interface(), mapVal.Interface())
				}
			}
		default:
			fmt.Printf("%s%s: %v\n", prefix, tag, value.Interface())
		}
	}
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	case reflect.Struct:
		// Check if all fields are zero
		for i := 0; i < v.NumField(); i++ {
			if !isZero(v.Field(i)) {
				return false
			}
		}
		return true
	default:
		return v.IsZero()
	}
}

func printMap(m map[string]interface{}, indent int) {
	prefix := strings.Repeat("  ", indent)
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			fmt.Printf("%s%s:\n", prefix, k)
			printMap(val, indent+1)
		case []interface{}:
			fmt.Printf("%s%s:\n", prefix, k)
			printSlice(val, indent+1)
		default:
			fmt.Printf("%s%s: %v\n", prefix, k, v)
		}
	}
}

func printSlice(s []interface{}, indent int) {
	prefix := strings.Repeat("  ", indent)
	for _, v := range s {
		switch val := v.(type) {
		case map[string]interface{}:
			fmt.Printf("%s-\n", prefix)
			printMap(val, indent+1)
		default:
			fmt.Printf("%s- %v\n", prefix, v)
		}
	}
}
