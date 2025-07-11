package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	Version = "1.1.3"
)

// FileSignature defines a file signature and its extension
type FileSignature struct {
	Extension   string
	MagicNumber []byte
	Offset      int
	Validator   func([]byte) bool
}

// OfficeFileType defines the type of Office document
type OfficeFileType int

const (
	UnknownOffice OfficeFileType = iota
	WordDocument
	ExcelDocument
	PowerPointDocument
)

// OfficeDocumentInfo contains information about an Office document
type OfficeDocumentInfo struct {
	Type        OfficeFileType
	Version     string
	IsEncrypted bool
	IsMacro     bool
}

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

// ExtractionStats contains extraction statistics
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

// FileProcessor interface defines the contract for file processing
type FileProcessor interface {
	Process(data []byte, outputDir string, counter int32, startPos int, allowedExtensions map[string]bool) (int, int, string, string, *OfficeDocumentInfo, error)
}

// DefaultFileProcessor implements the basic file processing
type DefaultFileProcessor struct{}

func (p *DefaultFileProcessor) Process(data []byte, outputDir string, counter int32, startPos int, allowedExtensions map[string]bool) (int, int, string, string, *OfficeDocumentInfo, error) {
	return extractFile(data, outputDir, counter, startPos, allowedExtensions)
}

// FileValidator interface defines the contract for file validation
type FileValidator interface {
	Validate(data []byte) bool
}

// MSOfficeValidator validates Microsoft Office files
type MSOfficeValidator struct{}

func (v *MSOfficeValidator) Validate(data []byte) bool {
	return validateMSOfficeFile(data)
}

// OfficeOpenXMLValidator validates Office Open XML files
type OfficeOpenXMLValidator struct {
	expectedContent string
	expectedType    OfficeFileType
}

func (v *OfficeOpenXMLValidator) Validate(data []byte) bool {
	return validateOfficeOpenXML(v.expectedContent, v.expectedType)(data)
}

// JPEGValidator validates JPEG files with improved checking
type JPEGValidator struct{}

func (v *JPEGValidator) Validate(data []byte) bool {
	return validateJpegImproved(data)
}

// Strategy pattern for file processing
type ProcessingStrategy interface {
	Execute(data []byte, outputDir string, counter int32, startPos int, allowedExtensions map[string]bool) (int, int, string, string, *OfficeDocumentInfo, error)
}

type DefaultProcessingStrategy struct {
	processor FileProcessor
}

func (s *DefaultProcessingStrategy) Execute(data []byte, outputDir string, counter int32, startPos int, allowedExtensions map[string]bool) (int, int, string, string, *OfficeDocumentInfo, error) {
	return s.processor.Process(data, outputDir, counter, startPos, allowedExtensions)
}

// FileChunk represents a file chunk for processing
type FileChunk struct {
	Data     []byte
	Start    int
	Counter  int32
	Priority int
}

