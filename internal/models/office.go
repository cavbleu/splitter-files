package models

type OfficeFileType int

const (
	UnknownOffice OfficeFileType = iota
	WordDocument
	ExcelDocument
	PowerPointDocument
)

type OfficeDocumentInfo struct {
	Type        OfficeFileType
	Version     string
	IsEncrypted bool
	IsMacro     bool
}
