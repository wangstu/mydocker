#!/bin/bash

mkdir -p output/
rm -rf output/*

go build -o ./output/mydocker .