// File signatures with validators
var fileSignatures = []FileSignature{
	// DOC (Microsoft Word Document)
	{
		Extension:   "doc",
		MagicNumber: []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1},
		Offset:      0,
		Validator:   validateMSOfficeFile,
	},
	// DOCX (Office Open XML)
	{
		Extension:   "docx",
		MagicNumber: []byte{0x50, 0x4B, 0x03, 0x04},
		Offset:      0,
		Validator:   validateOfficeOpenXML("word/", WordDocument),
	},
	// PPT (Microsoft PowerPoint)
	{
		Extension:   "ppt",
		MagicNumber: []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1},
		Offset:      0,
		Validator:   validateMSOfficeFile,
	},
	// PPTX (Office Open XML Presentation)
	{
		Extension:   "pptx",
		MagicNumber: []byte{0x50, 0x4B, 0x03, 0x04},
		Offset:      0,
		Validator:   validateOfficeOpenXML("ppt/", PowerPointDocument),
	},
	// XLS (Microsoft Excel)
	{
		Extension:   "xls",
		MagicNumber: []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1},
		Offset:      0,
		Validator:   validateMSOfficeFile,
	},
	// XLSX (Office Open XML Workbook)
	{
		Extension:   "xlsx",
		MagicNumber: []byte{0x50, 0x4B, 0x03, 0x04},
		Offset:      0,
		Validator:   validateOfficeOpenXML("xl/", ExcelDocument),
	},
	// JPEG (improved validation)
	{
		Extension:   "jpg",
		MagicNumber: []byte{0xFF, 0xD8, 0xFF},
		Offset:      0,
		Validator:   validateJpegImproved,
	},
	{
		Extension:   "jpeg",
		MagicNumber: []byte{0xFF, 0xD8, 0xFF},
		Offset:      0,
		Validator:   validateJpegImproved,
	},
	// PDF (improved validation)
	{
		Extension:   "pdf",
		MagicNumber: []byte{0x25, 0x50, 0x44, 0x46},
		Offset:      0,
		Validator:   validatePdf,
	},
	// RTF (Rich Text Format)
	{
		Extension:   "rtf",
		MagicNumber: []byte{0x7B, 0x5C, 0x72, 0x74, 0x66, 0x31},
		Offset:      0,
	},
	// ODT (OpenDocument Text)
	{
		Extension:   "odt",
		MagicNumber: []byte{0x50, 0x4B, 0x03, 0x04},
		Offset:      0,
		Validator:   validateOpenDocument,
	},
	// ODS (OpenDocument Spreadsheet)
	{
		Extension:   "ods",
		MagicNumber: []byte{0x50, 0x4B, 0x03, 0x04},
		Offset:      0,
		Validator:   validateOpenDocument,
	},
	// OTS (OpenDocument Spreadsheet Template)
	{
		Extension:   "ots",
		MagicNumber: []byte{0x50, 0x4B, 0x03, 0x04},
		Offset:      0,
		Validator:   validateOpenDocument,
	},
	// FODS (Flat XML OpenDocument Spreadsheet)
	{
		Extension:   "fods",
		MagicNumber: []byte{0x3C, 0x3F, 0x78, 0x6D, 0x6C, 0x20, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6F, 0x6E, 0x3D, 0x22, 0x31, 0x2E, 0x30, 0x22, 0x3F, 0x3E},
		Offset:      0,
	},
	// ODP (OpenDocument Presentation)
	{
		Extension:   "odp",
		MagicNumber: []byte{0x50, 0x4B, 0x03, 0x04},
		Offset:      0,
		Validator:   validateOpenDocument,
	},
	// ZIP
	{
		Extension:   "zip",
		MagicNumber: []byte{0x50, 0x4B, 0x03, 0x04},
		Offset:      0,
		Validator:   validateZipFile,
	},
	// HTML
	{
		Extension:   "html",
		MagicNumber: []byte{0x3C, 0x21, 0x44, 0x4F, 0x43, 0x54, 0x59, 0x50, 0x45, 0x20, 0x68, 0x74, 0x6D, 0x6C},
		Offset:      0,
	},
	{
		Extension:   "html",
		MagicNumber: []byte{0x3C, 0x68, 0x74, 0x6D, 0x6C},
		Offset:      0,
	},
	{
		Extension:   "html",
		MagicNumber: []byte{0x3C, 0x48, 0x54, 0x4D, 0x4C},
		Offset:      0,
	},
}

var (
	versionFlag    = flag.Bool("version", false, "Print version information")
	extensionsFlag = flag.String("ext", "", "Comma-separated list of file extensions to extract (e.g. 'pdf,jpg,docx')")
)

// ContentTypes represents [Content_Types].xml in Office Open XML
type ContentTypes struct {
	XMLName xml.Name `xml:"Types"`
	Default []struct {
		Extension   string `xml:"Extension,attr"`
		ContentType string `xml:"ContentType,attr"`
	} `xml:"Default"`
	Override []struct {
		PartName    string `xml:"PartName,attr"`
		ContentType string `xml:"ContentType,attr"`
	} `xml:"Override"`
}

// WorkerPool manages worker goroutines
type WorkerPool struct {
	numWorkers int
	jobs       chan FileChunk
	results    chan ExtractionResult
	wg         *sync.WaitGroup
}

