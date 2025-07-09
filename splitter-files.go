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
		Validator:   validateCompoundFile,
	},
	// DOCX (Office Open XML)
	{
		Extension:   "docx",
		MagicNumber: []byte{0x50, 0x4B, 0x03, 0x04},
		Offset:      0,
		Validator:   validateZipFile,
	},
	// PPT (Microsoft PowerPoint)
	{
		Extension:   "ppt",
		MagicNumber: []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1},
		Offset:      0,
		Validator:   validateCompoundFile,
	},
	// PPTX (Office Open XML Presentation)
	{
		Extension:   "pptx",
		MagicNumber: []byte{0x50, 0x4B, 0x03, 0x04},
		Offset:      0,
		Validator:   validateZipFile,
	},
	// XLS (Microsoft Excel)
	{
		Extension:   "xls",
		MagicNumber: []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1},
		Offset:      0,
		Validator:   validateCompoundFile,
	},
	// XLSX (Office Open XML Workbook)
	{
		Extension:   "xlsx",
		MagicNumber: []byte{0x50, 0x4B, 0x03, 0x04},
		Offset:      0,
		Validator:   validateZipFile,
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
	// TXT (простой текст, нет стандартной сигнатуры)
	{
		Extension: "txt",
		Validator: validateText,
	},
	// LNK (Windows Shortcut)
	{
		Extension:   "lnk",
		MagicNumber: []byte{0x4C, 0x00, 0x00, 0x00, 0x01, 0x14, 0x02, 0x00},
		Offset:      0,
	},
	// TMP (временный файл, нет стандартной сигнатуры)
	{
		Extension: "tmp",
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
		Validator:   validateZipFile,
	},
	// ZIP
	{
		Extension:   "zip",
		MagicNumber: []byte{0x50, 0x4B, 0x03, 0x04},
		Offset:      0,
		Validator:   validateZipFile,
	},
	// GZ
	{
		Extension:   "gz",
		MagicNumber: []byte{0x1F, 0x8B},
		Offset:      0,
	},
	// 7Z
	{
		Extension:   "7z",
		MagicNumber: []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C},
		Offset:      0,
	},
	// EXE (PE executable)
	{
		Extension:   "exe",
		MagicNumber: []byte{0x4D, 0x5A},
		Offset:      0,
		Validator:   validatePE,
	},
	// DLL (PE executable)
	{
		Extension:   "dll",
		MagicNumber: []byte{0x4D, 0x5A},
		Offset:      0,
		Validator:   validatePE,
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

// validateCompoundFile проверяет, является ли файл Compound File Binary Format
func validateCompoundFile(data []byte) bool {
	if len(data) < 8 {
		return false
	}
	// Проверяем сигнатуру
	if !bytes.Equal(data[:8], []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}) {
		return false
	}
	return true
}

// validateZipFile проверяет, является ли файл ZIP-архивом
func validateZipFile(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	// Проверяем сигнатуру
	if !bytes.Equal(data[:4], []byte{0x50, 0x4B, 0x03, 0x04}) {
		return false
	}
	return true
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

// validateText проверяет, является ли файл текстовым
func validateText(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	// Проверяем на наличие непечатных символов (кроме \t, \n, \r)
	for _, b := range data {
		if b < 32 && b != 9 && b != 10 && b != 13 {
			return false
		}
	}
	return true
}

// validatePE проверяет PE файл (EXE/DLL)
func validatePE(data []byte) bool {
	if len(data) < 0x40 {
		return false
	}
	// Проверяем сигнатуру MZ
	if !bytes.Equal(data[:2], []byte{0x4D, 0x5A}) {
		return false
	}
	// Получаем смещение PE заголовка
	peOffset := int(data[0x3C]) | int(data[0x3D])<<8 | int(data[0x3E])<<16 | int(data[0x3F])<<24
	if peOffset+4 > len(data) {
		return false
	}
	// Проверяем сигнатуру PE
	if !bytes.Equal(data[peOffset:peOffset+4], []byte{0x50, 0x45, 0x00, 0x00}) {
		return false
	}
	return true
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

	// Ограничиваем максимальный размер файла (чтобы не выйти за пределы данных)
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
