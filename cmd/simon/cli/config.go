package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/felixgeelhaar/simon/internal/credential"
	"github.com/spf13/cobra"
)

// sensitiveKeys lists configuration keys that should be encrypted.
var sensitiveKeys = []string{
	"openai_api_key",
	"anthropic_api_key",
	"gemini_api_key",
	"api_key",
}

// isSensitiveKey checks if a configuration key should be encrypted.
func isSensitiveKey(key string) bool {
	keyLower := strings.ToLower(key)
	for _, sensitive := range sensitiveKeys {
		if keyLower == sensitive || strings.HasSuffix(keyLower, "_api_key") || strings.HasSuffix(keyLower, "_secret") {
			return true
		}
	}
	return false
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Long: `Set a configuration value. Sensitive values like API keys are automatically encrypted.

Sensitive keys (automatically encrypted):
  - Any key ending in _api_key
  - Any key ending in _secret
  - openai_api_key, anthropic_api_key, gemini_api_key`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := args[1]

		s := getStore()
		defer s.Close()

		// Encrypt sensitive values
		if isSensitiveKey(key) {
			credMgr, err := credential.NewManager()
			if err != nil {
				fmt.Printf("Failed to initialize credential manager: %v\n", err)
				os.Exit(1)
			}

			encrypted, err := credMgr.Encrypt(value)
			if err != nil {
				fmt.Printf("Failed to encrypt value: %v\n", err)
				os.Exit(1)
			}
			value = encrypted
			fmt.Printf("(value encrypted)\n")
		}

		if err := s.SetConfig(key, value); err != nil {
			fmt.Printf("Failed to set config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Configuration saved: %s\n", key)
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a configuration value",
	Long:  `Get a configuration value. Encrypted values are automatically decrypted.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]

		s := getStore()
		defer s.Close()

		val, err := s.GetConfig(key)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		if val == "" {
			fmt.Println("(not set)")
			return
		}

		// Decrypt if encrypted
		if credential.IsEncrypted(val) {
			credMgr, err := credential.NewManager()
			if err != nil {
				fmt.Printf("Failed to initialize credential manager: %v\n", err)
				os.Exit(1)
			}

			decrypted, err := credMgr.Decrypt(val)
			if err != nil {
				fmt.Printf("Failed to decrypt value: %v\n", err)
				os.Exit(1)
			}

			// For sensitive keys, mask the output
			if isSensitiveKey(key) {
				fmt.Printf("%s (encrypted, masked)\n", credential.MaskSecret(decrypted))
			} else {
				fmt.Println(decrypted)
			}
			return
		}

		// For unencrypted sensitive keys, show a warning
		if isSensitiveKey(key) {
			fmt.Printf("%s (WARNING: not encrypted, re-save to encrypt)\n", credential.MaskSecret(val))
			return
		}

		fmt.Println(val)
	},
}

func init() {
	RootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
}
