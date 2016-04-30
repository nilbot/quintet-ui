#!/usr/bin/env bash
# I know I have sudo; so whatever
sudo pkill quintet-ui
openssl genrsa -out key 2048
openssl req -new -x509 -key key -out crt -days 365
GOPATH=$HOME go build -o quintet-ui main.go
./quintet-ui &>run.log &
disown

