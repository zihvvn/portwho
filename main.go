package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
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

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		fmt.Fprintln(os.Stderr, "Usage: portwho <port> [--kill]")
		os.Exit(1)
	}

	killRequested := false

	if len(os.Args) == 3 {
		if os.Args[2] != "--kill" {
			fmt.Fprintln(os.Stderr, "Unknown option:", os.Args[2])
			os.Exit(1)
		}

		killRequested = true
	}

	port, err := strconv.Atoi(os.Args[1])
	if err != nil || port < 1 || port > 65535 {
		fmt.Fprintln(os.Stderr, "Port must be a number between 1 and 65535")
		os.Exit(1)
	}

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

		if errors.As(err, &exitErr) &&
			exitErr.ExitCode() == 1 &&
			len(output) == 0 &&
			len(exitErr.Stderr) == 0 {
			fmt.Printf("No TCP listener found on port %d.\n", port)
			return
		}

		fmt.Fprintln(os.Stderr, "Lookup failed:", err)

		if exitErr != nil && len(exitErr.Stderr) > 0 {
			fmt.Fprint(os.Stderr, string(exitErr.Stderr))
		}

		os.Exit(1)
	}

	processes, err := parseProcesses(string(output))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not parse lsof output:", err)
		os.Exit(1)
	}

	if len(processes) == 0 {
		fmt.Printf("No TCP listener found on port %d.\n", port)
		return
	}

	for _, process := range processes {
		fmt.Printf(
			"Port %d is used by %s (PID %d).\n",
			port,
			process.Name,
			process.PID,
		)
	}

	if killRequested {
		fmt.Println("Kill requested. Process termination is not implemented yet.")
	}
}
