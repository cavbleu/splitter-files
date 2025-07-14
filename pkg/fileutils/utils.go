package fileutils

import (
	"fmt"
	"splitter-files/internal/models"
)

func GetMapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func PrintStats(stats *models.ExtractionStats, results []models.ExtractionResult) {
	fmt.Printf("\n=== Detailed Statistics ===\n")
	fmt.Printf("Input file size:       %d bytes\n", stats.InputSize)
	fmt.Printf("Extracted files:       %d\n", stats.TotalExtracted)
	fmt.Printf("Total extracted size:  %d bytes\n", stats.TotalSize)
	fmt.Printf("Data coverage:         %.2f%%\n", stats.Coverage)
	fmt.Printf("Overlaps detected:     %d\n", stats.Overlaps)

	if stats.Coverage < 90.0 {
		fmt.Printf("\nWarning: Low data coverage (%.2f%%). Possible issues with file detection.\n", stats.Coverage)
	}

	if float64(stats.TotalSize) > float64(stats.InputSize)*1.1 {
		fmt.Printf("\nWarning: Extracted data size (%.2f%%) exceeds input size. Possible overlaps or false positives.\n",
			float64(stats.TotalSize)/float64(stats.InputSize)*100)
	}

	fmt.Printf("\nFile types distribution:\n")
	for fileType, count := range stats.FileTypes {
		fmt.Printf("- %-30s: %d\n", fileType, count)
	}

	if len(stats.UncoveredAreas) > 0 {
		fmt.Printf("\nUncovered areas (total %d):\n", len(stats.UncoveredAreas))
		for i, area := range stats.UncoveredAreas {
			size := area.End - area.Start + 1
			if i < 10 || size > 1024 {
				fmt.Printf("- %8d - %8d (%6d bytes)\n", area.Start, area.End, size)
			}
			if i == 10 && len(stats.UncoveredAreas) > 10 {
				fmt.Printf("  ... and %d more uncovered areas\n", len(stats.UncoveredAreas)-10)
				break
			}
		}
	}

	var officeFiles, encryptedFiles, macroFiles int
	for _, res := range results {
		if res.OfficeInfo != nil {
			officeFiles++
			if res.OfficeInfo.IsEncrypted {
				encryptedFiles++
			}
			if res.OfficeInfo.IsMacro {
				macroFiles++
			}
		}
	}

	if officeFiles > 0 {
		fmt.Printf("\nOffice documents found: %d\n", officeFiles)
		fmt.Printf("- Encrypted: %d\n", encryptedFiles)
		fmt.Printf("- With macros: %d\n", macroFiles)
	}
}
