#!/bin/sh
gzip -c test.csv > test.csv.gz
bzip2 -c test.csv > test.csv.bz2

$GOPATH/bin/csvgrep -s=, 'z' test.csv*

rm test.csv.gz
rm test.csv.bz2

echo
echo 'Test TAB'
tr ',' '\t' < test.csv > test.tsv

$GOPATH/bin/csvgrep 'z' test.tsv

rm test.tsv

echo
echo 'Test flags'
$GOPATH/bin/csvgrep -w -i 'Z' test.csv
$GOPATH/bin/csvgrep -i 'Z' test.csv
