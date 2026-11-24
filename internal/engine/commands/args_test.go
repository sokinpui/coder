package commands

import (
	"reflect"
	"testing"
)

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"   ", nil},
		{"foo bar baz", []string{"foo", "bar", "baz"}},
		{`"foo bar" baz`, []string{"foo bar", "baz"}},
		{`'foo bar' 'baz qux'`, []string{"foo bar", "baz qux"}},
		{`foo\ bar baz`, []string{"foo bar", "baz"}},
		{`"foo \"bar\""`, []string{`foo "bar"`}},
	}

	for _, tt := range tests {
		result := SplitArgs(tt.input)
		if !reflect.DeepEqual(result, tt.expected) {
			t.Errorf("SplitArgs(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestFormatArg(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", `""`},
		{"simple", "simple"},
		{"path with spaces", `"path with spaces"`},
		{`quote"inside`, `"quote\"inside"`},
	}

	for _, tt := range tests {
		result := FormatArg(tt.input)
		if result != tt.expected {
			t.Errorf("FormatArg(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestActiveCommandArg(t *testing.T) {
	tests := []struct {
		input              string
		expectedCmd        string
		expectedArgPrefix  string
		expectedReplPrefix string
		expectedOk         bool
	}{
		{"@path/to", "file", "path/to", "@ ", true},
		{"@ ", "file", "", "@ ", true},
		{"/@path/to", "file", "path/to", "/@ ", true},
		{"!cmd", "shell", "cmd", "! ", true},
		{"/file foo ", "file", "", "/file foo ", true},
		{"/file foo bar", "file", "bar", "/file foo ", true},
		{"/file \"foo bar\"", "file", "foo bar", "/file ", true},
		{"hello", "", "", "", false},
	}

	for _, tt := range tests {
		cmd, argPrefix, replPrefix, ok := ActiveCommandArg(tt.input)
		if ok != tt.expectedOk || cmd != tt.expectedCmd || argPrefix != tt.expectedArgPrefix || replPrefix != tt.expectedReplPrefix {
			t.Errorf("ActiveCommandArg(%q) = (%q, %q, %q, %v), want (%q, %q, %q, %v)",
				tt.input, cmd, argPrefix, replPrefix, ok,
				tt.expectedCmd, tt.expectedArgPrefix, tt.expectedReplPrefix, tt.expectedOk)
		}
	}
}
