#!/bin/bash -e
curl -X GET -s http://localhost:9127/metrics | awk '$1 ~ /^[^#]/ ' | awk '{print $1;}' | diff $1 -

