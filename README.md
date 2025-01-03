# `gtee`

A Go implementation of `tee`, a GNU Coreutils tool. The original implementation in C can be checked on [here](https://github.com/coreutils/coreutils/blob/master/src/tee.c).

## Installation

The installation steps will be explained in detail once the initial version is ready to be distributed.

## Usage

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

# Ignore SIGINT for a during a long execution.
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
$ gtee [-h | --h | -help | --help]

NAME
	gtee - Duplicate standart input.

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

## The Implementation

The implemenation is done based on the POSIX specification of `tee`.

You can see the specification on [here](https://pubs.opengroup.org/onlinepubs/9799919799/).
Search for `tee` from index and you will see the document.

## Tests

Each test is written to satisfy a different part of the specification.
To see which requirement is tested in a given test, search the content of the `expected` variable on the specification.

## Benchmarks

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

## TODO

- Binary distribution & proper release tagging.
- Updating this README with installation steps.
- `man gtee`.
