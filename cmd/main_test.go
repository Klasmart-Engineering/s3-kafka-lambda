package main

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestFileType(t *testing.T) {
	type TestCases struct {
		description string
		input       string
		expected    string
	}

	for _, scenario := range []TestCases{
		{
			description: "should return standard file format",
			input:       uuid.New().String() + ".csv",
			expected:    "csv",
		},
		{
			description: "should handle multiple full stops",
			input:       "fdsf.asdfds.fasdf.fadsf.txt",
			expected:    "txt",
		},
		{
			description: "should return full string when file format are special characters",
			input:       "fdsaf.%*&!@",
			expected:    "fdsaf.%*&!@",
		},
		{
			description: "should return full string when file format has special characters",
			input:       "fdsaf.!txt",
			expected:    "fdsaf.!txt",
		},
		{
			description: "no suffix",
			input:       "abc",
			expected:    "abc",
		},
	} {
		t.Run(scenario.description, func(t *testing.T) {
			result := fileType(scenario.input)
			assert.Equal(t, scenario.expected, result)
		})
	}
}
