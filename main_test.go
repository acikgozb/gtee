package main_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"testing"
)

// The test cases are mainly inspired by the POSIX specification of `tee`:
// https://pubs.opengroup.org/onlinepubs/9799919799/
// Search for `tee` to see the specification.

var gtee string

func build() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	bin := fmt.Sprintf("%s/gtee", wd)
	cmd := exec.Command("go", "build", "-o", bin, ".")
	if err := cmd.Run(); err != nil {
		return err
	}

	gtee = bin

	return nil
}

func cleanup(fname string) error {
	if err := os.Remove(fname); err != nil {
		return err
	}

	return nil
}

func TestMain(t *testing.M) {
	err := build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot build the binary to test: %s\n", err)
		os.Exit(1)
	}

	code := t.Run()

	if err := cleanup(gtee); err != nil {
		fmt.Fprintf(os.Stderr, "cannot remove the binary: %s\n", err)
		os.Exit(1)
	}

	os.Exit(code)
}

func TestStdout(t *testing.T) {
	expected := []byte("The standard output shall be a copy of standard input.")
	cmd := exec.Command(gtee)
	cmd.Stdin = bytes.NewReader(expected)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	if string(expected) != string(out) {
		t.Fatalf("expected %s but got %s", string(expected), string(out))
	}
}

func TestStderr(t *testing.T) {
	expected := []byte("The standard error shall be used only for diagnostic messages.")

	cases := []struct {
		name string
		err  bool
		file func() string
	}{
		{
			name: "writeErr",
			err:  true,
			file: func() string {
				dir, err := os.MkdirTemp("", "dir")
				if err != nil {
					t.Fatalf("Could not create a temp dir: %s", err)
				}
				return dir
			},
		},
		{
			name: "noErr",
			err:  false,
			file: func() string {
				f, err := os.CreateTemp("", "file")
				if err != nil {
					t.Fatalf("Could not create a temp file: %s", err)
				}

				return f.Name()
			},
		},
	}

	for _, c := range cases {
		var outbuf bytes.Buffer
		var errbuf bytes.Buffer

		f := c.file()
		defer cleanup(f)

		cmd := exec.Command(gtee, f)

		cmd.Stdin = bytes.NewReader(expected)
		cmd.Stdout = &outbuf
		cmd.Stderr = &errbuf

		if err := cmd.Run(); err != nil {
			t.Fatalf("Expected to run cmd for %s, but got %s", c.name, err)
		}

		outb := outbuf.Bytes()
		errb := errbuf.Bytes()

		if c.err && bytes.Contains(outb, errb) {
			t.Fatalf("Expected stdout to not contain stderr: stdout %s, stderr %s", outb, errb)
		}

		if !c.err && len(errb) > 0 {
			t.Fatalf("Expected stderr to be empty, but got %s", errb)
		}
	}
}

func TestCopy(t *testing.T) {
	expected := []byte("If any file operands are specified, the standard input shall be copied to each named file.")

	f, err := os.CreateTemp("", "file")
	if err != nil {
		t.Fatalf("Expected to create a temp file, but got %s", err)
	}

	defer cleanup(f.Name())

	var outbuf bytes.Buffer
	var errbuf bytes.Buffer

	cmd := exec.Command(gtee, f.Name())

	cmd.Stdin = bytes.NewReader(expected)
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	if err := cmd.Run(); err != nil {
		t.Fatalf("Expected to run cmd, but got %s", err)
	}

	errb := errbuf.Bytes()
	if len(errb) > 0 {
		t.Fatalf("Expected to have no errors but got %s", errb)
	}

	outb := outbuf.Bytes()
	if !slices.Equal(expected, outb) {
		t.Fatalf("Expected stdin and stdout to be equal: stdin %s, stdout %s", expected, outb)
	}

	f, err = os.Open(f.Name())
	if err != nil {
		t.Fatalf("Expected to open temp file after copying, but got %s", err)
	}

	fb := make([]byte, len(expected))
	if _, err = f.Read(fb); err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("Expected to read temp file after copying, but got %s", err)
	}

	if !slices.Equal(expected, fb) {
		t.Fatalf("Expected stdin and file to be equal: stdin %s, file %s", expected, fb)
	}
}
