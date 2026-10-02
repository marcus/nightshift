package integrations

import "testing"

var sampleClaudeMD = `# Project

## Conventions
- Use Go standard library where possible
- Run tests before committing
- Keep functions under 50 lines

## Tasks
- Refactor database layer
- Add integration tests

## Constraints
- No external HTTP calls in unit tests
- Must support Go 1.21+
`

var sampleAgentsMD = `# Agent Configuration

## Allowed Actions
- Read any file in the repository
- Run go test and go build
- Create git branches

## Forbidden Actions
- Push to main directly
- Delete production databases
- Modify CI/CD pipelines without review

## Tool Restrictions
- No shell access outside project directory
- File writes limited to src/ and test/
`

func BenchmarkParseClaudeMD(b *testing.B) {
	for b.Loop() {
		_ = parseClaudeMD(sampleClaudeMD)
	}
}

func BenchmarkParseAgentsMD(b *testing.B) {
	for b.Loop() {
		_ = parseAgentsMD(sampleAgentsMD)
	}
}
