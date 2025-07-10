package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
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

// FileSignature определяет сигнатуру файла и его расширение
type FileSignature struct {
	Extension   string
	MagicNumber []byte
	Offset      int
	Validator   func([]byte) bool
}

// OfficeFileType определяет тип Office-документа
type OfficeFileType int

const (
	UnknownOffice OfficeFileType = iota
	WordDocument
	ExcelDocument
	PowerPointDocument
)

// OfficeDocumentInfo содержит информацию об Office-документе
type OfficeDocumentInfo struct {
	Type        OfficeFileType
	Version     string
	IsEncrypted bool
	IsMacro     bool
}

// Сигнатуры файлов с валидаторами
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
	// JPEG
	{
		Extension:   "jpg",
		MagicNumber: []byte{0xFF, 0xD8, 0xFF},
		Offset:      0,
		Validator:   validateJpeg,
	},
	{
		Extension:   "jpeg",
		MagicNumber: []byte{0xFF, 0xD8, 0xFF},
		Offset:      0,
		Validator:   validateJpeg,
	},
	// PDF (улучшенная проверка)
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

// ContentTypes представляет структуру [Content_Types].xml в Office Open XML
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

				// Улучшенная проверка на шифрование
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
		return false
	}

	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return false
	}

	var hasMimetype, hasContent bool

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

			if bytes.Contains(mimeData, []byte("application/vnd.oasis.opendocument.text")) {
				hasMimetype = true
			}

		case "content.xml":
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

