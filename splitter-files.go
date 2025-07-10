package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

// FileSignature определяет сигнатуру файла и его расширение
type FileSignature struct {
	Extension   string
	MagicNumber []byte
	Offset      int
	Validator   func([]byte) bool
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
		Validator:   validateOfficeOpenXML("word/"),
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
		Validator:   validateOfficeOpenXML("ppt/"),
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
		Validator:   validateOfficeOpenXML("xl/"),
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
	// PDF
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

// validateMSOfficeFile проверяет, является ли файл документом MS Office (DOC, PPT, XLS)
func validateMSOfficeFile(data []byte) bool {
	if len(data) < 8 {
		return false
	}
	return bytes.Equal(data[:8], []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1})
}

// validateOfficeOpenXML возвращает функцию для проверки конкретного типа Office Open XML файла
func validateOfficeOpenXML(contentType string) func([]byte) bool {
	return func(data []byte) bool {
		if !validateZipFile(data) {
			return false
		}

		// Проверяем наличие обязательных файлов в структуре Office Open XML
		requiredFiles := map[string]bool{
			"[Content_Types].xml": false,
		}

		// Проверяем наличие специфичного для типа документа содержимого
		contentFound := false

		// Упрощенная проверка - ищем сигнатуры ключевых файлов
		for file := range requiredFiles {
			if bytes.Contains(data, []byte(file)) {
				requiredFiles[file] = true
			}
		}

		// Проверяем наличие специфичного контента
		if bytes.Contains(data, []byte(contentType)) {
			contentFound = true
		}

		// Все обязательные файлы должны присутствовать и должен быть соответствующий контент
		for _, found := range requiredFiles {
			if !found {
				return false
			}
		}
		return contentFound
	}
}

// validateZipFile проверяет, является ли файл ZIP-архивом
func validateZipFile(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	return bytes.Equal(data[:4], []byte{0x50, 0x4B, 0x03, 0x04})
}

// validateOpenDocument проверяет OpenDocument файлы
func validateOpenDocument(data []byte) bool {
	if !validateZipFile(data) {
		return false
	}
	return bytes.Contains(data, []byte("mimetype")) &&
		bytes.Contains(data, []byte("content.xml"))
}

// validateJpeg проверяет JPEG файл
func validateJpeg(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	// Начало JPEG
	if !bytes.Equal(data[:3], []byte{0xFF, 0xD8, 0xFF}) {
		return false
	}
	// Проверяем маркер окончания
	if len(data) >= 20 {
		for i := len(data) - 2; i >= 0; i-- {
			if data[i] == 0xFF && data[i+1] == 0xD9 {
				return true
			}
		}
	}
	return false
}

// validatePdf проверяет PDF файл
func validatePdf(data []byte) bool {
	if len(data) < 8 {
		return false
	}
	// Начало PDF
	if !bytes.Equal(data[:4], []byte{0x25, 0x50, 0x44, 0x46}) {
		return false
	}
	// Проверяем конец файла
	if len(data) >= 8 {
		tail := data[len(data)-8:]
		if bytes.Contains(tail, []byte("%%EOF")) {
			return true
		}
	}
	return false
}

// findFileSignatures ищет все известные сигнатуры в данных
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
			// Если есть валидатор, проверяем файл
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

// FileChunk представляет часть данных для обработки
type FileChunk struct {
	Data    []byte
	Start   int
	Counter int32
}

// ExtractionResult содержит результат извлечения файла
type ExtractionResult struct {
	Filename string
	Size     int
	Counter  int32
	Error    error
}

