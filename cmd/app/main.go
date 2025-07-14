package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"splitter-files/internal/extractor"
	"splitter-files/internal/worker"
	"splitter-files/pkg/fileutils"
)

const Version = "1.2.1"

var (
	versionFlag    = flag.Bool("version", false, "Print version information")
	extensionsFlag = flag.String("ext", "", "Comma-separated list of file extensions to extract")
)

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Printf("File Splitter version %s\n", Version)
		os.Exit(0)
	}

	if len(flag.Args()) < 2 {
		printUsage()
		os.Exit(1)
	}

	args := flag.Args()
	inputFile := args[0]
	outputDir := args[1]

	allowedExtensions := parseExtensions(*extensionsFlag)
	numWorkers := fileutils.GetPhysicalCPUCount()
	if len(args) > 2 {
		if n, err := fmt.Sscanf(args[2], "%d", &numWorkers); err != nil || n != 1 || numWorkers < 1 {
			fmt.Printf("Invalid number of workers, using default (%d)\n", numWorkers)
		}
	}

	data, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("Error reading input file: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Processing file %s (%d bytes) with %d workers\n",
		inputFile, len(data), numWorkers)
	if len(allowedExtensions) > 0 {
		extList := fileutils.GetMapKeys(allowedExtensions)
		fmt.Printf("Extracting only: %s\n", strings.Join(extList, ", "))
	}

	startTime := time.Now()
	results, stats, err := worker.ProcessFile(data, outputDir, numWorkers, allowedExtensions)
	elapsed := time.Since(startTime)

	if err != nil {
		fmt.Printf("Processing completed with errors: %v\n", err)
	}

	fileutils.PrintStats(stats, results)
	fmt.Printf("\nProcessing completed in %s\n", elapsed)
}

func printUsage() {
	fmt.Println(`File Splitter - tool for extracting embedded files from binary data.
Version:`, Version, `
Usage: file-splitter [flags] <input_file> <output_directory> [num_workers]

Flags:`)
	flag.PrintDefaults()
	fmt.Printf("\nSupported file extensions: %s\n", strings.Join(extractor.GetSupportedExtensions(), ", "))
	fmt.Println(`Examples:
  file-splitter -ext pdf,jpg,docx data.bin output_dir
  file-splitter -ext all data.bin output_dir 8`)
}

func parseExtensions(extStr string) map[string]bool {
	allowed := make(map[string]bool)
	if extStr == "all" {
		for _, ext := range extractor.GetSupportedExtensions() {
			allowed[ext] = true
		}
		return allowed
	}

	exts := strings.Split(extStr, ",")
	for _, ext := range exts {
		ext = strings.TrimSpace(strings.ToLower(ext))
		if ext != "" {
			allowed[ext] = true
		}
	}
	return allowed
}
