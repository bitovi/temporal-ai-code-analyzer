package main

import (
	"log"

	"bitovi.com/code-analyzer/src/activities/git"
	"bitovi.com/code-analyzer/src/utils"
	"bitovi.com/code-analyzer/src/workflows"
	"go.temporal.io/sdk/worker"
)

func main() {
	c, err := utils.GetTemporalClient()
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	w := worker.New(c, "ai-code-analyzer-queue", worker.Options{})

	w.RegisterWorkflow(workflows.CodeAnalyzer)
	w.RegisterActivity(git.CloneRepository)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}
}
