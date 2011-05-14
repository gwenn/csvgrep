/*
Pretty-print lines matching a pattern in CSV files.
Transparent support for gzipped/bzipped files.
TODO
Ignore match in header line.
Make possible to customize output format 
Match only specific field by index or name
Match only whole field
Show header mode
Try to guess separator.

The author disclaims copyright to this source code.
*/
package main

import (
	"bufio"
	"bytes"
	"gocsv.googlecode.com/hg"
	"exec"
	"fmt"
	"flag"
	"io/ioutil"
	"os"
	"strings"
	"strconv"
)

type Config struct {
	grepOptions []string
	fields      []uint
	noHeader    bool
	separator   byte
	start       int
}

/*
-H --with-filename
-h --no-filename

-c column number
*/
func parseArgs() *Config {
	var i *bool = flag.Bool("i", false, "ignore case distinctions")
	var w *bool = flag.Bool("w", false, "force PATTERN to match only whole words")
	var n *bool = flag.Bool("n", false, "no header")
	var sep *string = flag.String("s", ";", "Set the field separator")
	var f *string = flag.String("f", "", "Set the field indexes to be matched (starts at 1)")
	var v *int = flag.Int("v", 1, "first column number")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-iwn] [-s=C] [-v=N] [-f=N,...] PATTERN FILE...\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "Missing PATTERN argument\n")
		flag.Usage()
		os.Exit(1)
	}
	if flag.NArg() == 1 {
		fmt.Fprintf(os.Stderr, "Missing FILE argument\n")
		flag.Usage()
		os.Exit(1)
	}
	// TODO flag.Visit
	options := []string{"-h"}
	if *i {
		options = append(options, "-i")
	}
	if *w {
		options = append(options, "-w")
	}
	if len(*sep) > 1 {
		fmt.Fprintf(os.Stderr, "Separator must be only one character long\n")
		flag.Usage()
		os.Exit(1)
	}
	var fields []uint
	if len(*f) > 0 {
		rawFields := strings.Split(*f, ",", -1)
		fields = make([]uint, len(rawFields))
		for i, s := range rawFields {
			f, err := strconv.Atoui(s)
			if err != nil  {
				fmt.Fprintf(os.Stderr, "Invalid field index (%v)\n", s)
				flag.Usage()
				os.Exit(1)
			}
			fields[i] = f - 1
		}
	}
	return &Config{grepOptions: options, noHeader: *n, separator: (*sep)[0], start: *v, fields: fields}
}

func run(argv []string, f func(*os.File) os.Error, checkExitStatus bool) (err os.Error) {
	exe, err := exec.LookPath(argv[0])
	if err != nil {
		return
	}
	getwd, _ := os.Getwd()
	cmd, err := exec.Run(exe, argv, os.Environ(), getwd, exec.DevNull, exec.Pipe, exec.PassThrough)
	if err != nil {
		return
	}
	defer cmd.Close()
	err = f(cmd.Stdout)
	if err != nil {
		cmd.Wait(0)
		return
	}
	w, err := cmd.Wait(1)
	if err != nil {
		return
	}
	if !w.Exited() || (checkExitStatus && w.ExitStatus() != 0) {
		err = w
	}
	return
}

func magicType(f string) (out string, err os.Error) {
	err = run([]string{"file", "-b", "-i", f},
		func(stdout *os.File) (e os.Error) {
			b, e := ioutil.ReadAll(stdout)
			if e != nil {
				return
			}
			out = string(bytes.TrimSpace(b)) // chomp
			return
		},
		true)
	return
}

const MAX = 25

func head(cat, f string, sep byte) (headers []string, headerMaxLength int, err os.Error) {
	err = run([]string{cat, f},
		func(stdout *os.File) (e os.Error) {
			bufIn := bufio.NewReader(stdout)
			reader := csv.NewReader(bufIn)
			reader.Config.FieldDelim = sep
			headers, e = reader.ReadRow()
			return
		},
		true)

	headerMaxLength = 0
	for _, header := range headers {
		if len(header) > headerMaxLength {
			headerMaxLength = len(header)
		}
	}
	if headerMaxLength > MAX {
		headerMaxLength = MAX
		for i, header := range headers {
			if len(header) > headerMaxLength {
				headers[i] = headers[i][0:headerMaxLength]
			}
		}
	}
	return
}

func match(fields []uint, pattern string, values []string) bool {
	if values == nil {
		return false
	} else if len(fields) == 0 {
		return true
	}
	for _, field := range fields {
		fmt.Printf("%v %s\n", values[field], pattern)
		if values[field - 1] == pattern { // FIXME regexp & -w & -i
			return true
		}
	}
	return false
}

func grep(cat, grep, pattern, f string, config *Config) (found bool, err os.Error) {
	//fmt.Println(f, config)
	var format string
	var headers []string
	if config.noHeader {
		format = "\t%2d : %s\n"
	} else {
		var headerMaxLength int
		headers, headerMaxLength, err = head(cat, f, config.separator)
		if err != nil {
			return
		}
		format = "\t%-" + strconv.Itoa(headerMaxLength) + "s (%2d) : %s\n"
		//fmt.Printf("Headers: %v (%d)\n", headers, headerMaxLength)
	}
	args := []string{grep}
	args = append(args, config.grepOptions...)
	args = append(args, pattern, f)
	//fmt.Printf("Grep: %v\n", args)
	err = run(args,
		func(stdout *os.File) (e os.Error) {
			bufIn := bufio.NewReader(stdout)
			reader := csv.NewReader(bufIn)
			reader.Config.FieldDelim = config.separator
			for {
				values, e := reader.ReadRow()
				if match(config.fields, pattern, values) {
					if !found {
						fmt.Println(f, ":")
						found = true
					}
					fmt.Println("-")
					// package tabwriter
					for i, value := range values {
						if config.noHeader {
							fmt.Printf(format, i+config.start, value)
						} else if i < len(headers) {
							fmt.Printf(format, headers[i], i+config.start, value)
						} else {
							fmt.Printf(format, "???", i+config.start, value)
						}
					}
				}
				if e != nil {
					if e != os.EOF {
						err = e
					}
					break
				}
			}
			return
		},
		false)
	return
}

func main() {
	config := parseArgs()
	pattern := flag.Arg(0)
	errorCount := 0
	matchCount := 0
	found := false
	for i := 1; i < flag.NArg(); i++ {
		if found {
			fmt.Println("---")
			found = false
		}
		f := flag.Arg(i)
		magic, err := magicType(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error while checking file type: '%s' (%s)\n", f, err)
			errorCount++
			continue
		}
		if strings.Contains(magic, "text/plain") {
			found, err = grep("cat", "grep", pattern, f, config)
		} else if strings.Contains(magic, "application/x-gzip") {
			found, err = grep("zcat", "zgrep", pattern, f, config)
		} else if strings.Contains(magic, "application/x-bzip2") {
			found, err = grep("bzcat", "bzgrep", pattern, f, config)
		} else {
			fmt.Fprintf(os.Stderr, "Unsupported file type: '%s' (%s)\n", f, magic)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			errorCount++
		} else if found {
			matchCount++
		}
	}
	if errorCount > 0 || matchCount == 0 {
		os.Exit(1)
	}
}
