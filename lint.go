package main

import (
	"fmt"
	"regexp"
	"strings"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

const (
	RuleDelimiterRow = "delimiter-row"
	RuleColumnCount  = "column-count"
	RuleEmptyTable   = "empty-table"
	RulePipeStyle    = "pipe-style"
)

// Finding is one lint result tied to a specific line in a specific file.
type Finding struct {
	File     string
	Line     int
	Rule     string
	Severity Severity
	Message  string
}

func (f Finding) String() string {
	return fmt.Sprintf("%s:%d: %s: [%s] %s", f.File, f.Line, f.Severity, f.Rule, f.Message)
}

var delimiterCellPattern = regexp.MustCompile(`^:?-+:?$`)

// splitTableRow splits a table row into cells, respecting backslash-escaped
// pipes so that "a \| b" stays one cell instead of two.
func splitTableRow(line string) (cells []string, leading, trailing bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil, false, false
	}

	leading = strings.HasPrefix(trimmed, "|")
	trailing = endsWithUnescapedPipe(trimmed)

	runes := []rune(trimmed)
	var cur strings.Builder
	for i := 0; i < len(runes); i++ {
		if runes[i] == '\\' && i+1 < len(runes) && runes[i+1] == '|' {
			cur.WriteRune('\\')
			cur.WriteRune('|')
			i++
			continue
		}
		if runes[i] == '|' {
			cells = append(cells, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteRune(runes[i])
	}
	cells = append(cells, cur.String())

	if leading && len(cells) > 0 {
		cells = cells[1:]
	}
	if trailing && len(cells) > 0 {
		cells = cells[:len(cells)-1]
	}
	for i, c := range cells {
		cells[i] = strings.TrimSpace(c)
	}
	return cells, leading, trailing
}

// endsWithUnescapedPipe reports whether s ends in a pipe that is a real
// column separator rather than a "\|" escape sequence.
func endsWithUnescapedPipe(s string) bool {
	if !strings.HasSuffix(s, "|") {
		return false
	}
	backslashes := 0
	for i := len(s) - 2; i >= 0 && s[i] == '\\'; i-- {
		backslashes++
	}
	return backslashes%2 == 0
}

func isDelimiterRow(line string) bool {
	cells, _, _ := splitTableRow(line)
	if len(cells) == 0 {
		return false
	}
	for _, c := range cells {
		if !delimiterCellPattern.MatchString(c) {
			return false
		}
	}
	return true
}

func looksLikeRow(line string) bool {
	t := strings.TrimSpace(line)
	return t != "" && strings.ContainsRune(t, '|')
}

type row struct {
	line             int
	cells            []string
	leading, trailing bool
}

type table struct {
	header   row
	delim    row
	dataRows []row
}

// findTables scans lines for GFM-style tables: a row containing a pipe,
// immediately followed by a valid delimiter row. It does not yet understand
// fenced code blocks, so a table-like block inside ``` fences will be
// misdetected; see the README roadmap.
func findTables(lines []string) []table {
	var tables []table
	i := 0
	for i < len(lines)-1 {
		if !looksLikeRow(lines[i]) || !isDelimiterRow(lines[i+1]) {
			i++
			continue
		}

		hCells, hLead, hTrail := splitTableRow(lines[i])
		dCells, dLead, dTrail := splitTableRow(lines[i+1])
		t := table{
			header: row{line: i + 1, cells: hCells, leading: hLead, trailing: hTrail},
			delim:  row{line: i + 2, cells: dCells, leading: dLead, trailing: dTrail},
		}

		j := i + 2
		for j < len(lines) && looksLikeRow(lines[j]) {
			cells, lead, trail := splitTableRow(lines[j])
			t.dataRows = append(t.dataRows, row{line: j + 1, cells: cells, leading: lead, trailing: trail})
			j++
		}

		tables = append(tables, t)
		i = j
	}
	return tables
}

// lintTable checks one table. In strict mode (the default) every row must
// match the header's column count and pipe style exactly. --lenient falls
// back to what the GFM spec actually permits: rows with fewer cells than the
// header are legal (missing cells are treated as empty), so those stop being
// errors. Rows with more cells than the header stay errors in both modes,
// since GFM silently discards the excess cells and that's a real way to lose
// data unnoticed.
func lintTable(filename string, t table, lenient bool) []Finding {
	var findings []Finding
	headerCount := len(t.header.cells)

	if len(t.delim.cells) != headerCount {
		findings = append(findings, Finding{
			File: filename, Line: t.delim.line, Rule: RuleDelimiterRow, Severity: SeverityError,
			Message: fmt.Sprintf("delimiter row has %d column(s), header has %d", len(t.delim.cells), headerCount),
		})
	}

	if len(t.dataRows) == 0 && !lenient {
		findings = append(findings, Finding{
			File: filename, Line: t.header.line, Rule: RuleEmptyTable, Severity: SeverityError,
			Message: "table has a header and delimiter row but no data rows",
		})
	}

	for _, r := range t.dataRows {
		n := len(r.cells)
		switch {
		case n > headerCount:
			findings = append(findings, Finding{
				File: filename, Line: r.line, Rule: RuleColumnCount, Severity: SeverityError,
				Message: fmt.Sprintf("row has %d column(s), header has %d; extra cells are silently dropped by GFM renderers", n, headerCount),
			})
		case n < headerCount && !lenient:
			findings = append(findings, Finding{
				File: filename, Line: r.line, Rule: RuleColumnCount, Severity: SeverityError,
				Message: fmt.Sprintf("row has %d column(s), header has %d", n, headerCount),
			})
		}

		if !lenient && (r.leading != t.header.leading || r.trailing != t.header.trailing) {
			findings = append(findings, Finding{
				File: filename, Line: r.line, Rule: RulePipeStyle, Severity: SeverityError,
				Message: "row's leading/trailing pipe style does not match the header row",
			})
		}
	}

	return findings
}

// LintLines runs every rule against every table found in lines.
func LintLines(filename string, lines []string, lenient bool) []Finding {
	var findings []Finding
	for _, t := range findTables(lines) {
		findings = append(findings, lintTable(filename, t, lenient)...)
	}
	return findings
}
