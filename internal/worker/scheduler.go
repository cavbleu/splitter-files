package worker

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"splitter-files/internal/extractor"
	"splitter-files/internal/models"
)

func ProcessFile(data []byte, outputDir string, numWorkers int, allowedExtensions map[string]bool) ([]models.ExtractionResult, *models.ExtractionStats, error) {
	wp := NewWorkerPool(numWorkers)
	processor := &extractor.DefaultFileProcessor{}
	wp.Start(outputDir, allowedExtensions, processor)

	stats := &models.ExtractionStats{
		InputSize: int64(len(data)),
		FileTypes: make(map[string]int),
	}

	var results []models.ExtractionResult
	var processingErrors []error
	var resultWg sync.WaitGroup
	var extractedFiles int32
	resultWg.Add(1)

	go func() {
		defer resultWg.Done()
		extractedRanges := make([][2]int, 0)

		for result := range wp.results {
			if result.Error != nil {
				processingErrors = append(processingErrors, result.Error)
				continue
			}

			atomic.AddInt32(&extractedFiles, 1)
			results = append(results, result)
			stats.TotalSize += int64(result.Size)
			stats.FileTypes[result.FileType]++

			newRange := [2]int{result.Start, result.End}
			overlapFound := false

			for _, r := range extractedRanges {
				if newRange[0] < r[1] && newRange[1] > r[0] {
					stats.Overlaps++
					overlapFound = true
					break
				}
			}

			if !overlapFound {
				extractedRanges = append(extractedRanges, newRange)
			}

			if result.OfficeInfo != nil {
				var officeType string
				switch result.OfficeInfo.Type {
				case models.WordDocument:
					officeType = "Word"
				case models.ExcelDocument:
					officeType = "Excel"
				case models.PowerPointDocument:
					officeType = "PowerPoint"
				default:
					officeType = "Unknown Office"
				}

				info := fmt.Sprintf("Extracted %s (%s, %d bytes, pos %d-%d)",
					filepath.Base(result.Filename), officeType, result.Size, result.Start, result.End)
				if result.OfficeInfo.IsEncrypted {
					info += " [ENCRYPTED]"
				}
				if result.OfficeInfo.IsMacro {
					info += " [MACROS]"
				}
				if result.OfficeInfo.Version != "" {
					info += fmt.Sprintf(" [v%s]", result.OfficeInfo.Version)
				}

				fmt.Println(info)
			} else {
				fmt.Printf("Extracted %s (%s, %d bytes, pos %d-%d)\n",
					filepath.Base(result.Filename), result.FileType, result.Size, result.Start, result.End)
			}
		}

		// Analyze data coverage
		covered := make([]bool, len(data))
		for _, r := range extractedRanges {
			start := r[0]
			if start < 0 {
				start = 0
			}
			end := r[1]
			if end > len(data) {
				end = len(data)
			}
			for i := start; i < end; i++ {
				covered[i] = true
			}
		}

		coveredCount := 0
		for _, v := range covered {
			if v {
				coveredCount++
			}
		}

		stats.Coverage = float64(coveredCount) / float64(len(data)) * 100
		stats.TotalExtracted = int(extractedFiles)
		stats.UncoveredAreas = analyzeUncoveredAreas(covered)
	}()

	pos := 0
	var counter int32 = 1
	const backoffTime = 100 * time.Millisecond

	officeQueue := make([]FileChunk, 0)
	regularQueue := make([]FileChunk, 0)

	for pos < len(data) || len(officeQueue) > 0 || len(regularQueue) > 0 {
		if len(officeQueue) > 0 {
			chunk := officeQueue[0]
			select {
			case wp.jobs <- chunk:
				officeQueue = officeQueue[1:]
				counter++
			case <-time.After(backoffTime):
			}
			continue
		}

		if len(regularQueue) > 0 {
			chunk := regularQueue[0]
			select {
			case wp.jobs <- chunk:
				regularQueue = regularQueue[1:]
				counter++
			case <-time.After(backoffTime):
			}
			continue
		}

		if pos < len(data) {
			remaining := data[pos:]
			if len(remaining) < 8 {
				break
			}

			var isOfficeFile bool
			foundSigs := extractor.FindFileSignatures(remaining, allowedExtensions)
			for _, sig := range foundSigs {
				if strings.HasPrefix(sig.Extension, "doc") ||
					strings.HasPrefix(sig.Extension, "xls") ||
					strings.HasPrefix(sig.Extension, "ppt") {
					isOfficeFile = true
					break
				}
			}

			chunk := FileChunk{
				Data:     remaining,
				Start:    pos,
				Counter:  counter,
				Priority: 0,
			}

			if isOfficeFile {
				chunk.Priority = 1
				officeQueue = append(officeQueue, chunk)
			} else {
				regularQueue = append(regularQueue, chunk)
			}

			pos++
		}
	}

	wp.Stop()
	resultWg.Wait()

	if len(processingErrors) > 0 {
		return results, stats, fmt.Errorf("encountered %d processing errors", len(processingErrors))
	}

	return results, stats, nil
}

func analyzeUncoveredAreas(covered []bool) []struct{ Start, End int } {
	var uncovered []struct{ Start, End int }
	inUncovered := false
	start := 0

	for i, v := range covered {
		if !v && !inUncovered {
			inUncovered = true
			start = i
		} else if v && inUncovered {
			inUncovered = false
			uncovered = append(uncovered, struct{ Start, End int }{start, i - 1})
		}
	}

	if inUncovered {
		uncovered = append(uncovered, struct{ Start, End int }{start, len(covered) - 1})
	}

	// Merge close areas
	if len(uncovered) > 1 {
		merged := make([]struct{ Start, End int }, 0)
		prev := uncovered[0]

		for i := 1; i < len(uncovered); i++ {
			current := uncovered[i]
			if current.Start-prev.End < 1024 { // Merge if distance is less than 1KB
				prev.End = current.End
			} else {
				merged = append(merged, prev)
				prev = current
			}
		}
		merged = append(merged, prev)
		return merged
	}

	return uncovered
}
