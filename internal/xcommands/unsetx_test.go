package xcommands

import (
	"os"
	"testing"
)

func TestUnsetxCommand_Execute(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flags       []string
		setup       func() error
		validate    func() error
		expectError bool
	}{
		{
			name:  "unset single existing variable",
			args:  []string{"TEST_VAR1"},
			flags: []string{},
			setup: func() error {
				return os.Setenv("TEST_VAR1", "test_value")
			},
			validate: func() error {
				if _, exists := os.LookupEnv("TEST_VAR1"); exists {
					t.Error("Expected TEST_VAR1 to be unset")
				}
				return nil
			},
			expectError: false,
		},
		{
			name:  "unset multiple existing variables",
			args:  []string{"TEST_VAR2", "TEST_VAR3", "TEST_VAR4"},
			flags: []string{},
			setup: func() error {
				if err := os.Setenv("TEST_VAR2", "value2"); err != nil {
					return err
				}
				if err := os.Setenv("TEST_VAR3", "value3"); err != nil {
					return err
				}
				return os.Setenv("TEST_VAR4", "value4")
			},
			validate: func() error {
				for _, varName := range []string{"TEST_VAR2", "TEST_VAR3", "TEST_VAR4"} {
					if _, exists := os.LookupEnv(varName); exists {
						t.Errorf("Expected %s to be unset", varName)
					}
				}
				return nil
			},
			expectError: false,
		},
		{
			name:  "unset non-existent variable",
			args:  []string{"NON_EXISTENT_VAR"},
			flags: []string{},
			setup: func() error {
				// Ensure the variable doesn't exist
				os.Unsetenv("NON_EXISTENT_VAR")
				return nil
			},
			validate: func() error {
				// This should be a no-op, no error expected
				return nil
			},
			expectError: false,
		},
		{
			name:  "unset mix of existing and non-existent variables",
			args:  []string{"EXISTING_VAR", "NON_EXISTENT_VAR", "ANOTHER_EXISTING_VAR"},
			flags: []string{},
			setup: func() error {
				if err := os.Setenv("EXISTING_VAR", "exists"); err != nil {
					return err
				}
				os.Unsetenv("NON_EXISTENT_VAR") // Ensure it doesn't exist
				return os.Setenv("ANOTHER_EXISTING_VAR", "also_exists")
			},
			validate: func() error {
				if _, exists := os.LookupEnv("EXISTING_VAR"); exists {
					t.Error("Expected EXISTING_VAR to be unset")
				}
				if _, exists := os.LookupEnv("ANOTHER_EXISTING_VAR"); exists {
					t.Error("Expected ANOTHER_EXISTING_VAR to be unset")
				}
				return nil
			},
			expectError: false,
		},
		{
			name:  "unset variable with special characters",
			args:  []string{"TEST_VAR_WITH_UNDERSCORES"},
			flags: []string{},
			setup: func() error {
				return os.Setenv("TEST_VAR_WITH_UNDERSCORES", "special_value")
			},
			validate: func() error {
				if _, exists := os.LookupEnv("TEST_VAR_WITH_UNDERSCORES"); exists {
					t.Error("Expected TEST_VAR_WITH_UNDERSCORES to be unset")
				}
				return nil
			},
			expectError: false,
		},
		{
			name:  "unset variable with empty string value",
			args:  []string{"EMPTY_VALUE_VAR"},
			flags: []string{},
			setup: func() error {
				return os.Setenv("EMPTY_VALUE_VAR", "")
			},
			validate: func() error {
				if _, exists := os.LookupEnv("EMPTY_VALUE_VAR"); exists {
					t.Error("Expected EMPTY_VALUE_VAR to be unset")
				}
				return nil
			},
			expectError: false,
		},
		{
			name:        "no operands provided",
			args:        []string{},
			flags:       []string{},
			setup:       func() error { return nil },
			validate:    func() error { return nil },
			expectError: true,
		},
		{
			name:  "unset with empty variable name (should be skipped)",
			args:  []string{"VALID_VAR", "", "ANOTHER_VALID_VAR"},
			flags: []string{},
			setup: func() error {
				if err := os.Setenv("VALID_VAR", "valid"); err != nil {
					return err
				}
				return os.Setenv("ANOTHER_VALID_VAR", "also_valid")
			},
			validate: func() error {
				if _, exists := os.LookupEnv("VALID_VAR"); exists {
					t.Error("Expected VALID_VAR to be unset")
				}
				if _, exists := os.LookupEnv("ANOTHER_VALID_VAR"); exists {
					t.Error("Expected ANOTHER_VALID_VAR to be unset")
				}
				return nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if err := tt.setup(); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Execute
			unsetx := &UnsetxCommand{}
			err := unsetx.Execute(tt.args, tt.flags)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Validate result if no error expected
			if !tt.expectError {
				if err := tt.validate(); err != nil {
					t.Errorf("Validation failed: %v", err)
				}
			}
		})
	}
}

