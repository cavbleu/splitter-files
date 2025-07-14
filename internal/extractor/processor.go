package extractor

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"splitter-files/internal/models"
	"strings"
)

const (
	UnknownOffice OfficeFileType = iota
	WordDocument
	ExcelDocument
	PowerPointDocument
)

type OfficeFileType int

type FileProcessor interface {
	Process(data []byte, outputDir string, counter int32, startPos int, allowedExtensions map[string]bool) (int, int, string, string, *models.OfficeDocumentInfo, error)
}

type DefaultFileProcessor struct{}

func (p *DefaultFileProcessor) Process(data []byte, outputDir string, counter int32, startPos int, allowedExtensions map[string]bool) (int, int, string, string, *models.OfficeDocumentInfo, error) {
	return ExtractFile(data, outputDir, counter, startPos, allowedExtensions)
}

func ExtractFile(data []byte, outputDir string, counter int32, startPos int, allowedExtensions map[string]bool) (int, int, string, string, *models.OfficeDocumentInfo, error) {
	const minFileSize = 2 * 1024

	foundSigs := FindFileSignatures(data, allowedExtensions)
	if len(foundSigs) == 0 {
		return 0, 0, "", "", nil, errors.New("no known file signatures found")
	}

	sig := foundSigs[0]
	ext := sig.Extension
	fileType := strings.ToUpper(ext)

	var officeInfo *models.OfficeDocumentInfo

	if strings.HasPrefix(ext, "doc") || strings.HasPrefix(ext, "xls") || strings.HasPrefix(ext, "ppt") {
		officeInfo = &models.OfficeDocumentInfo{}

		if ext == "doc" || ext == "xls" || ext == "ppt" {
			if bytes.Contains(data, []byte("WordDocument")) {
				officeInfo.Type = models.WordDocument
			} else if bytes.Contains(data, []byte("Workbook")) {
				officeInfo.Type = models.ExcelDocument
			} else if bytes.Contains(data, []byte("PowerPoint")) {
				officeInfo.Type = models.PowerPointDocument
			}

			if bytes.Contains(data, []byte("_VBA_PROJECT")) {
				officeInfo.IsMacro = true
			}

			if officeInfo.Type == models.WordDocument || officeInfo.Type == models.ExcelDocument || officeInfo.Type == models.PowerPointDocument {
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
					case models.WordDocument:
						protectionFlagOffset = 0x0B
					case models.ExcelDocument:
						protectionFlagOffset = 0x2F
					case models.PowerPointDocument:
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