func NewWorkerPool(numWorkers int) *WorkerPool {
	return &WorkerPool{
		numWorkers: numWorkers,
		jobs:       make(chan FileChunk, numWorkers*2),
		results:    make(chan ExtractionResult, numWorkers*2),
		wg:         &sync.WaitGroup{},
	}
}

func (wp *WorkerPool) Start(outputDir string, allowedExtensions map[string]bool) {
	for i := 0; i < wp.numWorkers; i++ {
		wp.wg.Add(1)
		go worker(i, wp.jobs, wp.results, outputDir, wp.wg, allowedExtensions)
	}
}

func (wp *WorkerPool) Stop() {
	close(wp.jobs)
	wp.wg.Wait()
	close(wp.results)
}

// Improved JPEG validation
func validateJpegImproved(data []byte) bool {
	// Minimum JPEG size is about 4 bytes
	if len(data) < 4 {
		return false
	}

	// Check for Start of Image (SOI) marker
	if !bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}) {
		return false
	}

	// Check for End of Image (EOI) marker
	hasEOI := false
	for i := len(data) - 2; i >= 0; i-- {
		if data[i] == 0xFF && data[i+1] == 0xD9 {
			hasEOI = true
			break
		}
	}
	if !hasEOI {
		return false
	}
	return true
}

func validateMSOfficeFile(data []byte) bool {
	if len(data) < 8 {
		return false
	}

	if !bytes.Equal(data[:8], []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}) {
		return false
	}

	if len(data) > 512 {
		hasWordDocument := bytes.Contains(data, []byte("WordDocument"))
		hasWorkbook := bytes.Contains(data, []byte("Workbook"))
		hasPowerPoint := bytes.Contains(data, []byte("PowerPoint"))

		return hasWordDocument || hasWorkbook || hasPowerPoint
	}

	return true
}

func validateOfficeOpenXML(expectedContent string, expectedType OfficeFileType) func([]byte) bool {
	return func(data []byte) bool {
		if !validateZipFile(data) {
			return false
		}

		zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return false
		}

		var contentTypes ContentTypes
		var hasContentTypes bool
		var officeInfo OfficeDocumentInfo

		for _, file := range zipReader.File {
			switch file.Name {
			case "[Content_Types].xml":
				rc, err := file.Open()
				if err != nil {
					continue
				}
				defer rc.Close()

				contentData, err := ioutil.ReadAll(rc)
				if err != nil {
					continue
				}

				err = xml.Unmarshal(contentData, &contentTypes)
				if err == nil {
					hasContentTypes = true

					for _, override := range contentTypes.Override {
						switch {
						case strings.Contains(override.ContentType, "wordprocessing"):
							officeInfo.Type = WordDocument
						case strings.Contains(override.ContentType, "spreadsheet"):
							officeInfo.Type = ExcelDocument
						case strings.Contains(override.ContentType, "presentation"):
							officeInfo.Type = PowerPointDocument
						}
					}
				}

			case "docProps/app.xml":
				rc, err := file.Open()
				if err != nil {
					continue
				}
				defer rc.Close()

				appData, err := ioutil.ReadAll(rc)
				if err != nil {
					continue
				}

				if bytes.Contains(appData, []byte("VBAProject")) {
					officeInfo.IsMacro = true
				}

			case "docProps/core.xml":
				rc, err := file.Open()
				if err != nil {
					continue
				}
				defer rc.Close()

				coreData, err := ioutil.ReadAll(rc)
				if err != nil {
					continue
				}

				if officeInfo.Type == WordDocument || officeInfo.Type == ExcelDocument || officeInfo.Type == PowerPointDocument {
					hasEncryptedMarker := bytes.Contains(coreData, []byte("E\x00n\x00c\x00r\x00y\x00p\x00t\x00"))
					hasEncryptionInfo := bytes.Contains(coreData, []byte("E\x00n\x00c\x00r\x00y\x00p\x00t\x00i\x00o\x00n\x00I\x00n\x00f\x00o"))
					officeInfo.IsEncrypted = hasEncryptedMarker || hasEncryptionInfo
				}

				if re := regexp.MustCompile(`<cp:revision>(\d+)</cp:revision>`); re.Match(coreData) {
					matches := re.FindSubmatch(coreData)
					if len(matches) > 1 {
						officeInfo.Version = string(matches[1])
					}
				}
			}
		}

		if !hasContentTypes {
			return false
		}

		if officeInfo.Type != expectedType {
			return false
		}

		for _, defaultType := range contentTypes.Default {
			if strings.Contains(defaultType.ContentType, expectedContent) {
				return true
			}
		}

		return false
	}
}

