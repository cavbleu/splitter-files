package fileutils

import (
	"io/ioutil"
	"regexp"
	"runtime"
)

func GetPhysicalCPUCount() int {
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" || runtime.GOOS == "freebsd" {
		data, err := ioutil.ReadFile("/proc/cpuinfo")
		if err == nil {
			physicalIDs := make(map[string]bool)
			re := regexp.MustCompile(`physical id\s*:\s*(\d+)`)
			matches := re.FindAllStringSubmatch(string(data), -1)
			for _, match := range matches {
				if len(match) > 1 {
					physicalIDs[match[1]] = true
				}
			}

			if len(physicalIDs) > 0 {
				coresPerCPU := make(map[string]int)
				reCore := regexp.MustCompile(`physical id\s*:\s*(\d+).*?cpu cores\s*:\s*(\d+)`)
				matchesCore := reCore.FindAllStringSubmatch(string(data), -1)
				for _, match := range matchesCore {
					if len(match) > 2 {
						coresPerCPU[match[1]] = coresPerCPU[match[1]] + 1
					}
				}

				totalCores := 0
				for _, cores := range coresPerCPU {
					totalCores += cores
				}

				if totalCores > 0 {
					return totalCores
				}
			}
		}
	}
	return runtime.NumCPU()
}
