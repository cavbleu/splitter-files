### Как программа разбивает исходный файл на потоки и собирает файлы

#### 1. **Разбиение исходного файла на потоки**

Программа использует **многопоточную обработку** для эффективного поиска и извлечения файлов из исходного бинарного потока. Вот как это работает:

**a. Чтение исходного файла:**
```go
data, err := ioutil.ReadFile(inputFile)
```
- Весь файл считывается в память как массив байт (`[]byte`).

**b. Создание рабочих потоков (worker'ов):**
```go
for i := 0; i < numWorkers; i++ {
    wg.Add(1)
    go worker(i, jobs, results, outputDir, &wg)
}
```
- Создается пул worker'ов (по умолчанию = количеству физических CPU ядер)
- Каждый worker работает в своем потоке

**c. Поиск сигнатур файлов:**
```go
foundSigs := findFileSignatures(data)
```
- Программа сканирует данные, ища известные "магические числа" (сигнатуры) файлов
- Для каждого найденного файла создается задача (`FileChunk`)

**d. Приоритетная очередь:**
```go
officeQueue := make([]FileChunk, 0)
regularQueue := make([]FileChunk, 0)
```
- Office-документы (Word, Excel, PowerPoint) обрабатываются в первую очередь
- Остальные файлы попадают в обычную очередь

#### 2. **Обработка потоков**

**a. Worker'ы получают задачи:**
```go
for chunk := range jobs {
    size, endPos, filename, fileType, officeInfo, err := extractFile(chunk.Data, outputDir, chunk.Counter, chunk.Start)
    // ...
}
```
- Каждый worker берет задачу из канала `jobs`
- Вызывает функцию `extractFile` для обработки фрагмента данных

**b. Определение границ файлов:**
```go
// Для PDF файлов
eofPos := bytes.LastIndex(data, []byte("%%EOF"))
fileEnd := eofPos + 5

// Для JPEG файлов
for i := len(data) - 2; i >= 0; i-- {
    if data[i] == 0xFF && data[i+1] == 0xD9 {
        fileEnd = i + 2
        break
    }
}
```
- Для каждого типа файла используется свой алгоритм определения конца файла
- PDF: ищется маркер `%%EOF`
- JPEG: ищется маркер `FF D9`
- ZIP/Office: ищется сигнатура `PK\x03\x04`

#### 3. **Сборка файлов**

**a. Извлечение данных:**
```go
fileData := data[:fileEnd]
filename := filepath.Join(outputDir, fmt.Sprintf("file_%04d.%s", counter, ext))
err := ioutil.WriteFile(filename, fileData, 0644)
```
- Данные от начала сигнатуры до конца файла сохраняются в отдельный файл
- Имена файлов генерируются последовательно (file_0001.pdf, file_0002.jpg и т.д.)

**b. Специальная обработка PDF:**
```go
func extractPdfFile(data []byte, ...) {
    // Проверка наличия xref таблицы
    xrefPos := bytes.LastIndex(data[:fileEnd], []byte("xref"))
    // ...
}
```
- Дополнительные проверки целостности PDF
- Гарантия, что извлекается только валидный PDF, а не его фрагменты

#### 4. **Синхронизация и статистика**

**a. Сбор результатов:**
```go
results <- ExtractionResult{
    Filename:   filename,
    Size:      size,
    Start:     chunk.Start,
    End:       endPos,
    // ...
}
```
- Каждый worker отправляет результаты в канал `results`
- Главный поток собирает статистику

**b. Анализ покрытия:**
```go
covered := make([]bool, len(data))
for _, r := range extractedRanges {
    for i := r[0]; i < r[1]; i++ {
        covered[i] = true
    }
}
```
- Программа строит карту покрытия исходных данных
- Выявляет непокрытые участки (потенциально поврежденные данные)