func validateZipFile(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	return bytes.Equal(data[:4], []byte{0x50, 0x4B, 0x03, 0x04})
}

func validateOpenDocument(data []byte) bool {
	if !validateZipFile(data) {
		if bytes.HasPrefix(data, []byte("<?xml version=\"1.0\"?>")) {
			return bytes.Contains(data, []byte("office:document"))
		}
		return false
	}

	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return false
	}

	var hasMimetype, hasContent bool
	var mimeType string

	for _, file := range zipReader.File {
		switch file.Name {
		case "mimetype":
			rc, err := file.Open()
			if err != nil {
				continue
			}
			defer rc.Close()

			mimeData, err := ioutil.ReadAll(rc)
			if err != nil {
				continue
			}

			mimeType = string(mimeData)
			switch {
			case strings.Contains(mimeType, "application/vnd.oasis.opendocument.text"):
				hasMimetype = true
			case strings.Contains(mimeType, "application/vnd.oasis.opendocument.spreadsheet"):
				hasMimetype = true
			case strings.Contains(mimeType, "application/vnd.oasis.opendocument.presentation"):
				hasMimetype = true
			case strings.Contains(mimeType, "application/vnd.oasis.opendocument.spreadsheet-template"):
				hasMimetype = true
			}

		case "content.xml", "styles.xml":
			hasContent = true
		}
	}

	return hasMimetype && hasContent
}

func validateJpeg(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	if !bytes.Equal(data[:3], []byte{0xFF, 0xD8, 0xFF}) {
		return false
	}
	if len(data) >= 20 {
		for i := len(data) - 2; i >= 0; i-- {
			if data[i] == 0xFF && data[i+1] == 0xD9 {
				return true
			}
		}
	}
	return false
}

func validatePdf(data []byte) bool {
	if len(data) < 100 {
		return false
	}

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return false
	}

	if len(data) >= 8 {
		version := string(data[5:8])
		if version < "1.0" || version > "2.0" {
			return false
		}
	}

	if !bytes.Contains(data, []byte("xref")) {
		return false
	}

	if !bytes.Contains(data, []byte(" 0 obj")) && !bytes.Contains(data, []byte("\n0 obj")) {
		return false
	}

	eofPos := bytes.LastIndex(data, []byte("%%EOF"))
	if eofPos == -1 {
		return false
	}

	if eofPos+5 < len(data) {
		trailer := data[eofPos+5]
		if trailer != '\r' && trailer != '\n' && trailer != ' ' && trailer != '\t' {
			return false
		}
	}

	if !bytes.Contains(data[:eofPos], []byte("startxref")) {
		return false
	}

	return true
}

func findFileSignatures(data []byte, allowedExtensions map[string]bool) []FileSignature {
	var found []FileSignature

	for _, sig := range fileSignatures {
		// Skip if extension not in allowed list
		if len(allowedExtensions) > 0 && !allowedExtensions[sig.Extension] {
			continue
		}

		if len(sig.MagicNumber) == 0 {
			continue
		}

		offset := sig.Offset
		end := offset + len(sig.MagicNumber)

		if end > len(data) {
			continue
		}

		if bytes.Equal(data[offset:end], sig.MagicNumber) {
			if sig.Validator != nil {
				if !sig.Validator(data) {
					continue
				}
			}
			found = append(found, sig)
		}
	}

	return found
}

