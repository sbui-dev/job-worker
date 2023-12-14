// Copyright 2023 Steven Bui
package jobworker

import (
	"fmt"
	"os"
)

const (
	CPUMaxFile = "cpu.max"
	MEMMaxFile = "mem.max"
	IOMaxFile  = "io.max"
	JobFolder  = "/sys/fs/cgroup/jobworker/"
	MaxCPU     = 20000
	MaxMEM     = 10000
	MaxDiskIO  = 12800
)

func SetupCGroup() error {
	if err := os.Mkdir(JobFolder, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create folder: %v", err)
	}
	return nil
}
