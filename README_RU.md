## Программа для извлечения файлов из бинарных данных

**Документация/Documentation:**

[`Read to English Language`](docs/README.md)
[`Метод востановления данных`](docs/RU/README_method_recovery.md)
[`Работа в многопоточном режиме`](docs/RU/Readme_thread.md)

### 1. Назначение программы
Программа предназначена для:
- Анализа бинарных файлов (включая поврежденные)
- Поиска и извлечения вложенных файлов различных форматов
- Восстановления структуры найденных файлов
- Сохранения извлеченных файлов с метаинформацией.

### 2. Поддерживаемые форматы файлов
Программа распознает и корректно обрабатывает:
- **Документы**:
  - Microsoft Office (DOC/DOCX, XLS/XLSX, PPT/PPTX)
  - PDF (Portable Document Format)
  - RTF (Rich Text Format)
  - ODT (OpenDocument Text)
  - ODF - Таблицы (OpenDocument Table)
  - ODP - Презентации (OpenDocument Presentation)
- **Архивы**: ZIP
- **Изображения**: JPEG/JPG
- **Веб-форматы**: HTML
- **Другие**: бинарные данные с известными сигнатурами

### 3. Сборка проекта

#### Для Windows:
1. Установите [GoLang](https://golang.org/dl/)
2. Откройте командную строку
3. Перейдите в папку с программой:
```powershell
   cd путь_к_папке_с_программой
```
4. Соберите exe-файл:
```powershell
   go build -o build\splitter-files.exe  cmd\app\main.go
```

#### Для Linux:
1. Установите Go:
```bash
   sudo apt install golang
```
2. Перейдите в папку с программой
3. Соберите бинарник:
```bash
   go build -o build\splitter-files cmd\app\main.go
```
4. Сделайте исполняемым:
```bash
   chmod +x splitter-files
```


**Использование:**
```
file-splitter [flags] <input_file> <output_directory> [num_workers]
```

**Флаги:**
- `-version` - вывести версию программы и выйти
- `-ext` - список расширений файлов для извлечения (через запятую) или "all" для всех

**Поддерживаемые расширения:**
doc, docx, ppt, pptx, xls, xlsx, jpg, jpeg, pdf, rtf, odt, ods, odp, ots, fods, zip, html

**Примеры:**

1. Извлечение только PDF и JPEG файлов:
```
file-splitter -ext pdf,jpg data.bin output_dir
```

2. Извлечение всех поддерживаемых форматов:
```
file-splitter -ext all data.bin output_dir
```

3. Извлечение с указанием количества рабочих потоков:
```
file-splitter -ext docx,xlsx data.bin output_dir 4
```

4. Просмотр версии программы:
```
file-splitter -version
```

**Примечания:**
- По умолчанию используется количество физических ядер CPU
- Если флаг `-ext` не указан, извлекаются все поддерживаемые форматы
- Программа выводит подробную статистику по завершении работы

**Выходные коды:**
- 0 - успешное выполнение
- 1 - ошибка в параметрах или при работе с файлами


