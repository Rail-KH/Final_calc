package main

import (
	"log"

	"github.com/Rail-KH/Final_calc/internal/agent"
)

func main() {
	agent := agent.NewAgent()
	log.Println("Starting Agent...")
	agent.Run()
}
