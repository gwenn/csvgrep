package main

import (
	"exec"
	"fmt"
	"io/ioutil"
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
	w, err := cmd.Wait(0)
	if err != nil {
		return
	}
	if !w.Exited() || w.ExitStatus() != 0 {
		err = w
	}
	return
}

func compress(cmd, source, target string) {
	err := run([]string{cmd, "-c", source},
		func(stdout *os.File) (e os.Error) {
			b, e := ioutil.ReadAll(stdout)
			if e != nil {
				return
			}
			e = ioutil.WriteFile(target, b, 0600)
			return
		},
		exec.Pipe, exec.PassThrough)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

const (
	FILE1 = "test.csv"
	FILE2 = "test.csv.gz"
	FILE3 = "test.csv.bz2"
)

func test() {
	err := run([]string{"./csvgrep", "-s=,", "c", FILE1, FILE2, FILE3},
		nil,
		exec.PassThrough, exec.PassThrough)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	compress("gzip", FILE1, FILE2)
	compress("bzip2", FILE1, FILE3)
	test()
}
