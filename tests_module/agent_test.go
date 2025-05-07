package tests_module

import (
	"errors"
	"testing"

	"github.com/Rail-KH/Final_calc/internal/agent"
)

func TestCalc(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		a, b      float64
		expected  float64
		expectErr bool
		err       error
	}{
		{
			name:      "Addition positive numbers",
			operation: "+",
			a:         42.5,
			b:         37.5,
			expected:  80.0,
			expectErr: false,
		},
		{
			name:      "Addition negative numbers",
			operation: "+",
			a:         -42.5,
			b:         -37.5,
			expected:  -80.0,
			expectErr: false,
		},

		{
			name:      "Subtraction positive numbers",
			operation: "-",
			a:         5.0,
			b:         2.6,
			expected:  2.4,
			expectErr: false,
		},
		{
			name:      "Subtraction negative numbers",
			operation: "-",
			a:         -5.0,
			b:         -2.6,
			expected:  -2.4,
			expectErr: false,
		},

		{
			name:      "Multiplication positive numbers",
			operation: "*",
			a:         2.0,
			b:         3.0,
			expected:  6.0,
			expectErr: false,
		},
		{
			name:      "Multiplication by zero",
			operation: "*",
			a:         2.0,
			b:         0.0,
			expected:  0.0,
			expectErr: false,
		},

		{
			name:      "Division positive numbers",
			operation: "/",
			a:         52.0,
			b:         2.0,
			expected:  26.0,
			expectErr: false,
		},
		{
			name:      "Division by zero",
			operation: "/",
			a:         52.0,
			b:         0.0,
			expected:  0.0,
			expectErr: true,
			err:       errors.New("ErrDivisionByZero"),
		},

		{
			name:      "Invalid operator",
			operation: "$",
			a:         2.0,
			b:         3.0,
			expected:  0.0,
			expectErr: true,
			err:       errors.New("InvalidOper"),
		},
	}
	t.Parallel()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := agent.Calc(tt.operation, tt.a, tt.b)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if err.Error() != tt.err.Error() {
					t.Errorf("expected error: %v, got: %v", tt.err, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected: %v, got: %v", tt.expected, result)
			}
		})
	}
}
