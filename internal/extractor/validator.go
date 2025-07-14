package extractor

import (
	"bytes"
	"splitter-files/internal/models"
)

type FileValidator interface {
	Validate(data []byte) bool
}

type MSOfficeValidator struct{}
type OfficeOpenXMLValidator struct {
	expectedContent string
	expectedType    models.OfficeFileType
}
type JPEGValidator struct{}

func (v *MSOfficeValidator) Validate(data []byte) bool {
	return validateMSOfficeFile(data)
}

func (v *OfficeOpenXMLValidator) Validate(data []byte) bool {
	return validateOfficeOpenXML(v.expectedContent, v.expectedType)(data)
}

func (v *JPEGValidator) Validate(data []byte) bool {
	return validateJpegImproved(data)
}

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

func validateZipFile(data []byte) bool {
	return bytes.Equal(data[:4], []byte{0x50, 0x4B, 0x03, 0x04})
}
