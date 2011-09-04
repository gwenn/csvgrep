/*
Pretty-print lines matching a pattern in CSV files.
Transparent support for gzipped/bzipped files.
TODO
Make possible to customize output format 
Match only specific field by index or name
Match only whole field
Show header mode
Try to guess separator.

The author disclaims copyright to this source code.
*/
package main

import (
	"fmt"
	"flag"
	"log"
	"os"
	"strings"
	"strconv"
	"tabwriter"
	"exp/regexp"
	"github.com/gwenn/yacr"
)

type Config struct {
	fields     []uint
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
	return err == nil && (s.IsRegular() || s.IsSymlink())
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
	var d *bool = flag.Bool("d", false, "only show header/describe first line (no grep)")
	var q *bool = flag.Bool("q", true, "quoted field mode")
	var sep *string = flag.String("s", ",", "set the field separator")
	var f *string = flag.String("f", "", "set the field indexes to be matched (starts at 1)")
	var v *int = flag.Int("v", 1, "first column number in output/result")
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

	var fields []uint
	if len(*f) > 0 {
		rawFields := strings.Split(*f, ",")
		fields = make([]uint, len(rawFields))
		for i, s := range rawFields {
			f, err := strconv.Atoui(s)
			if err != nil {
				flag.Usage()
				log.Fatalf("Invalid field index (%v)\n", s)
			}
			fields[i] = f - 1
		}
	}
	return &Config{noHeader: *n, ignoreCase: *i, wholeWord: *w, sep: (*sep)[0], guess: guess, quoted: *q, start: *v, fields: fields, descMode: *d}
}

func match(fields []uint, pattern *regexp.Regexp, values [][]byte) bool {
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

func grep(pattern *regexp.Regexp, f string, config *Config) (found bool, err os.Error) {
	//fmt.Println(f, config)
	reader, err := yacr.NewFileReader(f, config.sep, config.quoted)
	if err != nil {
		return
	}
	reader.Guess = config.guess
	defer reader.Close()

	var headers [][]byte
	if config.noHeader && !config.descMode {
	} else {
		headers, err = reader.ReadRow()
		// TODO Try to guess/fix the separator if an error occurs (or if only one column is found)
		if err != nil {
			return
		}
		headers = yacr.DeepCopy(headers)
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

	for {
		values, err := reader.ReadRow()
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
		if err != nil {
			break
		}
	}
	return
}

func mustCompile(p string, config *Config) *regexp.Regexp {
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
		if err != nil && err != os.EOF {
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
