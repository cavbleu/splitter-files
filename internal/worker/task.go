package worker

import (
	"fmt"
	"sync"

	"splitter-files/internal/extractor"
	"splitter-files/internal/models"
)

type FileChunk struct {
	Data     []byte
	Start    int
	Counter  int32
	Priority int
}

// DefaultFileProcessor implements the basic file processing
type DefaultFileProcessor struct{}

func worker(id int, jobs <-chan FileChunk, results chan<- models.ExtractionResult,
	outputDir string, wg *sync.WaitGroup, allowedExtensions map[string]bool,
	processor extractor.FileProcessor) {
	defer wg.Done()

	for chunk := range jobs {
		size, endPos, filename, fileType, officeInfo, err := processor.Process(
			chunk.Data, outputDir, chunk.Counter, chunk.Start, allowedExtensions)

		if err != nil {
			results <- models.ExtractionResult{
				Error:   fmt.Errorf("worker %d: %v", id, err),
				Counter: chunk.Counter,
			}
			continue
		}

		results <- models.ExtractionResult{
			Filename:   filename,
			Size:       size,
			Start:      chunk.Start,
			End:        endPos,
			Counter:    chunk.Counter,
			FileType:   fileType,
			OfficeInfo: officeInfo,
		}
	}
}
