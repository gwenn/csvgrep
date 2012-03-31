#!/bin/sh
gzip -c test.csv > test.csv.gz
bzip2 -c test.csv > test.csv.bz2

go run csvgrep.go -s=, 'z' test.csv*

rm test.csv.gz
rm test.csv.bz2

echo
echo 'Test TAB'
tr ',' '\t' < test.csv > test.tsv

go run csvgrep.go 'z' test.tsv

rm test.tsv

echo
echo 'Test flags'
go run csvgrep.go -w -i 'Z' test.csv
go run csvgrep.go -i 'Z' test.csv
