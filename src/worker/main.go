package main

import (
	"log"

	"bitovi.com/code-analyzer/src/activities/db"
	"bitovi.com/code-analyzer/src/activities/git"
	"bitovi.com/code-analyzer/src/activities/llm"
	"bitovi.com/code-analyzer/src/activities/s3"
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

	w.RegisterWorkflow(workflows.AnalyzeCode)

	w.RegisterActivity(db.InsertEmbedding)
	w.RegisterActivity(db.GetRelatedDocuments)
	w.RegisterActivity(db.GetEmbeddingCount)

	w.RegisterActivity(git.ArchiveRepository)

	w.RegisterActivity(llm.FetchEmbedding)
	w.RegisterActivity(llm.GetEmbeddingData)
	w.RegisterActivity(llm.InvokePrompt)

	w.RegisterActivity(s3.CreateBucket)
	w.RegisterActivity(s3.DeleteObject)
	w.RegisterActivity(s3.DeleteBucket)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}
}
