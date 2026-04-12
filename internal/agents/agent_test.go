package agents

import (
	"reflect"
	"testing"
)

func TestFormatOutput(t *testing.T) {
	tests := []struct {
		name      string
		agentName string
		output    string
		expected  string
	}{
		{
			name:      "Standard formatting",
			agentName: "Analyzer",
			output:    "This is good.",
			expected:  "**[Analyzer]**\nThis is good.",
		},
		{
			name:      "Empty output formatting",
			agentName: "EmptyBot",
			output:    "",
			expected:  "**[EmptyBot]**\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatOutput(tt.agentName, tt.output)
			if result != tt.expected {
				t.Errorf("FormatOutput() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractJSON(t *testing.T) {
	type DummyStruct struct {
		Field1 string `json:"field1"`
		Field2 int    `json:"field2"`
	}

	tests := []struct {
		name         string
		raw          string
		wantConfig   DummyStruct
		expectingErr bool
	}{
		{
			name:         "Clean JSON",
			raw:          `{"field1": "hello", "field2": 42}`,
			wantConfig:   DummyStruct{Field1: "hello", Field2: 42},
			expectingErr: false,
		},
		{
			name:         "JSON wrapped in markdown",
			raw:          "Here is your config:\n```json\n{\"field1\": \"world\", \"field2\": 99}\n```",
			wantConfig:   DummyStruct{Field1: "world", Field2: 99},
			expectingErr: false,
		},
		{
			name:         "JSON wrapped in random text",
			raw:          "Okay I analyzed the prompt and concluded { \"field1\": \"random\", \"field2\": 1} is best.",
			wantConfig:   DummyStruct{Field1: "random", Field2: 1},
			expectingErr: false,
		},
		{
			name:         "No JSON present",
			raw:          "Just text, no curly braces",
			wantConfig:   DummyStruct{},
			expectingErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result DummyStruct
			err := ExtractJSON(tt.raw, &result)

			if (err != nil) != tt.expectingErr {
				t.Errorf("ExtractJSON() error = %v, expectingErr %v", err, tt.expectingErr)
				return
			}
			if !tt.expectingErr && !reflect.DeepEqual(result, tt.wantConfig) {
				t.Errorf("ExtractJSON() result = %v, wantConfig %v", result, tt.wantConfig)
			}
		})
	}
}
