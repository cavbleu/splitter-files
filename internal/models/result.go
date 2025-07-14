package models

// ExtractionResult contains the result of file extraction
type ExtractionResult struct {
	Filename   string
	Size       int
	Start      int
	End        int
	Counter    int32
	Error      error
	FileType   string
	OfficeInfo *OfficeDocumentInfo
}

type ExtractionStats struct {
	TotalExtracted int
	TotalSize      int64
	InputSize      int64
	Overlaps       int
	Coverage       float64
	UncoveredAreas []struct {
		Start int
		End   int
	}
	FileTypes map[string]int
}
