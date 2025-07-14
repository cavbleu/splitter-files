package worker

import (
	"sync"

	"splitter-files/internal/extractor"
	"splitter-files/internal/models"
)

type WorkerPool struct {
	numWorkers int
	jobs       chan FileChunk
	results    chan models.ExtractionResult
	wg         *sync.WaitGroup
}

func NewWorkerPool(numWorkers int) *WorkerPool {
	return &WorkerPool{
		numWorkers: numWorkers,
		jobs:       make(chan FileChunk, numWorkers*2),
		results:    make(chan models.ExtractionResult, numWorkers*2),
		wg:         &sync.WaitGroup{},
	}
}

func (wp *WorkerPool) Start(outputDir string, allowedExtensions map[string]bool, processor extractor.FileProcessor) {
	for i := 0; i < wp.numWorkers; i++ {
		wp.wg.Add(1)
		go worker(i, wp.jobs, wp.results, outputDir, wp.wg, allowedExtensions, processor)
	}
}

func (wp *WorkerPool) Stop() {
	close(wp.jobs)
	wp.wg.Wait()
	close(wp.results)
}
