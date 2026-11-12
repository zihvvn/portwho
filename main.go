package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: portwho <port>")
		os.Exit(1)
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

	fmt.Print(string(output))
}
