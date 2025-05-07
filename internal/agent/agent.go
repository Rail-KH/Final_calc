package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Agent struct {
	ComputingPower  int
	OrchestratorURL string
}

func NewAgent() *Agent {
	cp, err := strconv.Atoi(os.Getenv("COMPUTING_POWER"))
	if err != nil || cp < 1 {
		cp = 4
	}
	orchestratorURL := os.Getenv("ORCHESTRATOR_URL")
	if orchestratorURL == "" {
		orchestratorURL = "http://localhost:8080"
	}
	return &Agent{
		ComputingPower:  cp,
		OrchestratorURL: orchestratorURL,
	}
}

func (a *Agent) Run() {
	for i := 0; i < a.ComputingPower; i++ {
		log.Printf("Starting worker %d", i)
		go a.worker(i)
	}
	select {}
}

func (a *Agent) worker(id int) {
	for {
		resp, err := http.Get(a.OrchestratorURL + "/internal/task")
		if err != nil {
			log.Printf("Worker %d: error getting task: %v", id, err)
			time.Sleep(2 * time.Second)
			continue
		}
		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			time.Sleep(1 * time.Second)
			continue
		}
		var taskResp struct {
			Task struct {
				ID            string  `json:"id"`
				Arg1          float64 `json:"arg1"`
				Arg2          float64 `json:"arg2"`
				Operation     string  `json:"operation"`
				OperationTime int     `json:"operation_time"`
			} `json:"task"`
		}
		err = json.NewDecoder(resp.Body).Decode(&taskResp)
		resp.Body.Close()
		if err != nil {
			log.Printf("Worker %d: error decoding task: %v", id, err)
			time.Sleep(1 * time.Second)
			continue
		}
		task := taskResp.Task
		log.Printf("Worker %d: received task %s: %f %s %f, simulating %d ms", id, task.ID, task.Arg1, task.Operation, task.Arg2, task.OperationTime)
		time.Sleep(time.Duration(task.OperationTime) * time.Millisecond)
		result, err := Calc(task.Operation, task.Arg1, task.Arg2)
		if err != nil {
			log.Printf("Worker %d: error computing task %s: %v", id, task.ID, err)
			continue
		}
		resultPayload := map[string]interface{}{
			"id":     task.ID,
			"result": result,
		}
		payloadBytes, _ := json.Marshal(resultPayload)
		respPost, err := http.Post(a.OrchestratorURL+"/internal/task", "application/json", bytes.NewReader(payloadBytes))
		if err != nil {
			log.Printf("Worker %d: error posting result for task %s: %v", id, task.ID, err)
			continue
		}
		if respPost.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(respPost.Body)
			log.Printf("Worker %d: error response posting result for task %s: %s", id, task.ID, string(body))
		} else {
			log.Printf("Worker %d: successfully completed task %s with result %f", id, task.ID, result)
		}
		respPost.Body.Close()
	}
}

func Calc(operation string, a, b float64) (float64, error) {
	switch operation {
	case "+":
		return a + b, nil
	case "-":
		return a - b, nil
	case "*":
		return a * b, nil
	case "/":
		if b == 0 {
			return 0, errors.New("ErrDivisionByZero")
		}
		return a / b, nil
	default:
		return 0, errors.New("InvalidOper")
	}
}
