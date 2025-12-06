#!/usr/bin/env bash

if [ ! -f ./tailwindcss-linux-x64 ]; then
    curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64
    chmod +x tailwindcss-linux-x64
fi

./tailwindcss-linux-x64 --content "./index.html,./main.go" -o style.css --minify

watchexec -w . --ignore index.html --ignore style.css "go run main.go timeline > index.html"

