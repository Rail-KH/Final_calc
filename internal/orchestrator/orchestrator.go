package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rail-KH/Final_calc/internal/auth"
	"github.com/Rail-KH/Final_calc/internal/database"
	"github.com/Rail-KH/Final_calc/pkg/calculation"
)

type Config struct {
	Addr                string
	TimeAddition        int
	TimeSubtraction     int
	TimeMultiplications int
	TimeDivisions       int
}

func ConfigFromEnv() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	ta, _ := strconv.Atoi(os.Getenv("TIME_ADDITION_MS"))
	if ta == 0 {
		ta = 10
	}
	ts, _ := strconv.Atoi(os.Getenv("TIME_SUBTRACTION_MS"))
	if ts == 0 {
		ts = 10
	}
	tm, _ := strconv.Atoi(os.Getenv("TIME_MULTIPLICATIONS_MS"))
	if tm == 0 {
		tm = 10
	}
	td, _ := strconv.Atoi(os.Getenv("TIME_DIVISIONS_MS"))
	if td == 0 {
		td = 10
	}
	return &Config{
		Addr:                port,
		TimeAddition:        ta,
		TimeSubtraction:     ts,
		TimeMultiplications: tm,
		TimeDivisions:       td,
	}
}

type Orchestrator struct {
	Config      *Config
	exprStore   map[string]*Expression
	taskStore   map[string]*Task
	TaskQueue   []*Task
	mu          sync.Mutex
	taskCounter int64
	Database    *database.DataBase
}

func NewOrchestrator() *Orchestrator {
	database, err := database.CreateTable()
	if err != nil {
		log.Fatal(err)
	}

	return &Orchestrator{
		Config:    ConfigFromEnv(),
		exprStore: make(map[string]*Expression),
		taskStore: make(map[string]*Task),
		TaskQueue: make([]*Task, 0),
		Database:  database,
	}
}

type Expression struct {
	ID     string            `json:"id"`
	Expr   string            `json:"expression"`
	UserID string            `json:"-"`
	Status string            `json:"status"`
	Result *float64          `json:"result"`
	AST    *calculation.Node `json:"-"`
}

type Task struct {
	ID            string            `json:"id"`
	ExprID        string            `json:"-"`
	UserID        string            `json:"-"`
	Arg1          float64           `json:"arg1"`
	Arg2          float64           `json:"arg2"`
	Operation     string            `json:"operation"`
	OperationTime int               `json:"operation_time"`
	Node          *calculation.Node `json:"-"`
}

var req struct {
	mal string
}

func (o *Orchestrator) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Login == "" || req.Password == "" {
		http.Error(w, `{"error":"Login and password are required"}`, http.StatusBadRequest)
		return
	}

	hashedPassword, err := auth.HashPass(req.Password)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	userID, err := o.Database.InsertUser(req.Login, hashedPassword)
	if err != nil {
		if errors.Is(err, errors.New("already exists")) {
			http.Error(w, `{"error":"User already exists"}`, http.StatusConflict)
			return
		}
		log.Printf("Failed to create user: %v", err)
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":    userID,
		"login": req.Login,
	})
}

func (o *Orchestrator) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	user, err := o.Database.SelectUser(req.Login)
	if err != nil {
		if errors.Is(err, errors.New("not found")) {
			http.Error(w, `{"error":"Invalid credentials"}`, http.StatusUnauthorized)
			return
		}
		log.Printf("Failed to get user: %v", err)
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	if auth.CheckCorPass(req.Password, user.Password) {
		http.Error(w, `{"error":"Invalid credentials"}`, http.StatusUnauthorized)
		return
	}
	token, err := auth.GenJWT(int(user.ID))
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":    user.ID,
			"login": user.Login,
		},
	})
}

func (o *Orchestrator) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" || r.URL.Path == "/api/v1/register" || r.URL.Path == "/internal/task" {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"Authorization header is required"}`, http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			http.Error(w, `{"error":"Invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}

		userID, err := auth.ParseJWT(tokenString)
		if err != nil {
			http.Error(w, `{"error":"Invalid token"}`, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), req, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (o *Orchestrator) CalculateHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(req).(int)
	if !ok {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Wrong Method: Need a POST Method"}`, http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Expression string `json:"expression"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Expression == "" {
		http.Error(w, `{"error":"Invalid Body"}`, http.StatusUnprocessableEntity)
		return
	}

	dbExpr, err := o.Database.CreateExpression(userID, req.Expression)
	if err != nil {
		http.Error(w, `{"error":"Failed to create expression"}`, http.StatusInternalServerError)
		return
	}

	expr := &Expression{
		ID:     strconv.Itoa(dbExpr.ID),
		Expr:   req.Expression,
		Status: "pending",
		UserID: strconv.Itoa(userID),
	}
	ast, err := calculation.ParseAST(req.Expression)

	if err != nil {
		o.Database.UpdateExpression(&database.Expression{
			ID:     dbExpr.ID,
			Status: "error",
		})

		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusUnprocessableEntity)
		return
	}
	expr.AST = ast
	o.exprStore[expr.ID] = expr
	o.ScheduleTasks(expr)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": expr.ID})
}

