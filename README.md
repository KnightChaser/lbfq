# lbfq

> **L**ist **B**ig **F**iles **Q**uickly!

## What is this for

`lbfq` is a command-line tool written in Go that scans a directory tree and identifies the largest files. It can be useful for disk usage analysis, finding space-consuming files, and cleaning up storage.

## How to compile

Ensure you have Go installed on your system. Then, in the project directory, run:

```
go build
```

This will create an executable binary named `lbfq` (or `lbfq.exe` on Windows).

## How to use

Run the compiled binary with the desired options. For example:

```sh
sudo ./lbfq -root /path/to/directory -n 10 -min 100M
```

Options can be found by running `./lbfq --help`. 