package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
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

// extractFile пытается извлечь файл из данных
func extractFile(data []byte, outputDir string, counter int) (int, error) {
	foundSigs := findFileSignatures(data)
	if len(foundSigs) == 0 {
		return 0, errors.New("no known file signatures found")
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
		return 0, errors.New("file too small")
	}

	fileData := data[:fileEnd]

	// Создаем имя файла
	filename := fmt.Sprintf("%s/file_%04d.%s", outputDir, counter, ext)
	err := ioutil.WriteFile(filename, fileData, 0644)
	if err != nil {
		return 0, fmt.Errorf("failed to write file %s: %v", filename, err)
	}

	fmt.Printf("Extracted %s (%d bytes)\n", filename, len(fileData))
	return fileEnd, nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: file_splitter <input_file> <output_directory>")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputDir := os.Args[2]

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

	fmt.Printf("Processing file %s (%d bytes)\n", inputFile, len(data))

	// Обрабатываем файл
	pos := 0
	counter := 1
	for pos < len(data) {
		remaining := data[pos:]
		if len(remaining) < 8 { // Минимальный размер для любой сигнатуры
			break
		}

		processed, err := extractFile(remaining, outputDir, counter)
		if err != nil {
			// Пропускаем байт и продолжаем
			pos++
			continue
		}

		pos += processed
		counter++
	}

	fmt.Printf("Extracted %d files to %s\n", counter-1, outputDir)
}