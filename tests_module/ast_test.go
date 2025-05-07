package tests_module

import (
	"testing"

	"github.com/Rail-KH/Final_calc/pkg/calculation"
)

func TestParseAST(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		want       *calculation.Node
		wantErr    bool
	}{
		{
			name:       "simple number",
			expression: "42",
			want:       &calculation.Node{IsLeaf: true, Value: 42},
			wantErr:    false,
		},
		{
			name:       "simple addition",
			expression: "2+3",
			want: &calculation.Node{
				IsLeaf:   false,
				Operator: "+",
				Left:     &calculation.Node{IsLeaf: true, Value: 2},
				Right:    &calculation.Node{IsLeaf: true, Value: 3},
			},
			wantErr: false,
		},
		{
			name:       "addition and multiplication",
			expression: "2+3*4",
			want: &calculation.Node{
				IsLeaf:   false,
				Operator: "+",
				Left:     &calculation.Node{IsLeaf: true, Value: 2},
				Right: &calculation.Node{
					IsLeaf:   false,
					Operator: "*",
					Left:     &calculation.Node{IsLeaf: true, Value: 3},
					Right:    &calculation.Node{IsLeaf: true, Value: 4},
				},
			},
			wantErr: false,
		},
		{
			name:       "with parentheses",
			expression: "(2+3)*4",
			want: &calculation.Node{
				IsLeaf:   false,
				Operator: "*",
				Left: &calculation.Node{
					IsLeaf:   false,
					Operator: "+",
					Left:     &calculation.Node{IsLeaf: true, Value: 2},
					Right:    &calculation.Node{IsLeaf: true, Value: 3},
				},
				Right: &calculation.Node{IsLeaf: true, Value: 4},
			},
			wantErr: false,
		},
		{
			name:       "division and subtraction",
			expression: "10/2-1",
			want: &calculation.Node{
				IsLeaf:   false,
				Operator: "-",
				Left: &calculation.Node{
					IsLeaf:   false,
					Operator: "/",
					Left:     &calculation.Node{IsLeaf: true, Value: 10},
					Right:    &calculation.Node{IsLeaf: true, Value: 2},
				},
				Right: &calculation.Node{IsLeaf: true, Value: 1},
			},
			wantErr: false,
		},
		{
			name:       "decimal numbers",
			expression: "3.14*2.5",
			want: &calculation.Node{
				IsLeaf:   false,
				Operator: "*",
				Left:     &calculation.Node{IsLeaf: true, Value: 3.14},
				Right:    &calculation.Node{IsLeaf: true, Value: 2.5},
			},
			wantErr: false,
		},
		{
			name:       "negative number",
			expression: "-5+3",
			want: &calculation.Node{
				IsLeaf:   false,
				Operator: "+",
				Left:     &calculation.Node{IsLeaf: true, Value: -5},
				Right:    &calculation.Node{IsLeaf: true, Value: 3},
			},
			wantErr: false,
		},
		{
			name:       "empty expression",
			expression: "",
			want:       nil,
			wantErr:    true,
		},
		{
			name:       "invalid expression",
			expression: "2+",
			want:       nil,
			wantErr:    true,
		},
		{
			name:       "unbalanced parentheses",
			expression: "(2+3",
			want:       nil,
			wantErr:    true,
		},
		{
			name:       "invalid characters",
			expression: "2+a",
			want:       nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calculation.ParseAST(tt.expression)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAST() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !compareNodes(got, tt.want) {
				t.Errorf("ParseAST() = %v, want %v", got, tt.want)
			}
		})
	}
}

func compareNodes(a, b *calculation.Node) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.IsLeaf != b.IsLeaf {
		return false
	}
	if a.IsLeaf {
		return a.Value == b.Value
	}
	return a.Operator == b.Operator &&
		compareNodes(a.Left, b.Left) &&
		compareNodes(a.Right, b.Right)
}
