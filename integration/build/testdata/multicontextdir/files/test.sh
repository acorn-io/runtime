#!/bin/sh
set -e -x
test -f /token
[ "hi" = "$(cat /token)" ]
