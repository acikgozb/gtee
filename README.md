# `gtee`

A Go implementation of `tee`, a GNU Coreutils tool. The original implementation in C can be checked on [here](https://github.com/coreutils/coreutils/blob/master/src/tee.c).

## Table of Contents

<!--toc:start-->

- [Installation](#a-idinstallation-installation)
  - [`go install`](#go-install)
  - [Prebuilt Binaries](#prebuilt-binaries)
- [Usage](#a-idusage-usage)
- [The Implementation](#a-idthe-implementation-the-implementation)
- [Tests](#a-idtests-tests)
- [Benchmarks](#a-idbenchmarks-benchmarks)
- [TODO](#a-idtodo-todo)
<!--toc:end-->

## <a id='installation' /> Installation

`gtee` is available on Linux and MacOS.

### <a id='go-install' /> `go install`

As a Go project, `gtee` can be installed via `go install`, like below:

```bash
go install github.com/acikgozb/gtee
```

### <a id='prebuilt-binaries' /> Prebuilt Binaries

If you wish to not install Go on your machine just to install `gtee`, you can download a prebuilt binary from the release page.
The binaries are available for the platforms below:

- _x86_64_ (amd64) Linux
- _arm64_ Darwin (macOS)

You can download the version you wish to use and then extract the binary from the archive to wherever you want.

Once downloaded and put into a directory, you can verify the installation by simply running:

```bash
gtee -h
```

If you see the usage, then you are all set!

Don't forget to make sure the binary is under `$PATH` to be able to use it without manually entering the location of the binary itself.
To verify whether `gtee` is under path, you can run:

```bash
which gtee
```

If `which` returns a path, that means you can successfully use `gtee` without specifying the full path of the binary.

## <a id='usage' /> Usage

`gtee` is designed to be used as a drop in replacement of `tee` with the exception of:

- Merging shorthand options like `-ai`,
- Environment variables that alter the language of diagnostic messages.

Here are the supported usages:

```bash
# Write to both stdout and a file named "example.txt".
$ echo "write me!" | gtee example.txt

# Use input redirection and write "input.txt" to both stdout and a file named "example.txt".
$ gtee example.txt < input.txt

# Write "input.txt" only to a file named "example.txt", not stdout.
$ gtee example.txt > /dev/null < input.txt

# Append only to a file named "example.txt".
$ echo "append me!" | gtee -a example.txt > /dev/null
$ echo "append me!" | gtee --a example.txt > /dev/null
$ echo "append me!" | gtee -append example.txt > /dev/null
$ echo "append me!" | gtee --append example.txt > /dev/null

# Ignore SIGINT during a long execution.
$ echo "write me, a super long input!" | gtee -i example.txt > /dev/null
$ echo "write me, a super long input!" | gtee --i example.txt > /dev/null
$ echo "write me, a super long input!" | gtee -ignore example.txt > /dev/null
$ echo "write me, a super long input!" | gtee --ignore example.txt > /dev/null

# Attempting to send a SIGINT signal results in a diagnostic message on stdout, not err.
$ echo "write me, a super long input!" | gtee -i example.txt > /dev/null
$ ^Cgtee: The SIGINT signal is ignored.

# Append to a file named "example.txt" and ignore the SIGINT signal.
$ echo "append me, a super long input!" | gtee -a -i example.txt > /dev/null
$ echo "append me, a super long input!" | gtee --a --i example.txt > /dev/null
$ echo "append me, a super long input!" | gtee -append -ignore example.txt > /dev/null
$ echo "append me, a super long input!" | gtee --append --ignore example.txt > /dev/null
```

Running `gtee` with `-h` or with a non-supported flag will result in showing the usage of the program, which is inspired from the man page of `tee`:

```bash
# Show usage:
$ gtee -h # --h, -help, --help

NAME
	gtee - Duplicate standard input.

SYNOPSIS
	gtee [-a] [-i] [file ...]

OPTIONS
	-i, --i, -ignore, --ignore	Ignore the SIGINT signal.
	-a, --a, -append, --append	Append the output to the files rather than overwriting them.

EXAMPLES
Send the echoed message to both stdout and a file called greetings.txt:

	$ echo "Hello" | gtee greetings.txt
	Hello
```

## <a id='the-implementation' /> The Implementation

The implemenation is done based on the POSIX specification of `tee`.

You can see the specification on [here](https://pubs.opengroup.org/onlinepubs/9799919799/).
Search for `tee` from index and you will see the document.

## <a id='tests' /> Tests

Each test is written to satisfy a different part of the specification.
To see which requirement is tested in a given test, search the content of the `expected` variable on the specification.

## <a id='benchmarks' /> Benchmarks

Currently, there is only one simple benchmark which compares both `tee` and `gtee` on a 100MB file created by `dd`.
Check out `BenchmarkTee` and `BenchmarkGtee` for more information.

Here is a sample comparison to showcase a result.
Keep in mind that the times you get on your own host may not match with these:

```bash
$ go test -bench=. -benchtime 100x -benchmem
pkg: github.com/acikgozb/gtee
BenchmarkTee-10     	     100	  55095693 ns/op	    5940 B/op	      22 allocs/op
BenchmarkGtee-10    	     100	  49339303 ns/op	    5988 B/op	      22 allocs/op
PASS
ok  	github.com/acikgozb/gtee	12.470s
```

## <a id='todo' /> TODO

- `man gtee`.
