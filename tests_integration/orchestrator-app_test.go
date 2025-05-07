package tests_integration

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Rail-KH/Final_calc/internal/auth"
	orch "github.com/Rail-KH/Final_calc/internal/orchestrator"
	"github.com/Rail-KH/Final_calc/pkg/calculation"
	"github.com/stretchr/testify/assert"
)

func TestConfigFromEnv(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		os.Unsetenv("PORT")
		os.Unsetenv("TIME_ADDITION_MS")
		os.Unsetenv("TIME_SUBTRACTION_MS")
		os.Unsetenv("TIME_MULTIPLICATIONS_MS")
		os.Unsetenv("TIME_DIVISIONS_MS")

		config := orch.ConfigFromEnv()

		assert.Equal(t, "8080", config.Addr)
		assert.Equal(t, 10, config.TimeAddition)
		assert.Equal(t, 10, config.TimeSubtraction)
		assert.Equal(t, 10, config.TimeMultiplications)
		assert.Equal(t, 10, config.TimeDivisions)
	})

	t.Run("custom values", func(t *testing.T) {
		os.Setenv("PORT", "9090")
		os.Setenv("TIME_ADDITION_MS", "20")
		os.Setenv("TIME_SUBTRACTION_MS", "30")
		os.Setenv("TIME_MULTIPLICATIONS_MS", "40")
		os.Setenv("TIME_DIVISIONS_MS", "50")

		config := orch.ConfigFromEnv()

		assert.Equal(t, "9090", config.Addr)
		assert.Equal(t, 20, config.TimeAddition)
		assert.Equal(t, 30, config.TimeSubtraction)
		assert.Equal(t, 40, config.TimeMultiplications)
		assert.Equal(t, 50, config.TimeDivisions)
	})
}

func TestScheduleTasks(t *testing.T) {
	o := orch.NewOrchestrator()
	expr := &orch.Expression{
		ID:   "test-expr",
		Expr: "2 + 3 * 4",
		AST: &calculation.Node{
			Operator: "+",
			Left: &calculation.Node{
				IsLeaf: true,
				Value:  2,
			},
			Right: &calculation.Node{
				Operator: "*",
				Left: &calculation.Node{
					IsLeaf: true,
					Value:  3,
				},
				Right: &calculation.Node{
					IsLeaf: true,
					Value:  4,
				},
			},
		},
	}

	o.ScheduleTasks(expr)

	assert.Equal(t, 1, len(o.TaskQueue))
	assert.Equal(t, "*", o.TaskQueue[0].Operation)
	assert.Equal(t, 3.0, o.TaskQueue[0].Arg1)
	assert.Equal(t, 4.0, o.TaskQueue[0].Arg2)
}

func TestAuthMiddleware(t *testing.T) {
	o := orch.NewOrchestrator()
	handler := o.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("allowed paths", func(t *testing.T) {
		paths := []string{"/api/v1/login", "/api/v1/register", "/internal/task"}
		for _, path := range paths {
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code, "path: %s", path)
		}
	})

	t.Run("unauthorized request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/expressions", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("authorized request", func(t *testing.T) {
		o.Database.InsertUser("middlewareuser", "middlewarepass")
		user, _ := o.Database.SelectUser("middlewareuser")
		token, _ := auth.GenJWT(int(user.ID))

		req := httptest.NewRequest("GET", "/api/v1/expressions", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
