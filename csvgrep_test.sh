#!/bin/sh
gzip -c test.csv > test.csv.gz
bzip2 -c test.csv > test.csv.bz2

./csvgrep -s=, 'z' test.csv*

rm test.csv.gz
rm test.csv.bz2
