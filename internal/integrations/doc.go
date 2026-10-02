// Package integrations provides readers for external configuration and task sources.
//
// It aggregates context and tasks from multiple sources — claude.md files,
// agents.md files, the td task management CLI, and GitHub Issues — and merges
// them into a unified result that can be injected into agent prompts.
//
// # Architecture
//
// Each integration implements the Reader interface (Name, Enabled, Read). A
// Manager coordinates all readers and merges their output via ReadAll.
//
// # Supported Integrations
//
//   - claude.md: Reads CLAUDE.md or claude.md from the project root for context
//   - agents.md: Reads AGENTS.md for agent behavior configuration
//   - td:        Reads tasks from the td CLI task manager
//   - github:    Reads issues labeled "nightshift" via the gh CLI
package integrations
