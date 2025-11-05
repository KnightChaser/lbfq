# lbfq

> **L**ist **B**ig **F**iles **Q**uickly!

<img width="1000" height="330" alt="image" src="https://github.com/user-attachments/assets/6b3f537e-9b50-4448-9739-38d6b7dfd8e6" />


## What is this for

`lbfq` is a command-line tool written in Go that scans a directory tree and identifies the largest files. It can be useful for disk usage analysis, finding space-consuming files, and cleaning up storage.

## How to compile

Ensure you have Go installed on your system. Then, in the project directory, run:

```
go build -o lbfq
```

This will create an executable binary named `lbfq` (or `lbfq.exe` on Windows).

## How to use

Run the compiled binary with the desired options. For example:

```sh
sudo ./lbfq -root /path/to/directory -n 10 -exclude-globs "/usr/share/ollama/*,*.safetensors"
```

Options can be found by running `./lbfq --help`.

