#!/bin/sh
# https://carlosbecker.com/posts/golang-completions-cobra/
set -e
rm -rf manpages
mkdir manpages
go run ./ man | gzip -c -9 >manpages/acorn.1.gz