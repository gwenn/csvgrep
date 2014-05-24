/*
Pretty-print lines matching a pattern in CSV files.
Transparent support for gzipped/bzipped files.
TODO
Make possible to customize output format
Match only specific field by index or name
Match only whole field

The author disclaims copyright to this source code.
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/gwenn/yacr"
)

type config struct {
	fields     []uint64
	ignoreCase bool
	wholeWord  bool
	noHeader   bool
	sep        byte
	guess      bool
	quoted     bool
	start      int
	descMode   bool
}

func isFile(name string) bool {
	s, err := os.Stat(name)
	return err == nil && ((s.Mode()&os.ModeType == 0) || (s.Mode()&os.ModeSymlink != 0))
}

/*
-H --with-filename
-h --no-filename

-c column number
*/
func parseArgs() *config {
	var i = flag.Bool("i", false, "ignore case distinctions")
	var w = flag.Bool("w", false, "force PATTERN to match only whole words")
	var n = flag.Bool("n", false, "no header")
	var d = flag.Bool("d", false, "only show header/describe first line (no grep)")
	var q = flag.Bool("q", true, "quoted field mode")
	var sep = flag.String("s", ",", "set the field separator")
	var f = flag.String("f", "", "set the field indexes to be matched (starts at 1)")
	var v = flag.Int("v", 1, "first column number in output/result")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-i] [-w] [-n] [-q] [-s=C] [-v=N] [-f=N,...] [-d|PATTERN] FILE...\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		if *d {
			log.Fatalf("Missing FILE argument\n")
		} else {
			log.Fatalf("Missing PATTERN argument\n")
		}
	}
	// TODO Add support to Stdin when no file is specified?
	if flag.NArg() == 1 && !*d {
		flag.Usage()
		if isFile(flag.Arg(0)) {
			log.Fatalf("Missing PATTERN argument\n")
		} else {
			log.Fatalf("Missing FILE argument\n")
		}
	}
	if len(*sep) == 0 {
		flag.Usage()
		log.Fatalf("Separator value missing\n")
	} else if *sep == "\\t" {
		*sep = "\t"
	} else if len(*sep) > 1 {
		flag.Usage()
		log.Fatalf("Separator must be only one character long\n")
	}
	guess := true
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "s" {
			guess = false
		}
	})

	var fields []uint64
	if len(*f) > 0 {
		rawFields := strings.Split(*f, ",")
		fields = make([]uint64, len(rawFields))
		for i, s := range rawFields {
			f, err := strconv.ParseUint(s, 10, 0)
			if err != nil {
				flag.Usage()
				log.Fatalf("Invalid field index (%v)\n", s)
			}
			fields[i] = f - 1
		}
	}
	return &config{noHeader: *n, ignoreCase: *i, wholeWord: *w, sep: (*sep)[0], guess: guess, quoted: *q, start: *v, fields: fields, descMode: *d}
}

func match(fields []uint64, pattern *regexp.Regexp, values [][]byte) bool {
	if values == nil {
		return false
	} else if len(fields) == 0 {
		for _, value := range values {
			if pattern.Match(value) {
				return true
			}
		}
		return false
	}
	for _, field := range fields {
		value := values[field]
		//fmt.Printf("%v %s\n", value, pattern)
		if pattern.Match(value) {
			return true
		}
	}
	return false
}

func grep(pattern *regexp.Regexp, f string, config *config) (found bool, err error) {
	//fmt.Println(f, config)
	in, err := yacr.Zopen(f)
	if err != nil {
		return
	}
	defer in.Close()
	reader := yacr.NewReader(in, config.sep, config.quoted, config.guess)

	var headers []string
	if config.noHeader && !config.descMode {
	} else {
		for reader.Scan() {
			headers = append(headers, reader.Text())
			if reader.EndOfRecord() {
				break
			}
		}
		// TODO Try to guess/fix the separator if an error occurs (or if only one column is found)
		if err = reader.Err(); err != nil {
			return
		}
		//fmt.Printf("Headers: %v (%d)\n", headers)
	}

	//tw := tabwriter.NewWriter(os.Stdout, 8, 1, 8, '\t', tabwriter.Debug)
	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	if config.descMode {
		fmt.Println(f, ":")
		for i, value := range headers {
			tw.Write([]byte(fmt.Sprintf("%d\t%s\n", i+config.start, value)))
		}
		tw.Flush()
		return
	}

	var values = make([][]byte, 0, 10)
	var v, cv []byte
	orig := values
	i := 0
	for reader.Scan() {
		v = reader.Bytes() // must be copied
		if i < len(orig) {
			cv = orig[i]
			cv = append(cv[:0], v...)
		} else {
			cv = make([]byte, len(v))
			copy(cv, v)
		}
		values = append(values, cv)
		if !reader.EndOfRecord() {
			i++
			continue
		}
		if match(config.fields, pattern, values) {
			if !found {
				fmt.Println(f, ":")
				found = true
			}
			fmt.Println("-")
			for i, value := range values {
				if config.noHeader {
					tw.Write([]byte(fmt.Sprintf("%d\t%s\n", i+config.start, value)))
				} else if i < len(headers) {
					tw.Write([]byte(fmt.Sprintf("%d\t%s\t%s\n", i+config.start, headers[i], value)))
				} else {
					tw.Write([]byte(fmt.Sprintf("%d\t%s\t%s\n", i+config.start, "???", value)))
				}
			}
			tw.Flush()
		}
		orig = values
		values = values[:0]
		i = 0
	}
	err = reader.Err()
	return
}

func mustCompile(p string, config *config) *regexp.Regexp {
	if config.wholeWord {
		p = "\\b" + p + "\\b"
	}
	if config.ignoreCase {
		p = "(?i)" + p
	}
	return regexp.MustCompile(p)
}

func main() {
	config := parseArgs()
	var start int
	var pattern *regexp.Regexp
	if !config.descMode {
		start = 1
		pattern = mustCompile(flag.Arg(0), config)
	}
	errorCount := 0
	matchCount := 0
	found := false
	for i := start; i < flag.NArg(); i++ {
		if found {
			fmt.Println("---")
			found = false
		}
		f := flag.Arg(i)
		found, err := grep(pattern, f, config)
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
