package commands

import (
	"testing"
)

func TestChangelogCmd_Flags(t *testing.T) {
	// Verify all expected flags are registered
	flags := []string{"from", "to", "version", "format", "output"}
	for _, name := range flags {
		f := changelogCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("flag %q not registered", name)
		}
	}
}

func TestChangelogCmd_FlagDefaults(t *testing.T) {
	tests := []struct {
		flag string
		want string
	}{
		{"from", ""},
		{"to", "HEAD"},
		{"version", ""},
		{"format", "markdown"},
		{"output", ""},
	}
	for _, tc := range tests {
		f := changelogCmd.Flags().Lookup(tc.flag)
		if f == nil {
			t.Fatalf("flag %q not found", tc.flag)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag %q default = %q, want %q", tc.flag, f.DefValue, tc.want)
		}
	}
}

func TestChangelogCmd_InvalidFormat(t *testing.T) {
	cmd := changelogCmd
	_ = cmd.Flags().Set("from", "v0.1.0")
	_ = cmd.Flags().Set("format", "html")
	defer func() {
		_ = cmd.Flags().Set("from", "")
		_ = cmd.Flags().Set("format", "markdown")
	}()

	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatal("expected error for invalid format, got nil")
	}
}

func TestChangelogCmd_Registration(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "changelog" {
			found = true
			break
		}
	}
	if !found {
		t.Error("changelog command not registered on root")
	}
}
