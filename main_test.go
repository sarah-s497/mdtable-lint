package main

import (
	"reflect"
	"testing"
)

func TestFilterErrors(t *testing.T) {
	in := []Finding{
		{Rule: RuleColumnCount, Severity: SeverityError},
		{Rule: RuleAlignmentConsistency, Severity: SeverityWarning},
		{Rule: RuleDelimiterRow, Severity: SeverityError},
	}
	want := []Finding{
		{Rule: RuleColumnCount, Severity: SeverityError},
		{Rule: RuleDelimiterRow, Severity: SeverityError},
	}
	if got := filterErrors(in); !reflect.DeepEqual(got, want) {
		t.Errorf("filterErrors = %#v, want %#v", got, want)
	}
}

func TestFilterErrorsNoErrors(t *testing.T) {
	in := []Finding{
		{Rule: RuleAlignmentConsistency, Severity: SeverityWarning},
	}
	if got := filterErrors(in); got != nil {
		t.Errorf("filterErrors = %#v, want nil", got)
	}
}
