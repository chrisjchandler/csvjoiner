# CSV Joiner

A lightweight Go-based utility to merge multiple CSV files using a common key column, even if the column headers differ across files.

## Features

- Merge any number of CSV files via command-line arguments
- Join rows based on a specified list of equivalent key column names
- Handles duplicate field names by appending suffixes (`_1`, `_2`, etc.)
- Outputs a single merged `output.csv` file

## Prerequisites

- [Go](https://golang.org/doc/install) installed on your machine (Go 1.16 or higher recommended)

## Configuration

Create a `config.json` file that lists all possible header names that should be treated as equivalent join keys. For example:


["req id", "req", "ReqID"]

This tells the joiner to match any column in any CSV file with a name equal to one of these (case-insensitive) and use it as the join key.

Usage
go run csvjoin.go -config config.json -out output.csv file1.csv file2.csv ...

Arguments
-config — Path to a JSON file listing join key column name variants

-out — Destination file for merged CSV output

Remaining arguments — Paths to input CSV files to join

go run csvjoin.go -config config.json -out merged.csv hiring.csv tracking.csv update.csv

Output
The resulting output.csv will contain:

All rows from all input files joined on the key column

All fields from all files (with suffixes added if duplicate columns exist)

Notes
This is a left-biased outer join — keys from all files are included, and values from later files are merged onto earlier keys.

Column matching is case-insensitive

Empty or missing values are preserved as blank
