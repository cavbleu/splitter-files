## Binary Data File Extraction Tool

**Documentation:**

[`Читать на Русском языке`](README_RU.md)  
[`Data Recovery Methodology`](docs/EN/README_method_recovery.md)  
[`Multithreaded Operation`](docs/EN/Readme_thread.md)  

### 1. Program Purpose  
The tool is designed for:  
- Analyzing binary files (including corrupted ones)  
- Locating and extracting embedded files of various formats  
- Recovering file structures  
- Saving extracted files with metadata  

### 2. Supported File Formats  
The tool recognizes and properly handles:  
- **Documents**:  
  - Microsoft Office (DOC/DOCX, XLS/XLSX, PPT/PPTX)  
  - PDF (Portable Document Format)  
  - RTF (Rich Text Format)  
  - ODT (OpenDocument Text)  
  - ODS - Spreadsheets (OpenDocument Spreadsheet)  
  - ODP - Presentations (OpenDocument Presentation)  
- **Archives**: ZIP  
- **Images**: JPEG/JPG  
- **Web Formats**: HTML  
- **Other**: Binary data with known signatures  

### 3. Building the Project  

#### For Windows:  
1. Install [GoLang](https://golang.org/dl/)  
2. Open Command Prompt  
3. Navigate to the program directory:  
```powershell  
cd path_to_program_directory  
```  
4. Build the executable:  
```powershell  
go build -o build\splitter-files.exe cmd\app\main.go  
```  

#### For Linux:  
1. Install Go:  
```bash  
sudo apt install golang  
```  
2. Navigate to the program directory  
3. Build the binary:  
```bash  
go build -o build/splitter-files cmd/app/main.go  
```  
4. Make it executable:  
```bash  
chmod +x splitter-files  
```  

**Usage:**  
```
splitter-files [flags] <input_file> <output_directory> [num_workers]
```

**Flags:**  
- `-version` - Display program version and exit  
- `-ext` - Comma-separated list of file extensions to extract (or "all" for all formats)  

**Supported Extensions:**  
doc, docx, ppt, pptx, xls, xlsx, jpg, jpeg, pdf, rtf, odt, ods, odp, ots, fods, zip, html  

**Examples:**  

1. Extract only PDF and JPEG files:  
```
splitter-files -ext pdf,jpg data.bin output_dir
```

2. Extract all supported formats:  
```
splitter-files -ext all data.bin output_dir
```

3. Extract with specified worker threads:  
```
splitter-files -ext docx,xlsx data.bin output_dir 4
```

4. Check program version:  
```
splitter-files -version
```

**Notes:**  
- Defaults to using all physical CPU cores  
- If `-ext` flag is omitted, extracts all supported formats  
- Program outputs detailed statistics upon completion  

**Exit Codes:**  
- 0 - Success  
- 1 - Parameter error or file operation failure  
