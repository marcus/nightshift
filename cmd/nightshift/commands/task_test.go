package commands

import (
	"testing"

	"github.com/marcus/nightshift/internal/tasks"
)

func TestParseCategoryFilter(t *testing.T) {
	tests := []struct {
		input string
		want  tasks.TaskCategory
		err   bool
	}{
		{"pr", tasks.CategoryPR, false},
		{"PR", tasks.CategoryPR, false},
		{"analysis", tasks.CategoryAnalysis, false},
		{"options", tasks.CategoryOptions, false},
		{"safe", tasks.CategorySafe, false},
		{"map", tasks.CategoryMap, false},
		{"emergency", tasks.CategoryEmergency, false},
		{"bogus", 0, true},
	}

	for _, tt := range tests {
		got, err := parseCategoryFilter(tt.input)
		if tt.err {
			if err == nil {
				t.Errorf("parseCategoryFilter(%q): want error, got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseCategoryFilter(%q): %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseCategoryFilter(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseCostFilter(t *testing.T) {
	tests := []struct {
		input string
		want  tasks.CostTier
		err   bool
	}{
		{"low", tasks.CostLow, false},
		{"LOW", tasks.CostLow, false},
		{"medium", tasks.CostMedium, false},
		{"high", tasks.CostHigh, false},
		{"veryhigh", tasks.CostVeryHigh, false},
		{"extreme", 0, true},
	}

	for _, tt := range tests {
		got, err := parseCostFilter(tt.input)
		if tt.err {
			if err == nil {
				t.Errorf("parseCostFilter(%q): want error, got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseCostFilter(%q): %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseCostFilter(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestFilterByCategory(t *testing.T) {
	defs := tasks.AllDefinitionsSorted()
	pr := filterByCategory(defs, tasks.CategoryPR)
	if len(pr) == 0 {
		t.Fatal("expected PR tasks, got none")
	}
	for _, d := range pr {
		if d.Category != tasks.CategoryPR {
			t.Errorf("got category %v, want PR", d.Category)
		}
	}
}

func TestFilterByCost(t *testing.T) {
	defs := tasks.AllDefinitionsSorted()
	low := filterByCost(defs, tasks.CostLow)
	if len(low) == 0 {
		t.Fatal("expected low-cost tasks, got none")
	}
	for _, d := range low {
		if d.CostTier != tasks.CostLow {
			t.Errorf("got cost %v, want Low", d.CostTier)
		}
	}
}

func TestFormatK(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{500, "500"},
		{1000, "1k"},
		{10000, "10k"},
		{50000, "50k"},
		{150000, "150k"},
		{500000, "500k"},
		{1000000, "1M"},
	}

	for _, tt := range tests {
		got := formatK(tt.input)
		if got != tt.want {
			t.Errorf("formatK(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCategoryShort(t *testing.T) {
	tests := []struct {
		input tasks.TaskCategory
		want  string
	}{
		{tasks.CategoryPR, "PR"},
		{tasks.CategoryAnalysis, "Analysis"},
		{tasks.CategoryOptions, "Options"},
		{tasks.CategorySafe, "Safe"},
		{tasks.CategoryMap, "Map"},
		{tasks.CategoryEmergency, "Emergency"},
	}

	for _, tt := range tests {
		got := categoryShort(tt.input)
		if got != tt.want {
			t.Errorf("categoryShort(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTaskInstanceFromDef(t *testing.T) {
	def := tasks.TaskDefinition{
		Type:        "lint-fix",
		Name:        "Linter Fixes",
		Description: "Fix lint errors",
	}

	// Without project path
	task := taskInstanceFromDef(def, "")
	if task.ID != "lint-fix" {
		t.Errorf("ID = %q, want %q", task.ID, "lint-fix")
	}
	if task.Title != "Linter Fixes" {
		t.Errorf("Title = %q", task.Title)
	}

	// With project path
	task = taskInstanceFromDef(def, "/tmp/proj")
	if task.ID != "lint-fix:/tmp/proj" {
		t.Errorf("ID = %q, want %q", task.ID, "lint-fix:/tmp/proj")
	}
}

func TestAgentByNameUnknown(t *testing.T) {
	_, err := agentByName("unknown-provider")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}
