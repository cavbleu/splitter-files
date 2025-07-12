### **How the Program Locates and Extracts Files from Binary Data**

#### Input Data Requirements

1. Supported:  
   - Binary files of any size  
   - Partially corrupted data  
   - Files with nested structures  

2. Limitations:  
   - Minimum extractable file size: 2 KB  
   - Valid signatures must be present  

The program ensures reliable file extraction even from corrupted data, providing detailed statistics and processing information. Its multithreaded architecture enables efficient handling of large files while maintaining high format recognition accuracy.  

This allows for effective recovery of **intact files** but **does not guarantee** functionality for corrupted data within them. Specialized tools may be required for complex cases (e.g., overwritten PDFs).  

---

### Data Recovery Workflow  

The program uses **file signatures (magic numbers)** and structural validation to identify file boundaries. Here’s how it works:  

#### Stage 1: Preparation and Initialization  
1. Load the entire input file into memory  
2. Create an output directory for extracted files  
3. Initialize a worker thread pool (default = CPU core count)  

#### Stage 2: File Header Detection  
The program scans for **unique byte sequences** (magic numbers) indicating the start of a file format by sequentially checking signatures in the binary data.  

---  
 **Example Signatures:**  
| Format  | Signature (HEX)                     | Description                     |  
|---------|------------------------------------|-----------------------------|  
| **PDF** | `25 50 44 46` (`%PDF`)             | First 4 bytes of a PDF file    |  
| **ZIP** | `50 4B 03 04` (`PK..`)             | ZIP archive signature        |  
| **JPEG**| `FF D8 FF`                         | JPEG image start marker     |  
| **DOCX**| `50 4B 03 04` + structure checks   | ZIP container + Word files  |  
---  
**Process:**  
1. The program scans data byte-by-byte.  
2. Upon detecting a signature (e.g., `%PDF`), it validates:  
   - **Legitimacy** (e.g., PDF version `%PDF-1.7`).  
   - **File structure** (e.g., presence of `xref` and `%%EOF` in PDFs).  

---  
Signature validation example:  
```go  
     var fileSignatures = []FileSignature{  
         {Extension: "pdf", MagicNumber: []byte{0x25, 0x50, 0x44, 0x46}, Validator: validatePdf},  
         // ... other formats  
}  
```  

#### Stage 3: File Validation  
   - Each potential file undergoes format-specific validation:  
   - Structural integrity checks  
   - Format-specific validations:  
     - **PDF**: `xref` table, start/end markers  
     - **Office**: OOXML structure  
     - **JPEG**: SOI/EOI markers (`FF D8`/`FF D9`)  

Not all files with correct signatures are valid. Additional checks include:  
- **PDF**:  
  ```go  
  if !bytes.Contains(data, []byte("xref")) || !bytes.Contains(data, []byte("%%EOF")) {  
      return false // Invalid PDF  
  }  
  ```  
- **JPEG**:  
  ```go  
  if !bytes.Contains(data, []byte{0xFF, 0xD9}) {  
      return false // Incomplete JPEG  
  }  
  ```  
- **ZIP/DOCX**:  
  ```go  
  if !bytes.Contains(data, []byte{0x50, 0x4B, 0x05, 0x06}) {  
      return false // Corrupted ZIP  
  }  
  ```  

#### Stage 4: File Extraction  
1. **Boundary Detection**:  
   - Format-specific algorithms:  
     - PDF: `%%EOF` + `xref` check  
     - JPEG: `FF D9` marker  
     - ZIP/Office: `PK\x03\x04` + central directory  

2. **Edge Cases**:  
   - Nested files (e.g., images in PDFs)  
   - Fragmented/corrupted files  
   - Overlapping files  

3. **Saving Output**:  
   - Generated filenames (e.g., `file_0001.pdf`, `file_0002.jpg`)  
   - Metadata preservation:  
     - Original file offset  
     - Size  
     - Additional attributes (e.g., Office version, macros, encryption)  

#### Stage 5: Footer Detection  

After header detection, the program locates **end markers** or **subsequent signatures**.  

**Footer Detection Methods:**  
---  
**A. Fixed End Markers**  
Some formats have explicit terminators:  
- **JPEG**: `FF D9` (EOI marker).  
- **PDF**: `%%EOF` (requires structural validation).  
---  
**B. Structural Analysis**  
Complex formats (PDF, DOCX) require **structural integrity checks**:  
- **PDF**:  
  - Must contain `xref`.  
  - Must end with `%%EOF` (may include trailing bytes).  
  - `startxref` must precede `%%EOF`.  

