#!/bin/bash

set -e -x
while true; do
  curl --connect-timeout 3 -v http://service:82 && exit 0
  sleep 1
done