func (o *Orchestrator) ExpressionsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(req).(int)
	if !ok {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Wrong Method: Need a GET Method"}`, http.StatusMethodNotAllowed)
		return
	}

	dbExprs, err := o.Database.GetExpressions(userID)
	if err != nil {
		http.Error(w, `{"error":"Failed to get expressions"}`, http.StatusInternalServerError)
		return
	}

	exprs := make([]*Expression, len(dbExprs))
	for i, dbExpr := range dbExprs {
		exprs[i] = &Expression{
			ID:     strconv.Itoa(dbExpr.ID),
			Expr:   dbExpr.Expression,
			Status: dbExpr.Status,
			Result: dbExpr.Result,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"expressions": exprs})
}

func (o *Orchestrator) ExpressionByIDHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(req).(int)
	if !ok {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Wrong Method"}`, http.StatusMethodNotAllowed)
		return
	}
	id := string(r.URL.Path[len("/api/v1/expressions/"):][1:])
	idInt, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, `{"error":"Invalid expression ID"}`, http.StatusBadRequest)
		return
	}
	dbExpr, err := o.Database.GetExpressionByID(idInt, userID)
	if err != nil {
		if errors.Is(err, errors.New("not found")) {
			http.Error(w, `{"error":"Expression not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"Failed to get expression"}`, http.StatusInternalServerError)
		return
	}

	expr := &Expression{
		ID:     id,
		Expr:   dbExpr.Expression,
		Status: dbExpr.Status,
		Result: dbExpr.Result,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"expression": expr})
}

func (o *Orchestrator) GetTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Wrong Method"}`, http.StatusMethodNotAllowed)
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if len(o.TaskQueue) == 0 {
		http.Error(w, `{"error":"No task available"}`, http.StatusNotFound)
		return
	}
	task := o.TaskQueue[0]
	o.TaskQueue = o.TaskQueue[1:]
	if expr, exists := o.exprStore[task.ExprID]; exists {
		expr.Status = "completed"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"task": task})
}

func (o *Orchestrator) PostTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Wrong Method"}`, http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID     string  `json:"id"`
		Result float64 `json:"result"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.ID == "" {
		http.Error(w, `{"error":"Invalid Body"}`, http.StatusUnprocessableEntity)
		return
	}
	o.mu.Lock()
	task, ok := o.taskStore[req.ID]
	if !ok {
		o.mu.Unlock()
		http.Error(w, `{"error":"Task not found"}`, http.StatusNotFound)
		return
	}
	task.Node.IsLeaf = true
	task.Node.Value = req.Result
	if expr, exists := o.exprStore[task.ExprID]; exists {
		o.ScheduleTasks(expr)
		if expr.AST.IsLeaf {
			expr.Status = "completed"
			expr.Result = &expr.AST.Value
		}
		user_id, err := strconv.Atoi(expr.UserID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		id, err := strconv.Atoi(expr.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		o.Database.UpdateExpression(&database.Expression{
			UserID:     user_id,
			ID:         id,
			Expression: expr.Expr,
			Status:     expr.Status,
			Result:     expr.Result,
		})
	}
	o.mu.Unlock()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"result accepted"}`))
}

func (o *Orchestrator) ScheduleTasks(expr *Expression) {
	var traverse func(node *calculation.Node)
	traverse = func(node *calculation.Node) {
		if node == nil || node.IsLeaf {
			return
		}
		traverse(node.Left)
		traverse(node.Right)
		if node.Left != nil && node.Right != nil && node.Left.IsLeaf && node.Right.IsLeaf {
			if !node.TaskScheduled {
				o.taskCounter++
				taskID := fmt.Sprintf("%d", o.taskCounter)
				var opTime int
				switch node.Operator {
				case "+":
					opTime = o.Config.TimeAddition
				case "-":
					opTime = o.Config.TimeSubtraction
				case "*":
					opTime = o.Config.TimeMultiplications
				case "/":
					opTime = o.Config.TimeDivisions
				default:
					opTime = 100
				}
				task := &Task{
					ID:            taskID,
					ExprID:        expr.ID,
					UserID:        expr.UserID,
					Arg1:          node.Left.Value,
					Arg2:          node.Right.Value,
					Operation:     node.Operator,
					OperationTime: opTime,
					Node:          node,
				}
				node.TaskScheduled = true
				o.taskStore[taskID] = task
				o.TaskQueue = append(o.TaskQueue, task)
			}
		}
	}
	traverse(expr.AST)
}

func (o *Orchestrator) RunServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/register", o.RegisterHandler)
	mux.HandleFunc("/api/v1/login", o.LoginHandler)
	mux.HandleFunc("/api/v1/calculate", o.CalculateHandler)
	mux.HandleFunc("/api/v1/expressions", o.ExpressionsHandler)
	mux.HandleFunc("/api/v1/expressions/", o.ExpressionByIDHandler)
	mux.HandleFunc("/internal/task", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			o.GetTaskHandler(w, r)

		} else if r.Method == http.MethodPost {
			o.PostTaskHandler(w, r)
		} else {
			http.Error(w, `{"error":"Wrong Method: Need a POST or GET Method"}`, http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"Not Found"}`, http.StatusNotFound)
	})
	go func() {
		for {
			time.Sleep(2 * time.Second)
			o.mu.Lock()
			if len(o.TaskQueue) > 0 {
				log.Printf("Pending tasks in queue: %d", len(o.TaskQueue))
			}
			o.mu.Unlock()
		}
	}()
	return http.ListenAndServe(":"+o.Config.Addr, o.AuthMiddleware(mux))
}
