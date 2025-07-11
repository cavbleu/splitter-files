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

**Execution:**
```bash
./splitter-files <input_file> <output_dir> [threads]
# Example:
./splitter-files bin recovered_files 8
```

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

