# Программа для расшифровки конкатинированного raw файла

## Описание

Программа для извлечения конкатенированного(скленного) файлов из одного целого файла методом прочтения HEX-представления произвольных двоичных данных сигнатур в текстовом виде. После выполняет валидацию данных и разделяет их на отдельные файлы по конечной сигнатуре.

Для извлечения данных потребуется дисковый объем превышающий исходный объем поврежденных файлов и потребует значительных временных ресурсов. Поэтому желательно выполнять извлечение на внешний накопитель и ожидать окончания работы программы.


Применяется в случае:
* Повреждения файла. 
* Архив имеет поврежденные заголовки. Пример ошибки извлечения
```
Open ERROR: Can not open the file as [7z] archive


ERRORS:
Headers Error
    
Can't open as archive: 1
Files: 0
Size:       0
Compressed: 0
```

Программа может искать сигнатуры:
* **doc** `0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1` MimeType: "application/msword"
* **docx** `0x50, 0x4B, 0x03, 0x04` MimeType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
* **ppt** `0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1` MimeType: "application/vnd.ms-powerpoint",
* **pptx** `0x50, 0x4B, 0x03, 0x04` MimeType: "application/vnd.openxmlformats-officedocument.presentationml.presentation"
* **xls** `0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1` MimeType: "application/vnd.ms-excel",
* **xlsx** `0x50, 0x4B, 0x03, 0x04` MimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
* **jpg** `0xFF, 0xD8, 0xFF` MimeType: "image/jpeg"
* **jpeg** `0xFF, 0xD8, 0xFF` MimeType: "image/jpeg"
* **pdf** `0x25, 0x50, 0x44, 0x46` MimeType: "application/pdf"
* **lnk** `0x4C, 0x00, 0x00, 0x00, 0x01, 0x14, 0x02, 0x00` MimeType: "application/x-ms-shortcut"
* **tmp** `0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00` MimeType: "application/octet-stream"
* **rtf** `0x7B, 0x5C, 0x72, 0x74, 0x66, 0x31` MimeType: "application/rtf"
* **odt** `0x50, 0x4B, 0x03, 0x04` MimeType: "application/vnd.oasis.opendocument.text"
* **zip** `0x50, 0x4B, 0x03, 0x04`
* **gz** `0x1F, 0x8B`
* **7z** `0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C`
* **exe** `0x4D, 0x5A` MimeType: "application/vnd.microsoft.portable-executable"
* **dll** `0x4D, 0x5A` MimeType: "application/vnd.microsoft.portable-executable"
* **html** `0x3C, 0x21, 0x44, 0x4F, 0x43, 0x54, 0x59, 0x50, 0x45, 0x20, 0x68, 0x74, 0x6D, 0x6C`
* **html** `0x4D, 0x5A`
* **html** `0x3C, 0x68, 0x74, 0x6D, 0x6C`
* **html** `0x3C, 0x48, 0x54, 0x4D, 0x4C`

## Сборка проекта

Для сборки проекта требуется установленный пакет Golang

### Сборка проекта под ОС Windows 

```
go mod init tiny
go build -o splitter-files.exe splitter-files.go
```


### Сборка проекта под ОС Linux

```
go mod init tiny
go build -o splitter-files splitter-files.go
```


## Использование программы

Запустите программу с указанием обрабатываемого файла и директории куда нужно извлечь файлы. Для извлечения данных потребуется дисковый объем превышающий исходный объем поврежденных файлов и потребует значительных верменных ресурсов. 

В операционной системе Windows
```
.\splitter-files.exe \path\to\raw_file E:\path\to\import\file
```

В операционной системе Linux
```
chmod +x splitter-files
splitter-files \path\to\raw_file \path\to\import\file
```

## Сохранение результатов

* Каждый найденный файл сохраняется с уникальным именем в формате file_Порядковый-№.расширение
* Выводится подробная информация о каждом найденном файле