func extractFile(data []byte, outputDir string, counter int32, startPos int, allowedExtensions map[string]bool) (int, int, string, string, *OfficeDocumentInfo, error) {
	const minFileSize = 2 * 1024

	foundSigs := findFileSignatures(data, allowedExtensions)
	if len(foundSigs) == 0 {
		return 0, 0, "", "", nil, errors.New("no known file signatures found")
	}

	sig := foundSigs[0]
	ext := sig.Extension
	fileType := strings.ToUpper(ext)

	var officeInfo *OfficeDocumentInfo

	if strings.HasPrefix(ext, "doc") || strings.HasPrefix(ext, "xls") || strings.HasPrefix(ext, "ppt") {
		officeInfo = &OfficeDocumentInfo{}

		if ext == "doc" || ext == "xls" || ext == "ppt" {
			if bytes.Contains(data, []byte("WordDocument")) {
				officeInfo.Type = WordDocument
			} else if bytes.Contains(data, []byte("Workbook")) {
				officeInfo.Type = ExcelDocument
			} else if bytes.Contains(data, []byte("PowerPoint")) {
				officeInfo.Type = PowerPointDocument
			}

			if bytes.Contains(data, []byte("_VBA_PROJECT")) {
				officeInfo.IsMacro = true
			}

			if officeInfo.Type == WordDocument || officeInfo.Type == ExcelDocument || officeInfo.Type == PowerPointDocument {
				hasEncryptedMarker := bytes.Contains(data, []byte("E\x00n\x00c\x00r\x00y\x00p\x00t\x00"))
				hasEncryptionHeader := false
				if len(data) > 512 {
					if bytes.HasPrefix(data[512:], []byte{0xFE, 0xFF, 0xFF, 0xFF}) {
						hasEncryptionHeader = true
					}
					if bytes.Contains(data[:512], []byte("E\x00n\x00c\x00r\x00y\x00p\x00t\x00P\x00a\x00c\x00k\x00a\x00g\x00e")) {
						hasEncryptionHeader = true
					}
				}

				isEncrypted := false
				if len(data) > 0x200 {
					var protectionFlagOffset int
					switch officeInfo.Type {
					case WordDocument:
						protectionFlagOffset = 0x0B
					case ExcelDocument:
						protectionFlagOffset = 0x2F
					case PowerPointDocument:
						protectionFlagOffset = 0x0F
					}

					if protectionFlagOffset > 0 && len(data) > protectionFlagOffset {
						protectionFlag := data[protectionFlagOffset]
						isEncrypted = (protectionFlag & 0x01) != 0
					}
				}

				hasEncryptionStream := bytes.Contains(data, []byte("E\x00n\x00c\x00r\x00y\x00p\x00t\x00i\x00o\x00n\x00I\x00n\x00f\x00o"))

				officeInfo.IsEncrypted = hasEncryptedMarker || hasEncryptionHeader || isEncrypted || hasEncryptionStream

				if officeInfo.IsMacro && bytes.Contains(data, []byte("D\x00e\x00f\x00a\x00u\x00l\x00t\x00P\x00a\x00s\x00s\x00w\x00o\x00r\x00d")) {
					officeInfo.IsEncrypted = true
				}
			}
		}
	}

	fileEnd := len(data)
	for i := 1; i < len(fileSignatures); i++ {
		otherSig := fileSignatures[i]
		if len(otherSig.MagicNumber) == 0 {
			continue
		}

		idx := bytes.Index(data, otherSig.MagicNumber)
		if idx != -1 && idx < fileEnd && idx > 0 {
			fileEnd = idx
		}
	}

	switch ext {
	case "jpg", "jpeg":
		// Improved JPEG end detection
		for i := len(data) - 2; i >= 0; i-- {
			if data[i] == 0xFF && data[i+1] == 0xD9 { // EOI marker
				fileEnd = i + 2
				break
			}
		}
		fileType = "JPEG Image"
	case "pdf":
		if idx := bytes.LastIndex(data, []byte("%%EOF")); idx != -1 {
			fileEnd = idx + 5
			if fileEnd < len(data) {
				trailer := data[fileEnd]
				if trailer == '\r' || trailer == '\n' {
					fileEnd++
				} else if fileEnd+1 < len(data) && data[fileEnd] == '\r' && data[fileEnd+1] == '\n' {
					fileEnd += 2
				}
			}
		}
		fileType = "PDF Document"
	case "zip", "docx", "xlsx", "pptx", "odt":
		if idx := bytes.LastIndex(data, []byte{0x50, 0x4B, 0x05, 0x06}); idx != -1 {
			fileEnd = idx + 22
		}

		switch ext {
		case "docx":
			fileType = "Word Document (Open XML)"
		case "xlsx":
			fileType = "Excel Workbook (Open XML)"
		case "pptx":
			fileType = "PowerPoint Presentation (Open XML)"
		case "odt":
			fileType = "OpenDocument Text"
		case "zip":
			fileType = "ZIP Archive"
		case "ods", "ots":
			fileType = "OpenDocument Spreadsheet"
		case "fods":
			fileType = "Flat XML OpenDocument Spreadsheet"
		case "odp":
			fileType = "OpenDocument Presentation"
		}
	case "doc":
		fileType = "Word Document (Binary)"
	case "xls":
		fileType = "Excel Workbook (Binary)"
	case "ppt":
		fileType = "PowerPoint Presentation (Binary)"
	case "rtf":
		fileType = "Rich Text Format"
	case "html":
		fileType = "HTML Document"
	}

	if fileEnd == len(data) {
		if len(data) > 100 {
			nextSig := bytes.Index(data[1:], sig.MagicNumber)
			if nextSig != -1 {
				fileEnd = nextSig + 1
			}
		}
	}

	if fileEnd > len(data) {
		fileEnd = len(data)
	}

	if fileEnd < minFileSize {
		return 0, 0, "", "", nil, fmt.Errorf("file too small (less than %d bytes)", minFileSize)
	}

	fileData := data[:fileEnd]

	filename := filepath.Join(outputDir, fmt.Sprintf("file_%04d.%s", counter, ext))
	err := ioutil.WriteFile(filename, fileData, 0644)
	if err != nil {
		return 0, 0, "", "", nil, fmt.Errorf("failed to write file %s: %v", filename, err)
	}

	return fileEnd, startPos + fileEnd, filename, fileType, officeInfo, nil
}

