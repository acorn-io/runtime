#!/bin/sh

set -eu

ip=$(cat /etc/resolv.conf | grep nameserver | awk '{print $2}')
echo "resolver $ip valid=15s;" > /etc/nginx/conf.d/resolver.conf