package main

import (
	"context"
	"log"
	"os"

	"bitovi.com/code-analyzer/src/utils"
	"bitovi.com/code-analyzer/src/workflows"
	"github.com/joho/godotenv"
	"go.temporal.io/sdk/client"
)

func main() {
	if len(os.Args) < 1 {
		log.Fatalln("Usage: `go run src/client/main.go <repository URL>`")
	}
	repository := os.Args[1]

	err := godotenv.Load()
	if err != nil {
		log.Fatalln("Unable to load .env file", err)
	}

	c, err := utils.GetTemporalClient()
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	input := workflows.AnalyzeInput{
		Repository: repository,
	}
	workflowID := "analyze-" + utils.CleanRepository(repository) + "-workflow"
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "ai-code-analyzer-queue",
	}
	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, workflows.CodeAnalyzer, input)
	if err != nil {
		log.Fatalln("Unable to execute workflow", err)
	}

	log.Println("Workflow ", we.GetID(), "running")

	var result workflows.AnalyzeOutput
	err = we.Get(context.Background(), &result)
	if err != nil {
		log.Fatalln("Unable get workflow result", err)
	}
	log.Println("Workflow result:", result)
}
