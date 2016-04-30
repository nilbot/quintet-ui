#!/usr/bin/env bash
# I know I have sudo; so whatever
sudo pkill quintet-ui
go build -o quintet-ui main.go
quintet-ui &>run.log &
disown