func TestUnsetxCommand_ExecuteErrorHandling(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectedMsg  string
	}{
		{
			name:        "missing operands",
			args:        []string{},
			expectedMsg: "unsetx: missing operands",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unsetx := &UnsetxCommand{}
			err := unsetx.Execute(tt.args, []string{})

			if err == nil {
				t.Error("Expected error but got none")
			}

			if err.Error() != tt.expectedMsg {
				t.Errorf("Expected error message %q, got %q", tt.expectedMsg, err.Error())
			}
		})
	}
}

func TestUnsetxCommand_VariableExistenceCheck(t *testing.T) {
	// Test that the command properly checks for variable existence
	// and handles both existing and non-existing variables correctly

	testVarName := "UNSETX_TEST_EXISTENCE"
	
	// Ensure variable doesn't exist initially
	os.Unsetenv(testVarName)
	
	// Verify it doesn't exist
	if _, exists := os.LookupEnv(testVarName); exists {
		t.Fatalf("Test variable %s should not exist initially", testVarName)
	}

	// Test unsetting non-existent variable (should not error)
	unsetx := &UnsetxCommand{}
	err := unsetx.Execute([]string{testVarName}, []string{})
	if err != nil {
		t.Errorf("Unsetting non-existent variable should not error, got: %v", err)
	}

	// Set the variable
	if err := os.Setenv(testVarName, "test_value"); err != nil {
		t.Fatalf("Failed to set test variable: %v", err)
	}

	// Verify it exists
	if value, exists := os.LookupEnv(testVarName); !exists || value != "test_value" {
		t.Fatalf("Test variable should exist with value 'test_value', got exists=%v, value=%q", exists, value)
	}

	// Unset the existing variable
	err = unsetx.Execute([]string{testVarName}, []string{})
	if err != nil {
		t.Errorf("Unsetting existing variable failed: %v", err)
	}

	// Verify it no longer exists
	if _, exists := os.LookupEnv(testVarName); exists {
		t.Error("Variable should be unset after execution")
	}
}

func TestUnsetxCommand_PreserveOtherVariables(t *testing.T) {
	// Test that unsetting one variable doesn't affect others
	
	testVars := map[string]string{
		"UNSETX_PRESERVE_1": "value1",
		"UNSETX_PRESERVE_2": "value2",
		"UNSETX_PRESERVE_3": "value3",
	}

	// Set all test variables
	for name, value := range testVars {
		if err := os.Setenv(name, value); err != nil {
			t.Fatalf("Failed to set test variable %s: %v", name, err)
		}
	}

	// Unset only one variable
	unsetx := &UnsetxCommand{}
	err := unsetx.Execute([]string{"UNSETX_PRESERVE_2"}, []string{})
	if err != nil {
		t.Errorf("Failed to unset variable: %v", err)
	}

	// Verify the unset variable is gone
	if _, exists := os.LookupEnv("UNSETX_PRESERVE_2"); exists {
		t.Error("UNSETX_PRESERVE_2 should be unset")
	}

	// Verify other variables still exist
	for name, expectedValue := range testVars {
		if name == "UNSETX_PRESERVE_2" {
			continue // Skip the one we unset
		}
		
		if value, exists := os.LookupEnv(name); !exists {
			t.Errorf("Variable %s should still exist", name)
		} else if value != expectedValue {
			t.Errorf("Variable %s should have value %q, got %q", name, expectedValue, value)
		}
	}

	// Clean up remaining variables
	for name := range testVars {
		os.Unsetenv(name)
	}
}

func TestUnsetxCommand_EmptyVariableNames(t *testing.T) {
	// Test behavior with empty variable names in the argument list
	
	// Set up a test variable
	testVar := "UNSETX_EMPTY_TEST"
	if err := os.Setenv(testVar, "test_value"); err != nil {
		t.Fatalf("Failed to set test variable: %v", err)
	}

	// Execute with empty strings in args (should be skipped)
	unsetx := &UnsetxCommand{}
	err := unsetx.Execute([]string{"", testVar, ""}, []string{})
	if err != nil {
		t.Errorf("Unexpected error with empty variable names: %v", err)
	}

	// Verify the actual variable was unset
	if _, exists := os.LookupEnv(testVar); exists {
		t.Error("Test variable should be unset")
	}

	// Clean up
	os.Unsetenv(testVar)
}