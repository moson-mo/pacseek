#!/bin/sh

go test ./... -covermode=count -coverprofile=coverage.out fmt
go tool cover -func=coverage.out -o=test_coverage.out
go tool cover -html=coverage.out
