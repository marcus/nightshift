package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var ollamaCmd = &cobra.Command{
	Use:   "ollama",
	Short: "Manage Ollama Cloud authentication",
	Long: `Manage authentication for Ollama Cloud rate limit monitoring.

Ollama Cloud does not provide an API for usage tracking. Nightshift
scrapes the web interface at https://ollama.com/settings which requires
authentication via cookies.`,
}

var ollamaAuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Set up Ollama Cloud authentication",
	Long: `Generate a template for Ollama Cloud authentication.

This command creates a template file at ~/.ollama/cookies.txt where
you can add your ollama.com session cookies. The file format is
Netscape cookie format, which you can export from your browser using
an extension like "EditThisCookie" or "Get cookies.txt LOCALLY".`,
	RunE: runOllamaAuth,
}

func runOllamaAuth(cmd *cobra.Command, args []string) error {
	home, _ := os.UserHomeDir()
	ollamaDir := filepath.Join(home, ".ollama")

	if err := os.MkdirAll(ollamaDir, 0755); err != nil {
		return fmt.Errorf("create .ollama directory: %w", err)
	}

	cookiesPath := filepath.Join(ollamaDir, "cookies.txt")

	template := `# Ollama Cloud authentication cookies for nightshift
# 
# To populate this file:
# 1. Sign in to https://ollama.com/settings in your browser
# 2. Export cookies to Netscape format (use browser extension)
# 3. Paste cookies below, keeping only ollama.com entries
#
# Format: domain \t flag \t path \t secure \t expiration \t name \t value
#
# Example (your actual cookies will be different):
# .ollama.com	TRUE	/	TRUE	1735689600	__Secure-next-auth.session-token	eyJhbGc...
# .ollama.com	TRUE	/	TRUE	1735689600	aid	your-aid-value
# .ollama.com	TRUE	/	TRUE	1735689600	cf_clearance	your-cf-clearance-value
#
`

	if _, err := os.Stat(cookiesPath); err == nil {
		fmt.Printf("Cookie file already exists at: %s\n", cookiesPath)
		fmt.Println("Edit it to update your cookies, or delete it to regenerate.")
		return nil
	}

	if err := os.WriteFile(cookiesPath, []byte(template), 0600); err != nil {
		return fmt.Errorf("write cookie template: %w", err)
	}

	fmt.Printf("Created cookie template at: %s\n", cookiesPath)
	fmt.Println("\nNext steps:")
	fmt.Println("1. Sign in to https://ollama.com/settings")
	fmt.Println("2. Export your ollama.com cookies in Netscape format")
	fmt.Println("   Recommended extensions:")
	fmt.Println("   - Chrome/Firefox: 'EditThisCookie' or 'Get cookies.txt LOCALLY'")
	fmt.Println("3. Paste cookies into the template file")
	fmt.Println("4. Run 'nightshift budget --provider ollama' to verify")

	return nil
}

func init() {
	ollamaCmd.AddCommand(ollamaAuthCmd)
	rootCmd.AddCommand(ollamaCmd)
}