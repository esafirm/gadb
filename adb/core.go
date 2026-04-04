package adb

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

const DefaultTimeout = 5 * time.Second

func runWithPrint(name string, arg ...string) CommandReturn {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()

	output, err := exec.CommandContext(ctx, name, arg...).CombinedOutput()
	if err == nil {
		fmt.Println(string(output))
	}
	return CommandReturn{output, err}
}

func runOnly(name string, arg ...string) CommandReturn {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()

	output, err := exec.CommandContext(ctx, name, arg...).CombinedOutput()
	return CommandReturn{output, err}
}

// RunOnly executes a command and returns the output without printing
func RunOnly(name string, arg ...string) CommandReturn {
	return runOnly(name, arg...)
}

// RunOnlyWithTimeout executes a command with a timeout and returns the output without printing
func RunOnlyWithTimeout(timeout time.Duration, name string, arg ...string) CommandReturn {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	output, err := exec.CommandContext(ctx, name, arg...).CombinedOutput()
	return CommandReturn{output, err}
}
