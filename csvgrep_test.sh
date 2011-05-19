#!/bin/sh
gzip -c test.csv > test.csv.gz
bzip2 -c test.csv > test.csv.bz2

./csvgrep -s=, 'z' test.csv*

rm test.csv.gz
rm test.csv.bz2

echo
echo 'Test TAB'
tr ',' '\t' < test.csv > test.tsv

./csvgrep -t 'z' test.tsv

rm test.tsv