- **ZIP/DOCX/XLSX**:  
  - Locate `50 4B 05 06` (ZIP end signature).  
  - Validate internal structure (e.g., `[Content_Types].xml` in DOCX).  

---  
**C. Next Signature Fallback**  
For formats without clear terminators:  
1. Scan for the **next signature** (e.g., JPEG after PDF).  
2. Set the previous file’s end **before the new signature**.  

---  

#### Stage 6: Post-Processing and Reporting  
1. **Coverage Analysis**:  
   - Processed data mapping  
   - Identification of unrecovered regions  
   - Detection of potential corruption  

2. **Reporting**:  
   - Extraction statistics  
   - Warnings/issues  
   - Input data visualization  

Example report:  
```  
=== Detailed Statistics ===  
Input file size:       2,145,678 bytes  
Extracted files:       23  
Total extracted size:  2,010,432 bytes  
Data coverage:         93.7%  

File type distribution:  
- PDF Document:        5  
- Word Document:       3  
- JPEG Image:          10  
- ZIP Archive:         2  
- Other:               3  

Unrecovered regions (3):  
- 1,250,000 - 1,251,200 (1,200 bytes)  
- 1,980,000 - 2,000,000 (20,000 bytes)  
```  

### Key Format-Specific Handling  

**PDF Files**:  
1. Strict `%PDF-` header matching  
2. Mandatory checks:  
   - `xref` table  
   - Valid footer (`%%EOF`)  
   - Minimum size (1 KB)  
3. Ignores embedded images  
4. Handles post-`%%EOF` newlines correctly  

**Office Documents**:  
1. Type detection (Word/Excel/PPT)  
2. Metadata analysis:  
   - Version  
   - Macros (`VBAProject`)  
   - Encryption flags  
3. OOXML structure validation  

**JPEG Images**:  
1. Precise boundary detection via markers  
2. Structure validation  
3. Supports progressive JPEGs  

### **Handling Corrupted Files**  
- If **no footer is found**, the program may:  
  - Extract up to the next signature (potentially incomplete).  
  - Skip files below size threshold (default: 2 KB).  

- If **structure is invalid**, files are flagged as **potentially corrupted**.  

---  

### Practical Example  

**Sample Binary Dump**:  
```  
[Some data][%PDF-1.4 ... xref ... %%EOF][FF D8 FF ... FF D9][50 4B 03 04 ... 50 4B 05 06]  
```  
1. Detects `%PDF` → validates `xref`/`%%EOF` → extracts PDF.  
2. Finds `FF D8 FF` → locates `FF D9` → extracts JPEG.  
3. Detects `50 4B 03 04` → validates `50 4B 05 06` → extracts ZIP.  

---  

### Program Limitations (What It **Cannot** Do):  

---  

#### 1. **Limited Format Support**  
   - Only predefined formats:  
     `PDF, DOC/DOCX, XLS/XLSX, PPT/PPTX, JPEG, ZIP, RTF, ODT, HTML`.  
   - **Unsupported**:  
     - Video (MP4, AVI, etc.)  
     - Audio (MP3, WAV)  
     - Specialized formats (CAD, databases, disk images).  

---  

#### 2. **No Decryption**  
   - Detects encryption (e.g., password-protected PDFs) but **cannot**:  
     - Crack passwords.  
     - Decrypt without keys.  

---  

#### 3. **No Filesystem Recovery**  
   - Operates on raw binary data:  
     - **No** folder structure recovery.  
     - **No** original filenames (outputs generic names like `file_0001.pdf`).  
     - **No** deleted file recovery (use tools like `PhotoRec`).  

---  

#### 4. **No Data Repair**  
   - Partially overwritten files (e.g., half a JPEG missing):  
     - Extracted but may be **unreadable**.  
     - **No** internal repair (e.g., broken Excel tables).  

---  

#### 5. **No 100% Recovery Guarantee**  
   - Effectiveness depends on:  
     - Corruption severity.  
     - Fragment availability.  
   - May miss files with damaged signatures.  

---  

#### 6. **No Live System Support**  
   - Only analyzes **static files** (e.g., memory dumps/disk images).  
   - **Cannot**:  
     - Scan disks directly.  
     - Recover files from running systems.  

---  

#### 7. **No Split Archive Handling**  
   - Multi-part ZIPs (`part1.zip`, `part2.zip`):  
     - Extracts each part separately.  
     - **No** auto-reassembly.  

---  

### When the Program **Fails**?  
1. Files overwritten with zeros/random data (no signatures).  
2. Filesystem-level encryption (BitLocker, VeraCrypt).  
3. Metadata recovery (timestamps, permissions).  

For such tasks, use specialized tools (e.g., `TestDisk` for FS or `John the Ripper` for passwords).  
