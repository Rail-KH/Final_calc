package tests_module

import (
	"strconv"
	"testing"

	"github.com/Rail-KH/Final_calc/internal/auth"
)

func TestVerifyJWTToken(t *testing.T) {
	userID := 12345
	token, err := auth.GenJWT(userID)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	tests := []struct {
		name       string
		token      string
		expectUser string
		expectErr  bool
	}{
		{
			name:       "valid token",
			token:      token,
			expectUser: "12345",
			expectErr:  false,
		},
		{
			name:       "invalid token",
			token:      "invalid.token.string",
			expectUser: "0",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, err := auth.ParseJWT(tt.token)

			if (err != nil) != tt.expectErr {
				t.Errorf("expected error status %v, got %v", tt.expectErr, err != nil)
			}

			if strconv.Itoa(userID) != tt.expectUser {
				t.Errorf("expected userID %s, got %d", tt.expectUser, userID)
			}
		})
	}
}
