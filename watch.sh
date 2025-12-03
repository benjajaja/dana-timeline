#!/usr/bin/env bash

watchexec -w . --ignore index.html "go run main.go timeline > index.html"