func findFileSignatures(data []byte) []FileSignature {
	var found []FileSignature

	for _, sig := range fileSignatures {
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

type FileChunk struct {
	Data     []byte
	Start    int
	Counter  int32
	Priority int
}

type ExtractionResult struct {
	Filename   string
	Size       int
	Counter    int32
	Error      error
	FileType   string
	OfficeInfo *OfficeDocumentInfo
}

func extractFile(data []byte, outputDir string, counter int32) (int, string, string, *OfficeDocumentInfo, error) {
	const minFileSize = 2 * 1024

	foundSigs := findFileSignatures(data)
	if len(foundSigs) == 0 {
		return 0, "", "", nil, errors.New("no known file signatures found")
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

			// Улучшенная проверка на шифрование для бинарных форматов
			if officeInfo.Type == WordDocument || officeInfo.Type == ExcelDocument || officeInfo.Type == PowerPointDocument {
				// 1. Проверка стандартного маркера
				hasEncryptedMarker := bytes.Contains(data, []byte("E\x00n\x00c\x00r\x00y\x00p\x00t\x00"))

				// 2. Проверка сигнатуры шифрования
				hasEncryptionHeader := false
				if len(data) > 512 {
					if bytes.HasPrefix(data[512:], []byte{0xFE, 0xFF, 0xFF, 0xFF}) {
						hasEncryptionHeader = true
					}
					if bytes.Contains(data[:512], []byte("E\x00n\x00c\x00r\x00y\x00p\x00t\x00P\x00a\x00c\x00k\x00a\x00g\x00e")) {
						hasEncryptionHeader = true
					}
				}

				// 3. Проверка флагов в заголовке
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

				// 4. Проверка наличия потоков шифрования
				hasEncryptionStream := bytes.Contains(data, []byte("E\x00n\x00c\x00r\x00y\x00p\x00t\x00i\x00o\x00n\x00I\x00n\x00f\x00o"))

				officeInfo.IsEncrypted = hasEncryptedMarker || hasEncryptionHeader || isEncrypted || hasEncryptionStream

				// 5. Дополнительная проверка для документов с макросами
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
		for i := len(data) - 2; i >= 0; i-- {
			if data[i] == 0xFF && data[i+1] == 0xD9 {
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
		return 0, "", "", nil, fmt.Errorf("file too small (less than %d bytes)", minFileSize)
	}

	fileData := data[:fileEnd]

	filename := filepath.Join(outputDir, fmt.Sprintf("file_%04d.%s", counter, ext))
	err := ioutil.WriteFile(filename, fileData, 0644)
	if err != nil {
		return 0, "", "", nil, fmt.Errorf("failed to write file %s: %v", filename, err)
	}

	return fileEnd, filename, fileType, officeInfo, nil
}

func worker(id int, jobs <-chan FileChunk, results chan<- ExtractionResult, outputDir string, wg *sync.WaitGroup) {
	defer wg.Done()
	for chunk := range jobs {
		processed, filename, fileType, officeInfo, err := extractFile(chunk.Data, outputDir, chunk.Counter)
		if err != nil {
			results <- ExtractionResult{
				Error:   fmt.Errorf("worker %d: %v", id, err),
				Counter: chunk.Counter,
			}
			continue
		}

		results <- ExtractionResult{
			Filename:   filename,
			Size:       processed,
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

func processFile(data []byte, outputDir string, numWorkers int) (int, []ExtractionResult, error) {
	jobs := make(chan FileChunk, numWorkers*2)
	results := make(chan ExtractionResult, numWorkers*2)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(i, jobs, results, outputDir, &wg)
	}

	var extractedFiles int32
	var allResults []ExtractionResult
	var processingErrors []error
	var resultWg sync.WaitGroup
	resultWg.Add(1)
	go func() {
		defer resultWg.Done()
		for result := range results {
			if result.Error != nil {
				processingErrors = append(processingErrors, result.Error)
				continue
			}
			atomic.AddInt32(&extractedFiles, 1)
			allResults = append(allResults, result)

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

				info := fmt.Sprintf("Extracted %s (%s, %d bytes)", result.Filename, officeType, result.Size)
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
				fmt.Printf("Extracted %s (%s, %d bytes)\n", result.Filename, result.FileType, result.Size)
			}
		}
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
			case jobs <- chunk:
				officeQueue = officeQueue[1:]
				counter++
			case <-time.After(backoffTime):
			}
			continue
		}

		if len(regularQueue) > 0 {
			chunk := regularQueue[0]
			select {
			case jobs <- chunk:
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
			foundSigs := findFileSignatures(remaining)
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

	close(jobs)
	wg.Wait()

	close(results)
	resultWg.Wait()

	if len(processingErrors) > 0 {
		return int(extractedFiles), allResults, fmt.Errorf("encountered %d processing errors", len(processingErrors))
	}

	return int(extractedFiles), allResults, nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: file_splitter <input_file> <output_directory> [num_workers]")
		fmt.Println("Default number of workers is equal to physical CPU cores")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputDir := os.Args[2]

	numWorkers := getPhysicalCPUCount()
	if len(os.Args) > 3 {
		_, err := fmt.Sscanf(os.Args[3], "%d", &numWorkers)
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

	fmt.Printf("Processing file %s (%d bytes) with %d workers (physical cores)\n", inputFile, len(data), numWorkers)

	extracted, results, err := processFile(data, outputDir, numWorkers)
	if err != nil {
		fmt.Printf("Processing completed with errors: %v\n", err)
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Extracted %d files to %s\n", extracted, outputDir)

	var officeFiles int
	for _, res := range results {
		if res.OfficeInfo != nil {
			officeFiles++
		}
	}

	if officeFiles > 0 {
		fmt.Printf("\nOffice documents found: %d\n", officeFiles)
		for _, res := range results {
			if res.OfficeInfo != nil {
				var docType string
				switch res.OfficeInfo.Type {
				case WordDocument:
					docType = "Word"
				case ExcelDocument:
					docType = "Excel"
				case PowerPointDocument:
					docType = "PowerPoint"
				default:
					docType = "Unknown"
				}

				info := fmt.Sprintf("- %s (%s)", filepath.Base(res.Filename), docType)
				if res.OfficeInfo.IsEncrypted {
					info += " [ENCRYPTED]"
				}
				if res.OfficeInfo.IsMacro {
					info += " [MACROS]"
				}
				if res.OfficeInfo.Version != "" {
					info += fmt.Sprintf(" [v%s]", res.OfficeInfo.Version)
				}

				fmt.Println(info)
			}
		}
	}
}
