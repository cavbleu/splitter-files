Here's the consolidated and grouped technical documentation:

### File Recovery Tool Technical Overview

#### 1. Program Purpose and Capabilities
A forensic file carving utility designed to:
- Recover files from corrupted storage/dumps using signature scanning
- Supported formats:
  - Documents: PDF, DOCX/XLSX/PPTX, RTF, ODT
  - Images: JPEG
  - Archives: ZIP
  - Web: HTML
- Key features:
  - Embedded file extraction (e.g., images from PDFs)
  - Recovery statistics and coverage analysis
  - Parallel processing support

#### 2. Installation and Usage

**Compilation:**
# Windows:
```powershell
# Windows (requires Go installation):
go build -o splitter-files.exe main.go
```

# Linux:
```bash
sudo apt install golang
go build -o splitter-files main.go
chmod +x filecarver
```

---

## Usage
Basic command format:
```bash
./file-splitter [flags] <input_file> <output_directory> [num_workers]
```

### Flags
| Flag        | Description                                      |
|-------------|--------------------------------------------------|
| `-version`  | Show program version and exit                    |
| `-ext`      | Comma-separated list of extensions to extract (or "all" for all supported types) |

### Examples

1. Extract all supported file types:
   ```bash
   ./file-splitter data.bin output_dir
   ```

2. Extract only PDF and JPEG files:
   ```bash
   ./file-splitter -ext pdf,jpg data.bin output_dir
   ```

3. Extract all supported types with 8 worker threads:
   ```bash
   ./file-splitter -ext all data.bin output_dir 8
   ```

4. Show version information:
   ```bash
   ./file-splitter -version
   ```

### Supported File Extensions
The program can extract these file types:
- Documents: `doc`, `docx`, `ppt`, `pptx`, `xls`, `xlsx`, `odt`, `rtf`, `pdf`
- Images: `jpg`, `jpeg`
- Archives: `zip`
- Web: `html`

## Output
The program provides:
- Real-time progress of extracted files
- Detailed statistics including:
  - File types distribution
  - Data coverage percentage
  - Uncovered areas
  - Office document properties (encryption, macros)

## Tips
1. For large files, increase the number of workers (up to your CPU core count)
2. Use `-ext` to speed up processing by limiting to specific file types
3. Check the statistics for potential file carving issues

## Troubleshooting
- **"No known file signatures found"**: The input may not contain supported file types
- **Low coverage warning**: The input may have corrupted or unsupported files
- **Permission errors**: Ensure you have write access to the output directory


#### 3. Technical Implementation

**File Detection Methodology:**

| Format  | Header Signature       | Validation Method               |
|---------|------------------------|---------------------------------|
| PDF     | `25 50 44 46` (%PDF)   | Requires xref and %%EOF markers |
| JPEG    | `FF D8 FF`             | Validates EOI marker `FF D9`    |
| ZIP     | `50 4B 03 04` (PK..)   | Checks end-of-archive marker    |

**Recovery Process:**
1. Header Scanning:
   - Linear search for magic bytes
   - Byte-offset validation

2. Boundary Detection:
   ```go
   // PDF example
   eof := bytes.LastIndex(data, []byte("%%EOF"))
   if eof != -1 && containsValidTrailer(data[eof+5:]) {
       return eof + 8
   }
   
   // JPEG example
   for i := len(data)-2; i >=0; i-- {
       if data[i] == 0xFF && data[i+1] == 0xD9 {
           return i+2
       }
   }
   ```

3. Validation:
   - Structure verification
   - Minimum size check (2KB)
   - Overlap prevention

#### 4. Limitations and Constraints

**Functional Limitations:**
- Format Support:
  - Unsupported: Media files (MP4/MP3), databases, proprietary formats
- Encryption:
  - Detects but cannot break encryption (AES-256, etc.)
  - No password recovery capability

**Technical Constraints:**
- Filesystem:
  - No directory structure recovery
  - No metadata preservation (timestamps, etc.)
- Data Integrity:
  - No repair of corrupted content
  - Cannot handle fragmented files
- Operational:
  - Static file analysis only (no live system scanning)
  - No multi-part archive merging

**When to Use Alternatives:**
- For encrypted data: `John the Ripper`
- For filesystem recovery: `TestDisk`
- For fragmented files: `Foremost`, `Scalpel`

#### 5. Output and Analysis

**Generated Output:**
- Recovered files (sequential naming)
- Statistics:
  - Recovery success rate
  - Data coverage percentage
  - Unallocated space mapping

**Design Priorities:**
1. Signature accuracy over speed
2. False-positive minimization
3. Parallel processing efficiency

**Forensic Enhancement Options:**
- Add SHA-256 hash verification
- Implement write-blocking for live analysis
- Expand filesystem-specific format support

