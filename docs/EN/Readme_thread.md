### **How the Program Splits Source Files into Streams and Reconstructs Files**

#### 1. **Splitting Source Files into Processing Streams**

The program uses **multithreaded processing** to efficiently locate and extract files from binary data streams. Here's how it works:

**a. Reading source file:**
```go
data, err := ioutil.ReadFile(inputFile)
```
- The entire file is loaded into memory as a byte array (`[]byte`).

**b. Creating worker threads:**
```go
for i := 0; i < numWorkers; i++ {
    wg.Add(1)
    go worker(i, jobs, results, outputDir, &wg)
}
```
- A worker pool is created (default = number of physical CPU cores)
- Each worker operates in its own thread

**c. File signature detection:**
```go
foundSigs := findFileSignatures(data)
```
- The program scans data for known "magic numbers" (file signatures)
- Creates a `FileChunk` task for each detected file

**d. Priority queues:**
```go
officeQueue := make([]FileChunk, 0)
regularQueue := make([]FileChunk, 0)
```
- Office documents (Word, Excel, PowerPoint) get priority processing
- Other files go into the standard queue

#### 2. **Stream Processing**

**a. Workers processing tasks:**
```go
for chunk := range jobs {
    size, endPos, filename, fileType, officeInfo, err := extractFile(chunk.Data, outputDir, chunk.Counter, chunk.Start)
    // ...
}
```
- Each worker pulls tasks from the `jobs` channel
- Calls `extractFile` to process data chunks

**b. File boundary detection:**
```go
// For PDF files
eofPos := bytes.LastIndex(data, []byte("%%EOF"))
fileEnd := eofPos + 5

// For JPEG files
for i := len(data) - 2; i >= 0; i-- {
    if data[i] == 0xFF && data[i+1] == 0xD9 {
        fileEnd = i + 2
        break
    }
}
```
- Format-specific boundary detection:
  - PDF: `%%EOF` marker
  - JPEG: `FF D9` marker
  - ZIP/Office: `PK\x03\x04` signature

#### 3. **File Reconstruction**

**a. Data extraction:**
```go
fileData := data[:fileEnd]
filename := filepath.Join(outputDir, fmt.Sprintf("file_%04d.%s", counter, ext))
err := ioutil.WriteFile(filename, fileData, 0644)
```
- Saves data from signature start to file end
- Generates sequential filenames (file_0001.pdf, file_0002.jpg, etc.)

**b. Special PDF handling:**
```go
func extractPdfFile(data []byte, ...) {
    // Verify xref table presence
    xrefPos := bytes.LastIndex(data[:fileEnd], []byte("xref"))
    // ...
}
```
- Additional PDF integrity checks
- Ensures only valid PDFs are extracted (no fragments)

#### 4. **Synchronization and Statistics**

**a. Result aggregation:**
```go
results <- ExtractionResult{
    Filename:   filename,
    Size:      size,
    Start:     chunk.Start,
    End:       endPos,
    // ...
}
```
- Workers send results via the `results` channel
- Main thread compiles statistics

**b. Coverage analysis:**
```go
covered := make([]bool, len(data))
for _, r := range extractedRanges {
    for i := r[0]; i < r[1]; i++ {
        covered[i] = true
    }
}
```
- Builds a data coverage map
- Identifies unrecovered regions (potential corruption)

---

### Key Technical Notes:
1. **Worker Pool Pattern**: Uses buffered channels for task distribution
2. **Boundary Detection**: Implements format-specific termination logic
3. **Error Handling**: Invalid files are logged but don't halt processing
4. **Memory Efficiency**: Processes large files without temporary storage

### Optimization Highlights:
- **Parallel Signature Scanning**: Workers process different file segments concurrently
- **Priority Queuing**: Critical formats (Office docs) get immediate attention
- **Zero-Copy Extraction**: Direct byte slicing minimizes memory overhead