package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Process struct {
	PID  int
	Name string
}

func parseProcesses(output string) ([]Process, error) {
	var processes []Process

	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}

		switch line[0] {
		case 'p':
			pid, err := strconv.Atoi(line[1:])
			if err != nil || pid <= 0 {
				return nil, fmt.Errorf("invalid process ID: %q", line[1:])
			}

			processes = append(processes, Process{
				PID:  pid,
				Name: "unknown",
			})

		case 'c':
			if len(processes) == 0 {
				return nil, fmt.Errorf("process name appeared before a process ID")
			}

			processes[len(processes)-1].Name = line[1:]
		}
	}

	return processes, nil
}

func findProcesses(port int) ([]Process, error) {
	cmd := exec.Command(
		"lsof",
		"-nP",
		fmt.Sprintf("-iTCP:%d", port),
		"-sTCP:LISTEN",
		"-Fpc",
	)

	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError

		if errors.As(err, &exitErr) {
			details := strings.TrimSpace(string(exitErr.Stderr))

			if exitErr.ExitCode() == 1 &&
				len(output) == 0 &&
				details == "" {
				return nil, nil
			}

			if details != "" {
				return nil, fmt.Errorf("lsof: %s: %w", details, err)
			}
		}

		return nil, fmt.Errorf("running lsof: %w", err)
	}

	return parseProcesses(string(output))
}

func confirmStop() bool {
	fmt.Print("Stop the listed processes? [y/N]: ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false
	}

	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "y" || answer == "yes"
}

func stopProcesses(port int, targets []Process) error {
	// Refresh the lookup after confirmation, since ownership may have changed.
	current, err := findProcesses(port)
	if err != nil {
		return fmt.Errorf("rechecking port ownership: %w", err)
	}

	owners := make(map[int]string)
	for _, process := range current {
		owners[process.PID] = process.Name
	}

	stopped := make(map[int]bool)
	var failures []error

	for _, target := range targets {
		name, exists := owners[target.PID]
		if !exists || name != target.Name {
			fmt.Printf("Skipping PID %d: port ownership changed.\n", target.PID)
			continue
		}

		if stopped[target.PID] {
			continue
		}
		stopped[target.PID] = true

		// Protect system processes and portwho itself.
		if target.PID <= 1 || target.PID == os.Getpid() {
			failures = append(failures,
				fmt.Errorf("refusing to stop PID %d", target.PID))
			continue
		}

		if err := syscall.Kill(target.PID, syscall.SIGTERM); err != nil {
			if errors.Is(err, syscall.ESRCH) {
				fmt.Printf("PID %d has already exited.\n", target.PID)
				continue
			}

			failures = append(failures,
				fmt.Errorf("stopping PID %d: %w", target.PID, err))
			continue
		}

		fmt.Printf("Sent SIGTERM to %s (PID %d).\n", target.Name, target.PID)
	}

	// Give processes up to three seconds to release the port.
	deadline := time.Now().Add(3 * time.Second)

	for {
		remaining, err := findProcesses(port)
		if err != nil {
			failures = append(failures,
				fmt.Errorf("checking port after termination: %w", err))
			break
		}

		if len(remaining) == 0 {
			fmt.Printf("No TCP listener remains on port %d.\n", port)
			break
		}

		if time.Now().After(deadline) {
			for _, process := range remaining {
				fmt.Printf(
					"Port %d still has a listener: %s (PID %d).\n",
					port, process.Name, process.PID,
				)
			}

			failures = append(failures,
				fmt.Errorf("port %d still has a TCP listener", port))
			break
		}

		time.Sleep(200 * time.Millisecond)
	}

	return errors.Join(failures...)
}

func run() error {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		return fmt.Errorf("usage: portwho <port> [--kill]")
	}

	killRequested := false

	if len(os.Args) == 3 {
		if os.Args[2] != "--kill" {
			return fmt.Errorf("unknown option: %s", os.Args[2])
		}

		killRequested = true
	}

	port, err := strconv.Atoi(os.Args[1])
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("port must be a number between 1 and 65535")
	}

	processes, err := findProcesses(port)
	if err != nil {
		return err
	}

	if len(processes) == 0 {
		fmt.Printf("No TCP listener found on port %d.\n", port)
		return nil
	}

	for _, process := range processes {
		fmt.Printf(
			"Port %d is used by %s (PID %d).\n",
			port, process.Name, process.PID,
		)
	}

	if !killRequested {
		return nil
	}

	if !confirmStop() {
		fmt.Println("Cancelled.")
		return nil
	}

	return stopProcesses(port, processes)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
