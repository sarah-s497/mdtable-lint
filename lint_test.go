package main

import (
	"reflect"
	"testing"
)

func TestSplitTableRow(t *testing.T) {
	cases := []struct {
		name         string
		line         string
		wantCells    []string
		wantLeading  bool
		wantTrailing bool
	}{
		{
			name:         "both pipes",
			line:         "| Name | Age |",
			wantCells:    []string{"Name", "Age"},
			wantLeading:  true,
			wantTrailing: true,
		},
		{
			name:         "no leading or trailing pipe",
			line:         "Name | Age",
			wantCells:    []string{"Name", "Age"},
			wantLeading:  false,
			wantTrailing: false,
		},
		{
			name:         "leading only",
			line:         "| Name | Age",
			wantCells:    []string{"Name", "Age"},
			wantLeading:  true,
			wantTrailing: false,
		},
		{
			name:         "trailing only",
			line:         "Name | Age |",
			wantCells:    []string{"Name", "Age"},
			wantLeading:  false,
			wantTrailing: true,
		},
		{
			name:         "escaped pipe stays in one cell",
			line:         `| a \| b | c |`,
			wantCells:    []string{`a \| b`, "c"},
			wantLeading:  true,
			wantTrailing: true,
		},
		{
			name:         "trailing escaped pipe is not a separator",
			line:         `| a | b \|`,
			wantCells:    []string{"a", `b \|`},
			wantLeading:  true,
			wantTrailing: false,
		},
		{
			name:         "empty cells preserved",
			line:         "| a || c |",
			wantCells:    []string{"a", "", "c"},
			wantLeading:  true,
			wantTrailing: true,
		},
		{
			name:         "blank line",
			line:         "   ",
			wantCells:    nil,
			wantLeading:  false,
			wantTrailing: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cells, leading, trailing := splitTableRow(c.line)
			if !reflect.DeepEqual(cells, c.wantCells) {
				t.Errorf("cells = %#v, want %#v", cells, c.wantCells)
			}
			if leading != c.wantLeading {
				t.Errorf("leading = %v, want %v", leading, c.wantLeading)
			}
			if trailing != c.wantTrailing {
				t.Errorf("trailing = %v, want %v", trailing, c.wantTrailing)
			}
		})
	}
}

func TestIsDelimiterRow(t *testing.T) {
	cases := []struct {
		name string
		line string
		want bool
	}{
		{"plain dashes", "|---|---|", true},
		{"left align", "|:--|:--|", true},
		{"right align", "|--:|--:|", true},
		{"center align", "|:--:|:--:|", true},
		{"single dash per cell", "|-|-|", true},
		{"mixed alignment", "|:--|--:|:--:|", true},
		{"no pipes at all", "not a delimiter row", false},
		{"header row", "| Name | Age |", false},
		{"contains a letter", "|--x--|---|", false},
		{"colon in the wrong place", "|-:-|---|", false},
		{"empty line", "", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isDelimiterRow(c.line); got != c.want {
				t.Errorf("isDelimiterRow(%q) = %v, want %v", c.line, got, c.want)
			}
		})
	}
}

func TestLintLinesRules(t *testing.T) {
	cases := []struct {
		name      string
		lines     []string
		lenient   bool
		wantRules []string
	}{
		{
			name: "clean table has no findings",
			lines: []string{
				"| Name  | Age |",
				"|-------|-----|",
				"| Alice | 30  |",
			},
			wantRules: nil,
		},
		{
			name: "delimiter row column mismatch",
			lines: []string{
				"| Name  | Age |",
				"|-------|",
				"| Alice | 30  |",
			},
			wantRules: []string{RuleDelimiterRow},
		},
		{
			name: "row with too many columns is always an error",
			lines: []string{
				"| Name  | Age |",
				"|-------|-----|",
				"| Alice | 30  | Extra |",
			},
			wantRules: []string{RuleColumnCount},
		},
		{
			name: "row with too few columns is strict-only",
			lines: []string{
				"| Name  | Age | City |",
				"|-------|-----|------|",
				"| Alice | 30",
			},
			wantRules: []string{RuleColumnCount, RulePipeStyle},
		},
		{
			name: "row with too few columns is allowed leniently",
			lines: []string{
				"| Name  | Age | City |",
				"|-------|-----|------|",
				"| Alice | 30",
			},
			lenient:   true,
			wantRules: nil,
		},
		{
			name: "pipe style mismatch is strict-only",
			lines: []string{
				"| Name  | Age |",
				"|-------|-----|",
				"Alice | 30",
			},
			wantRules: []string{RulePipeStyle},
		},
		{
			name: "pipe style mismatch allowed leniently",
			lines: []string{
				"| Name  | Age |",
				"|-------|-----|",
				"Alice | 30",
			},
			lenient:   true,
			wantRules: nil,
		},
		{
			name: "header and delimiter with no data rows",
			lines: []string{
				"| Name  | Age |",
				"|-------|-----|",
			},
			wantRules: []string{RuleEmptyTable},
		},
		{
			name: "empty table allowed leniently",
			lines: []string{
				"| Name  | Age |",
				"|-------|-----|",
			},
			lenient:   true,
			wantRules: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			findings := LintLines("test.md", c.lines, c.lenient)
			var gotRules []string
			for _, f := range findings {
				gotRules = append(gotRules, f.Rule)
			}
			if !reflect.DeepEqual(gotRules, c.wantRules) {
				t.Errorf("rules = %#v, want %#v (findings: %v)", gotRules, c.wantRules, findings)
			}
		})
	}
}

