package logtui

import (
	"bufio"
	"context"
	"os/exec"
	"time"

	"github.com/esafirm/gadb/adb"
)

// StartLogcatStream spawns `adb logcat` and sends parsed LogEntry objects to outCh
func StartLogcatStream(ctx context.Context, serial string, outCh chan<- *LogEntry, errCh chan<- error) func() {
	args := []string{}
	if serial != "" {
		args = append(args, "-s", serial)
	}
	// -T 1000 tails last 1000 lines then follows live stream
	// -v threadtime outputs date, time, pid, tid, level, tag, and message
	args = append(args, "logcat", "-T", "1000", "-v", "threadtime")

	cmd := exec.CommandContext(ctx, "adb", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		errCh <- err
		return func() {}
	}

	if err := cmd.Start(); err != nil {
		errCh <- err
		return func() {}
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		// Set buffer capacity to 1MB to handle large stack traces or json payloads
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				line := scanner.Text()
				if entry := ParseLogLine(line); entry != nil {
					outCh <- entry
				}
			}
		}

		if err := scanner.Err(); err != nil && ctx.Err() == nil {
			errCh <- err
		}
	}()

	cancel := func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}

	return cancel
}

// WatchPids periodically checks running PIDs for a package on a device
func WatchPids(ctx context.Context, serial string, packageName string, interval time.Duration, pidCh chan<- []string) {
	if packageName == "" {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial check
	if pids, err := adb.GetPidsForPackage(serial, packageName); err == nil {
		pidCh <- pids
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if pids, err := adb.GetPidsForPackage(serial, packageName); err == nil {
				pidCh <- pids
			}
		}
	}
}
