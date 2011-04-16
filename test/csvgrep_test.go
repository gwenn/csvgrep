package main

import (
	"exec"
	"fmt"
	"os"
)

func run(argv []string, f func(*os.File) os.Error, stdout, stderr int) (err os.Error) {
	exe, err := exec.LookPath(argv[0])
	if err != nil {
		return
	}
	getwd, _ := os.Getwd()
	cmd, err := exec.Run(exe, argv, os.Environ(), getwd, exec.DevNull, stdout, stderr)
	if err != nil {
		return
	}
	defer cmd.Close()
	if stdout == exec.Pipe {
		err = f(cmd.Stdout)
	}
	if err != nil {
		cmd.Wait(0)
		return
	}
	w, err := cmd.Wait(1)
	if err != nil {
		return
	}
	if !w.Exited() || w.ExitStatus() != 0 {
		err = w
	}
	return
}

func compress(cmd, source, target string) {
	err := run([]string{"sh", "-x", "-c", cmd + " -c " + source + " > " + target},
		nil,
		exec.PassThrough, exec.PassThrough)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func testWithMultipleFiles() {
	err := run([]string{"sh", "-x", "-c", "../csvgrep -s=, 'c' test.csv*"},
		nil,
		exec.DevNull, exec.PassThrough)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

const (
	FILE = "test.csv"
)

func main() {
	compress("gzip", FILE, FILE+".gz")
	compress("bzip2", FILE, FILE+".bz2")
	testWithMultipleFiles()
}
