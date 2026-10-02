package commit

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		want        *CommitMessage
		wantErr     bool
		errContains string
	}{
		{
			name:    "simple feat commit",
			message: "feat: add new feature",
			want: &CommitMessage{
				Type:    TypeFeat,
				Subject: "add new feature",
			},
			wantErr: false,
		},
		{
			name:    "commit with scope",
			message: "fix(api): resolve authentication bug",
			want: &CommitMessage{
				Type:    TypeFix,
				Scope:   "api",
				Subject: "resolve authentication bug",
			},
			wantErr: false,
		},
		{
			name:    "commit with body",
			message: "docs: update README\n\nAdd installation instructions and examples.",
			want: &CommitMessage{
				Type:    TypeDocs,
				Subject: "update README",
				Body:    "Add installation instructions and examples.",
			},
			wantErr: false,
		},
		{
			name:    "commit with footer",
			message: "fix(auth): patch security vulnerability\n\nFixes: CVE-2023-1234",
			want: &CommitMessage{
				Type:    TypeFix,
				Scope:   "auth",
				Subject: "patch security vulnerability",
				Footer:  "Fixes: CVE-2023-1234",
			},
			wantErr: false,
		},
		{
			name:    "breaking change with BREAKING CHANGE footer",
			message: "feat(api)!: redesign REST endpoints\n\nBREAKING CHANGE: API v1 endpoints removed",
			want: &CommitMessage{
				Type:           TypeFeat,
				Scope:          "api",
				Subject:        "redesign REST endpoints",
				Footer:         "BREAKING CHANGE: API v1 endpoints removed",
				BreakingChange: true,
			},
			wantErr: false,
		},
		{
			name:    "commit with body and footer",
			message: "refactor(core): improve performance\n\nOptimize database queries and caching.\n\nCloses: #123",
			want: &CommitMessage{
				Type:    TypeRefactor,
				Scope:   "core",
				Subject: "improve performance",
				Body:    "Optimize database queries and caching.",
				Footer:  "Closes: #123",
			},
			wantErr: false,
		},
		{
			name:        "empty message",
			message:     "",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:        "invalid format - no colon",
			message:     "feat add feature",
			wantErr:     true,
			errContains: "must follow format",
		},
		{
			name:        "invalid type",
			message:     "feature: add something",
			wantErr:     true,
			errContains: "invalid type",
		},
		{
			name:        "missing subject",
			message:     "feat: ",
			wantErr:     true,
			errContains: "must follow format",
		},
		{
			name:    "all valid types",
			message: "chore: update dependencies",
			want: &CommitMessage{
				Type:    TypeChore,
				Subject: "update dependencies",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err == nil {
					t.Error("Parse() expected error but got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Parse() error = %v, should contain %v", err, tt.errContains)
				}
				return
			}

			if got.Type != tt.want.Type {
				t.Errorf("Parse() Type = %v, want %v", got.Type, tt.want.Type)
			}
			if got.Scope != tt.want.Scope {
				t.Errorf("Parse() Scope = %v, want %v", got.Scope, tt.want.Scope)
			}
			if got.Subject != tt.want.Subject {
				t.Errorf("Parse() Subject = %v, want %v", got.Subject, tt.want.Subject)
			}
			if got.Body != tt.want.Body {
				t.Errorf("Parse() Body = %v, want %v", got.Body, tt.want.Body)
			}
			if got.Footer != tt.want.Footer {
				t.Errorf("Parse() Footer = %v, want %v", got.Footer, tt.want.Footer)
			}
			if got.BreakingChange != tt.want.BreakingChange {
				t.Errorf("Parse() BreakingChange = %v, want %v", got.BreakingChange, tt.want.BreakingChange)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
		wantErr bool
	}{
		{
			name:    "already normalized",
			message: "feat: add feature",
			want:    "feat: add feature",
			wantErr: false,
		},
		{
			name:    "with scope",
			message: "fix(api): resolve bug",
			want:    "fix(api): resolve bug",
			wantErr: false,
		},
		{
			name:    "with body preserves formatting",
			message: "docs: update guide\n\nAdd new examples.",
			want:    "docs: update guide\n\nAdd new examples.",
			wantErr: false,
		},
		{
			name:    "invalid message",
			message: "not a valid commit",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Normalize(tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("Normalize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Normalize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommitMessage_Format(t *testing.T) {
	tests := []struct {
		name string
		cm   *CommitMessage
		want string
	}{
		{
			name: "simple message",
			cm: &CommitMessage{
				Type:    TypeFeat,
				Subject: "add feature",
			},
			want: "feat: add feature",
		},
		{
			name: "with scope",
			cm: &CommitMessage{
				Type:    TypeFix,
				Scope:   "api",
				Subject: "fix bug",
			},
			want: "fix(api): fix bug",
		},
		{
			name: "with body",
			cm: &CommitMessage{
				Type:    TypeDocs,
				Subject: "update docs",
				Body:    "Add examples",
			},
			want: "docs: update docs\n\nAdd examples",
		},
		{
			name: "with footer",
			cm: &CommitMessage{
				Type:    TypeFix,
				Subject: "security patch",
				Footer:  "Fixes: #123",
			},
			want: "fix: security patch\n\nFixes: #123",
		},
		{
			name: "complete message",
			cm: &CommitMessage{
				Type:    TypeFeat,
				Scope:   "core",
				Subject: "new capability",
				Body:    "Detailed description",
				Footer:  "Closes: #456",
			},
			want: "feat(core): new capability\n\nDetailed description\n\nCloses: #456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cm.Format(); got != tt.want {
				t.Errorf("CommitMessage.Format() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommitMessage_Validate(t *testing.T) {
	tests := []struct {
		name        string
		cm          *CommitMessage
		wantErr     bool
		errContains string
	}{
		{
			name: "valid commit",
			cm: &CommitMessage{
				Type:    TypeFeat,
				Subject: "add feature",
			},
			wantErr: false,
		},
		{
			name: "valid with scope",
			cm: &CommitMessage{
				Type:    TypeFix,
				Scope:   "api",
				Subject: "resolve issue",
			},
			wantErr: false,
		},
		{
			name: "subject starts with capital",
			cm: &CommitMessage{
				Type:    TypeFeat,
				Subject: "Add feature",
			},
			wantErr:     true,
			errContains: "lowercase",
		},
		{
			name: "subject ends with period",
			cm: &CommitMessage{
				Type:    TypeFix,
				Subject: "fix bug.",
			},
			wantErr:     true,
			errContains: "period",
		},
		{
			name: "empty subject",
			cm: &CommitMessage{
				Type:    TypeDocs,
				Subject: "",
			},
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name: "subject too long",
			cm: &CommitMessage{
				Type:    TypeFeat,
				Subject: strings.Repeat("a", 100),
			},
			wantErr:     true,
			errContains: "too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cm.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CommitMessage.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" {
				if err == nil {
					t.Error("CommitMessage.Validate() expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("CommitMessage.Validate() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestIsValidType(t *testing.T) {
	tests := []struct {
		name string
		t    string
		want bool
	}{
		{"feat", "feat", true},
		{"fix", "fix", true},
		{"docs", "docs", true},
		{"style", "style", true},
		{"refactor", "refactor", true},
		{"perf", "perf", true},
		{"test", "test", true},
		{"build", "build", true},
		{"ci", "ci", true},
		{"chore", "chore", true},
		{"revert", "revert", true},
		{"invalid", "feature", false},
		{"empty", "", false},
		{"uppercase", "FEAT", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidType(tt.t); got != tt.want {
				t.Errorf("IsValidType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		maxLength int
		want      string
	}{
		{
			name:      "short text",
			text:      "short",
			maxLength: 100,
			want:      "short",
		},
		{
			name:      "text needs wrapping",
			text:      "This is a very long line that needs to be wrapped at a reasonable length",
			maxLength: 30,
			want:      "This is a very long line that\nneeds to be wrapped at a\nreasonable length",
		},
		{
			name:      "preserve paragraphs",
			text:      "First paragraph.\n\nSecond paragraph.",
			maxLength: 100,
			want:      "First paragraph.\n\nSecond paragraph.",
		},
		{
			name:      "preserve lists",
			text:      "- Item 1\n- Item 2",
			maxLength: 100,
			want:      "- Item 1\n- Item 2",
		},
		{
			name:      "empty text",
			text:      "",
			maxLength: 100,
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := wrapText(tt.text, tt.maxLength); got != tt.want {
				t.Errorf("wrapText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatSubject(t *testing.T) {
	tests := []struct {
		name       string
		commitType CommitType
		scope      string
		subject    string
		want       string
	}{
		{
			name:       "no scope",
			commitType: TypeFeat,
			scope:      "",
			subject:    "add feature",
			want:       "feat: add feature",
		},
		{
			name:       "with scope",
			commitType: TypeFix,
			scope:      "api",
			subject:    "fix bug",
			want:       "fix(api): fix bug",
		},
		{
			name:       "docs type",
			commitType: TypeDocs,
			scope:      "readme",
			subject:    "update instructions",
			want:       "docs(readme): update instructions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatSubject(tt.commitType, tt.scope, tt.subject); got != tt.want {
				t.Errorf("formatSubject() = %v, want %v", got, tt.want)
			}
		})
	}
}
