package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/marcus/nightshift/internal/analysis"
	"github.com/marcus/nightshift/internal/releasenotes"
	"github.com/spf13/cobra"
)

var releaseNotesCmd = &cobra.Command{
	Use:   "release-notes [path]",
	Short: "Draft release notes from git history",
	Long: `Generate release notes by analyzing git history between tags.

Commits are automatically classified using conventional commit prefixes
(feat, fix, docs, etc.) and grouped into sections. Non-conventional commits
are classified by keyword inference.

By default, generates notes for the latest tag compared to the previous tag.
Use --tag and --prev-tag to specify a custom range.

Examples:
  nightshift release-notes                     # Latest tag vs previous tag
  nightshift release-notes --tag v1.0.0        # Notes for v1.0.0
  nightshift release-notes --tag HEAD          # Unreleased changes since last tag
  nightshift release-notes --tag v2.0 --prev-tag v1.0  # Custom range
  nightshift release-notes --flat              # Flat list, no grouping
  nightshift release-notes --json              # Structured JSON output`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("path")
		if path == "" && len(args) > 0 {
			path = args[0]
		}
		if path == "" {
			var err error
			path, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}
		}

		tag, _ := cmd.Flags().GetString("tag")
		prevTag, _ := cmd.Flags().GetString("prev-tag")
		flat, _ := cmd.Flags().GetBool("flat")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		noHashes, _ := cmd.Flags().GetBool("no-hashes")
		authors, _ := cmd.Flags().GetBool("authors")

		return runReleaseNotes(path, tag, prevTag, flat, jsonOutput, noHashes, authors)
	},
}

func init() {
	releaseNotesCmd.Flags().StringP("path", "p", "", "Repository path")
	releaseNotesCmd.Flags().String("tag", "", "Tag to generate notes for (default: latest tag)")
	releaseNotesCmd.Flags().String("prev-tag", "", "Previous tag to compare against (default: auto-detect)")
	releaseNotesCmd.Flags().Bool("flat", false, "Flat list instead of grouped by category")
	releaseNotesCmd.Flags().Bool("json", false, "Output as JSON")
	releaseNotesCmd.Flags().Bool("no-hashes", false, "Omit commit hashes from output")
	releaseNotesCmd.Flags().Bool("authors", false, "Include author names")
	rootCmd.AddCommand(releaseNotesCmd)
}

func runReleaseNotes(path, tag, prevTag string, flat, jsonOutput, noHashes, authors bool) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	if !analysis.RepositoryExists(absPath) {
		return fmt.Errorf("not a git repository: %s", absPath)
	}

	gen := releasenotes.NewGenerator(absPath)

	opts := releasenotes.Options{
		Tag:                 tag,
		PrevTag:             prevTag,
		IncludeCommitHashes: !noHashes,
		IncludeAuthors:      authors,
		GroupByCategory:     !flat,
	}

	rn, err := gen.Generate(opts)
	if err != nil {
		return fmt.Errorf("generating release notes: %w", err)
	}

	if jsonOutput {
		return outputReleaseNotesJSON(rn)
	}

	fmt.Print(rn.Render(opts))
	return nil
}

type releaseNotesJSON struct {
	Version    string                           `json:"version"`
	PrevTag    string                           `json:"prev_tag,omitempty"`
	Date       string                           `json:"date"`
	Commits    int                              `json:"total_commits"`
	Categories map[string][]releaseNoteCommitJSON `json:"categories"`
}

type releaseNoteCommitJSON struct {
	Hash     string `json:"hash"`
	Subject  string `json:"subject"`
	Author   string `json:"author"`
	Date     string `json:"date"`
	Scope    string `json:"scope,omitempty"`
	Breaking bool   `json:"breaking,omitempty"`
}

func outputReleaseNotesJSON(rn *releasenotes.ReleaseNotes) error {
	cats := make(map[string][]releaseNoteCommitJSON)
	for cat, commits := range rn.Categories {
		var entries []releaseNoteCommitJSON
		for _, c := range commits {
			entries = append(entries, releaseNoteCommitJSON{
				Hash:     c.ShortHash,
				Subject:  c.Subject,
				Author:   c.Author,
				Date:     c.Date.Format("2006-01-02"),
				Scope:    c.Scope,
				Breaking: c.Breaking,
			})
		}
		cats[string(cat)] = entries
	}

	out := releaseNotesJSON{
		Version:    rn.Version,
		PrevTag:    rn.PrevTag,
		Date:       rn.Date.Format("2006-01-02"),
		Commits:    len(rn.AllCommits),
		Categories: cats,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
