package extractor

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"splitter-files/internal/models"
	"strings"
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

func validateOfficeOpenXML(expectedContent string, expectedType models.OfficeFileType) func([]byte) bool {
	return func(data []byte) bool {
		if !validateZipFile(data) {
			return false
		}

		zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return false
		}

		var contentTypes struct {
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

		var hasContentTypes bool
		officeInfo := models.OfficeDocumentInfo{}

		for _, file := range zipReader.File {
			switch file.Name {
			case "[Content_Types].xml":
				// ... парсинг XML ...
				for _, override := range contentTypes.Override {
					switch {
					case strings.Contains(override.ContentType, "wordprocessing"):
						officeInfo.Type = models.WordDocument
					case strings.Contains(override.ContentType, "spreadsheet"):
						officeInfo.Type = models.ExcelDocument
					case strings.Contains(override.ContentType, "presentation"):
						officeInfo.Type = models.PowerPointDocument
					}
				}

				// ... обработка других файлов ...
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