func TestCellAlignment(t *testing.T) {
	cases := []struct {
		cell string
		want alignment
	}{
		{"---", alignNone},
		{":--", alignLeft},
		{"--:", alignRight},
		{":--:", alignCenter},
		{"-", alignNone},
	}
	for _, c := range cases {
		if got := cellAlignment(c.cell); got != c.want {
			t.Errorf("cellAlignment(%q) = %v, want %v", c.cell, got, c.want)
		}
	}
}

func TestLintLinesAlignmentConsistency(t *testing.T) {
	cases := []struct {
		name      string
		lines     []string
		wantRules []string
	}{
		{
			name: "same column realigned in a later table is flagged",
			lines: []string{
				"| Name  | Age |",
				"|-------|----:|",
				"| Alice | 30  |",
				"",
				"| Name  | Age |",
				"|-------|:----|",
				"| Bob   | 41  |",
			},
			wantRules: []string{RuleAlignmentConsistency},
		},
		{
			name: "same column with matching alignment across tables is fine",
			lines: []string{
				"| Name  | Age |",
				"|-------|----:|",
				"| Alice | 30  |",
				"",
				"| Name  | Age |",
				"|-------|----:|",
				"| Bob   | 41  |",
			},
			wantRules: nil,
		},
		{
			name: "different column names never compared",
			lines: []string{
				"| Name  | Age |",
				"|-------|----:|",
				"| Alice | 30  |",
				"",
				"| Name  | Height |",
				"|-------|:-------|",
				"| Bob   | 180    |",
			},
			wantRules: nil,
		},
		{
			name: "mismatched delimiter column count is skipped, not compared",
			lines: []string{
				"| Name  | Age |",
				"|-------|----:|",
				"| Alice | 30  |",
				"",
				"| Name  | Age |",
				"|:----|",
				"| Bob | 41 |",
			},
			wantRules: []string{RuleDelimiterRow},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			findings := LintLines("test.md", c.lines, false)
			var gotRules []string
			for _, f := range findings {
				gotRules = append(gotRules, f.Rule)
			}
			if !reflect.DeepEqual(gotRules, c.wantRules) {
				t.Errorf("rules = %#v, want %#v (findings: %v)", gotRules, c.wantRules, findings)
			}
		})
	}
}

func TestIsClosingFence(t *testing.T) {
	cases := []struct {
		name      string
		line      string
		char      byte
		minLen    int
		wantClose bool
	}{
		{"exact match", "```", '`', 3, true},
		{"longer closer", "````", '`', 3, true},
		{"shorter closer is not enough", "``", '`', 3, false},
		{"trailing whitespace allowed", "```   ", '`', 3, true},
		{"trailing text is not a closer", "``` still code", '`', 3, false},
		{"up to three leading spaces allowed", "   ```", '`', 3, true},
		{"four leading spaces is not a fence", "    ```", '`', 3, false},
		{"tilde closer needs tilde", "```", '~', 3, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isClosingFence(c.line, c.char, c.minLen); got != c.wantClose {
				t.Errorf("isClosingFence(%q, %q, %d) = %v, want %v", c.line, c.char, c.minLen, got, c.wantClose)
			}
		})
	}
}

func TestLintLinesSkipsFencedCodeBlocks(t *testing.T) {
	lines := []string{
		"```",
		"| Name  | Age |",
		"|-------|",
		"| Alice | 30  | Extra |",
		"```",
		"",
		"| Name  | Age |",
		"|-------|-----|",
		"| Alice | 30  |",
	}
	findings := LintLines("test.md", lines, false)
	if len(findings) != 0 {
		t.Errorf("expected no findings for table-shaped text inside a fence, got %v", findings)
	}
}

func TestLintLinesRealTableAfterFence(t *testing.T) {
	lines := []string{
		"~~~",
		"| broken | Extra |",
		"~~~",
		"| Name  | Age |",
		"|-------|-----|",
		"| Alice | 30  |",
	}
	findings := LintLines("test.md", lines, false)
	if len(findings) != 0 {
		t.Errorf("expected no findings for a clean table after a fence, got %v", findings)
	}
}

func TestLintLinesNonTableTextIgnored(t *testing.T) {
	lines := []string{
		"This is a paragraph with a | pipe in it but no delimiter row.",
		"So is this one, still no table here.",
	}
	if got := LintLines("test.md", lines, false); got != nil {
		t.Errorf("expected no findings for non-table text, got %v", got)
	}
}