// extractFile пытается извлечь файл из данных
func extractFile(data []byte, outputDir string, counter int32) (int, string, error) {
	foundSigs := findFileSignatures(data)
	if len(foundSigs) == 0 {
		return 0, "", errors.New("no known file signatures found")
	}

	// Выбираем первую подходящую сигнатуру
	sig := foundSigs[0]
	ext := sig.Extension

	// Определяем длину файла (эвристически)
	fileEnd := len(data)
	for i := 1; i < len(fileSignatures); i++ {
		otherSig := fileSignatures[i]
		if len(otherSig.MagicNumber) == 0 {
			continue
		}

		// Ищем следующую сигнатуру
		idx := bytes.Index(data, otherSig.MagicNumber)
		if idx != -1 && idx < fileEnd && idx > 0 {
			fileEnd = idx
		}
	}

	// Для некоторых форматов определяем конец файла специальным образом
	switch ext {
	case "jpg", "jpeg":
		// Ищем маркер конца JPEG
		for i := len(data) - 2; i >= 0; i-- {
			if data[i] == 0xFF && data[i+1] == 0xD9 {
				fileEnd = i + 2
				break
			}
		}
	case "pdf":
		// Ищем конец PDF
		if idx := bytes.LastIndex(data, []byte("%%EOF")); idx != -1 {
			fileEnd = idx + 5
		}
	case "zip", "docx", "xlsx", "pptx", "odt":
		// Для ZIP-подобных файлов ищем конец центрального каталога
		if idx := bytes.LastIndex(data, []byte{0x50, 0x4B, 0x05, 0x06}); idx != -1 {
			fileEnd = idx + 22 // 22 - размер End of central directory record
		}
	}

	// Если не удалось определить конец файла, берем до следующей сигнатуры или до конца данных
	if fileEnd == len(data) {
		// Ищем следующую сигнатуру такого же типа
		if len(data) > 100 {
			nextSig := bytes.Index(data[1:], sig.MagicNumber)
			if nextSig != -1 {
				fileEnd = nextSig + 1
			}
		}
	}

	// Ограничиваем максимальный размер файла
	if fileEnd > len(data) {
		fileEnd = len(data)
	}

	// Если файл слишком маленький, пропускаем
	if fileEnd < 10 {
		return 0, "", errors.New("file too small")
	}

	fileData := data[:fileEnd]

	// Создаем имя файла
	filename := filepath.Join(outputDir, fmt.Sprintf("file_%04d.%s", counter, ext))
	err := ioutil.WriteFile(filename, fileData, 0644)
	if err != nil {
		return 0, "", fmt.Errorf("failed to write file %s: %v", filename, err)
	}

	return fileEnd, filename, nil
}

// worker обрабатывает задачи из канала и отправляет результаты
func worker(id int, jobs <-chan FileChunk, results chan<- ExtractionResult, outputDir string, wg *sync.WaitGroup) {
	defer wg.Done()
	for chunk := range jobs {
		processed, filename, err := extractFile(chunk.Data, outputDir, chunk.Counter)
		if err != nil {
			results <- ExtractionResult{
				Error:   fmt.Errorf("worker %d: %v", id, err),
				Counter: chunk.Counter,
			}
			continue
		}

		results <- ExtractionResult{
			Filename: filename,
			Size:     processed,
			Counter:  chunk.Counter,
		}
	}
}

// processFile обрабатывает входной файл многопоточно
func processFile(data []byte, outputDir string, numWorkers int) (int, error) {
	// Создаем каналы для задач и результатов
	jobs := make(chan FileChunk, numWorkers*2)
	results := make(chan ExtractionResult, numWorkers*2)

	// Создаем воркеров
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(i, jobs, results, outputDir, &wg)
	}

	// Запускаем горутину для сбора результатов
	var extractedFiles int32
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
			fmt.Printf("Extracted %s (%d bytes)\n", result.Filename, result.Size)
		}
	}()

	// Отправляем задачи воркерам
	pos := 0
	var counter int32 = 1
	for pos < len(data) {
		remaining := data[pos:]
		if len(remaining) < 8 { // Минимальный размер для любой сигнатуры
			break
		}

		// Создаем задачу для воркера
		chunk := FileChunk{
			Data:    remaining,
			Start:   pos,
			Counter: counter,
		}

		// Отправляем задачу (неблокирующе, если канал полон)
		select {
		case jobs <- chunk:
			pos++
			counter++
		default:
			// Если канал задач полон, ждем немного
			// В реальном приложении можно добавить более сложную логику
			continue
		}
	}

	// Закрываем канал задач и ждем завершения воркеров
	close(jobs)
	wg.Wait()

	// Закрываем канал результатов и ждем завершения сборщика результатов
	close(results)
	resultWg.Wait()

	// Проверяем ошибки
	if len(processingErrors) > 0 {
		return int(extractedFiles), fmt.Errorf("encountered %d processing errors", len(processingErrors))
	}

	return int(extractedFiles), nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: file_splitter <input_file> <output_directory> [num_workers]")
		fmt.Println("Default number of workers is 4")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputDir := os.Args[2]

	// Определяем количество воркеров
	numWorkers := 4
	if len(os.Args) > 3 {
		_, err := fmt.Sscanf(os.Args[3], "%d", &numWorkers)
		if err != nil || numWorkers < 1 {
			fmt.Println("Invalid number of workers, using default (4)")
			numWorkers = 4
		}
	}

	// Читаем входной файл
	data, err := ioutil.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("Error reading input file: %v\n", err)
		os.Exit(1)
	}

	// Создаем выходную директорию
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Processing file %s (%d bytes) with %d workers\n", inputFile, len(data), numWorkers)

	// Обрабатываем файл многопоточно
	extracted, err := processFile(data, outputDir, numWorkers)
	if err != nil {
		fmt.Printf("Processing completed with errors: %v\n", err)
	}

	fmt.Printf("Extracted %d files to %s\n", extracted, outputDir)
}