func worker(id int, jobs <-chan FileChunk, results chan<- ExtractionResult, outputDir string, wg *sync.WaitGroup, allowedExtensions map[string]bool) {
	defer wg.Done()

	processor := &DefaultFileProcessor{}
	strategy := &DefaultProcessingStrategy{processor: processor}

	for chunk := range jobs {
		size, endPos, filename, fileType, officeInfo, err := strategy.Execute(chunk.Data, outputDir, chunk.Counter, chunk.Start, allowedExtensions)
		if err != nil {
			results <- ExtractionResult{
				Error:   fmt.Errorf("worker %d: %v", id, err),
				Counter: chunk.Counter,
			}
			continue
		}

		results <- ExtractionResult{
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

func getPhysicalCPUCount() int {
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" || runtime.GOOS == "freebsd" {
		data, err := ioutil.ReadFile("/proc/cpuinfo")
		if err == nil {
			physicalIDs := make(map[string]bool)
			re := regexp.MustCompile(`physical id\s*:\s*(\d+)`)
			matches := re.FindAllStringSubmatch(string(data), -1)
			for _, match := range matches {
				if len(match) > 1 {
					physicalIDs[match[1]] = true
				}
			}

			if len(physicalIDs) > 0 {
				coresPerCPU := make(map[string]int)
				reCore := regexp.MustCompile(`physical id\s*:\s*(\d+).*?cpu cores\s*:\s*(\d+)`)
				matchesCore := reCore.FindAllStringSubmatch(string(data), -1)
				for _, match := range matchesCore {
					if len(match) > 2 {
						coresPerCPU[match[1]] = coresPerCPU[match[1]] + 1
					}
				}

				totalCores := 0
				for _, cores := range coresPerCPU {
					totalCores += cores
				}

				if totalCores > 0 {
					return totalCores
				}
			}
		}
	}

	return runtime.NumCPU()
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

func processFile(data []byte, outputDir string, numWorkers int, allowedExtensions map[string]bool) (int, []ExtractionResult, *ExtractionStats, error) {
	wp := NewWorkerPool(numWorkers)
	wp.Start(outputDir, allowedExtensions)

	stats := &ExtractionStats{
		InputSize: int64(len(data)),
		FileTypes: make(map[string]int),
	}

	var extractedFiles int32
	var allResults []ExtractionResult
	var processingErrors []error
	var resultWg sync.WaitGroup
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
			allResults = append(allResults, result)
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
				case WordDocument:
					officeType = "Word"
				case ExcelDocument:
					officeType = "Excel"
				case PowerPointDocument:
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
			foundSigs := findFileSignatures(remaining, allowedExtensions)
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
		return int(extractedFiles), allResults, stats, fmt.Errorf("encountered %d processing errors", len(processingErrors))
	}

	return int(extractedFiles), allResults, stats, nil
}

func printStats(stats *ExtractionStats) {
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
			if i < 10 || size > 1024 { // Show first 10 or large areas
				fmt.Printf("- %8d - %8d (%6d bytes)\n", area.Start, area.End, size)
			}
			if i == 10 && len(stats.UncoveredAreas) > 10 {
				fmt.Printf("  ... and %d more uncovered areas\n", len(stats.UncoveredAreas)-10)
				break
			}
		}
	}
}

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Printf("File Splitter version %s\n", Version)
		os.Exit(0)
	}

	if len(flag.Args()) < 2 {
		fmt.Println("File Splitter - tool for extracting embedded files from binary data.")
		fmt.Println("Version:", Version)
		fmt.Println("\nUsage: file-splitter [flags] <input_file> <output_directory> [num_workers]")
		fmt.Println("\nFlags:")
		flag.PrintDefaults()
		fmt.Println("\nSupported file extensions: doc, docx, ppt, pptx, xls, xlsx, jpg, jpeg, pdf, rtf, odt, ods, odp, ots, fods, zip, html")
		fmt.Println("\nExamples:")
		fmt.Println("  file-splitter -ext pdf,jpg,docx data.bin output_dir")
		fmt.Println("  file-splitter -ext all data.bin output_dir 8")
		fmt.Println("\nDefault number of workers is equal to physical CPU cores")
		os.Exit(1)
	}

	args := flag.Args()
	inputFile := args[0]
	outputDir := args[1]

	// Parse allowed extensions
	allowedExtensions := make(map[string]bool)
	if *extensionsFlag != "" {
		if *extensionsFlag == "all" {
			// Allow all extensions
			for _, sig := range fileSignatures {
				allowedExtensions[sig.Extension] = true
			}
		} else {
			exts := strings.Split(*extensionsFlag, ",")
			for _, ext := range exts {
				ext = strings.TrimSpace(strings.ToLower(ext))
				if ext != "" {
					allowedExtensions[ext] = true
				}
			}
		}
	}

	numWorkers := getPhysicalCPUCount()
	if len(args) > 2 {
		_, err := fmt.Sscanf(args[2], "%d", &numWorkers)
		if err != nil || numWorkers < 1 {
			fmt.Printf("Invalid number of workers, using default (%d physical cores)\n", numWorkers)
		}
	}

	data, err := ioutil.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("Error reading input file: %v\n", err)
		os.Exit(1)
	}

	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Processing file %s (%d bytes) with %d workers (physical cores)\n",
		inputFile, len(data), numWorkers)
	if len(allowedExtensions) > 0 {
		fmt.Printf("Extracting only files with extensions: %v\n", getKeys(allowedExtensions))
	}

	startTime := time.Now()
	_, results, stats, err := processFile(data, outputDir, numWorkers, allowedExtensions)
	elapsed := time.Since(startTime)

	if err != nil {
		fmt.Printf("Processing completed with errors: %v\n", err)
	}

	printStats(stats)

	var officeFiles int
	var encryptedFiles int
	var macroFiles int
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

	fmt.Printf("\nProcessing completed in %s\n", elapsed)
}

func getKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
