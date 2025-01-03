package main_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"syscall"
	"testing"
	"time"
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
		fmt.Fprintf(os.Stderr, "cannot build the binary to test: %q\n", err)
		os.Exit(1)
	}

	code := t.Run()

	if err := cleanup(gtee); err != nil {
		fmt.Fprintf(os.Stderr, "cannot remove the binary: %q\n", err)
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
		t.Fatalf("Expected to have no errors but got %q", err)
	}

	if string(expected) != string(out) {
		t.Fatalf("Expected %q but got %q", string(expected), string(out))
	}
}

func TestStderr(t *testing.T) {
	expected := []byte("The standard error shall be used only for diagnostic messages.")

	cases := []struct {
		name  string
		isErr bool
		file  func() string
	}{
		{
			name:  "writeErr",
			isErr: true,
			file: func() string {
				dir, err := os.MkdirTemp("", "dir")
				if err != nil {
					t.Fatalf("Could not create a temp dir: %q", err)
				}
				return dir
			},
		},
		{
			name:  "noErr",
			isErr: false,
			file: func() string {
				f, err := os.CreateTemp("", "file")
				if err != nil {
					t.Fatalf("Could not create a temp file: %q", err)
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

		if err := cmd.Run(); err != nil && !c.isErr {
			t.Fatalf("Expected to run the cmd for %q, but got %q", c.name, err)
		}

		outb := outbuf.Bytes()
		errb := errbuf.Bytes()

		if c.isErr && bytes.Contains(outb, errb) {
			t.Fatalf("Expected stdout to not contain stderr: stdout %q, stderr %q", outb, errb)
		}

		if !c.isErr && len(errb) > 0 {
			t.Fatalf("Expected stderr to be empty, but got %q", errb)
		}
	}
}

func TestCopy(t *testing.T) {
	expected := []byte("If any file operands are specified, the standard input shall be copied to each named file.")

	f, err := os.CreateTemp("", "file")
	if err != nil {
		t.Fatalf("Expected to create a temp file, but got %q", err)
	}

	defer cleanup(f.Name())

	var outbuf bytes.Buffer
	var errbuf bytes.Buffer

	cmd := exec.Command(gtee, f.Name())

	cmd.Stdin = bytes.NewReader(expected)
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	if err := cmd.Run(); err != nil {
		t.Fatalf("Expected to run the cmd, but got %q", err)
	}

	errb := errbuf.Bytes()
	if len(errb) > 0 {
		t.Fatalf("Expected to have no errors but got %q", errb)
	}

	outb := outbuf.Bytes()
	if !slices.Equal(expected, outb) {
		t.Fatalf("Expected stdin and stdout to be equal: stdin %q, stdout %q", expected, outb)
	}

	f, err = os.Open(f.Name())
	if err != nil {
		t.Fatalf("Expected to open the temp file after copying, but got %q", err)
	}

	fb := make([]byte, len(expected))
	if _, err = f.Read(fb); err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("Expected to read the temp file after copying, but got %q", err)
	}

	if !slices.Equal(expected, fb) {
		t.Fatalf("Expected stdin and file to be equal: stdin %q, file %q", expected, fb)
	}
}

func TestHyphenFileOperand(t *testing.T) {
	expected := []byte("If a file operand is '-', it shall refer to a file named '-'; implementations shall not treat it as meaning standard output.")
	fname := "-"

	cmd := exec.Command(gtee, fname)

	var errbuf bytes.Buffer

	cmd.Stdin = bytes.NewReader(expected)
	cmd.Stdout = nil
	cmd.Stderr = &errbuf

	if err := cmd.Run(); err != nil {
		t.Fatalf("Expected to run the cmd, but got %q", err)
	}

	errb := errbuf.Bytes()
	if len(errb) > 0 {
		t.Fatalf("Expected to have no errors but got %q", errb)
	}

	f, err := os.Open(fname)
	if err != nil {
		t.Fatalf("Expected to have a file named %s, but got %q", fname, err)
	}
	defer cleanup(fname)

	fb := make([]byte, len(expected))
	if _, err = f.Read(fb); err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("Expected to read a file named %s, but got %q", fname, err)
	}

	if !slices.Equal(expected, fb) {
		t.Fatalf("Expected file and stdin to be equal, but got %q", err)
	}
}

func TestFileOperands(t *testing.T) {
	expected := []byte("Processing of at least 13 file operands shall be supported.")

	fnames := make([]string, 13)
	for i := 0; i < 13; i++ {
		fnames[i] = fmt.Sprintf("file%d", i)
	}

	var errbuf bytes.Buffer
	var outbuf bytes.Buffer

	cmd := exec.Command(gtee, fnames...)

	cmd.Stdin = bytes.NewReader(expected)
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	if err := cmd.Run(); err != nil {
		t.Fatalf("Expected to run the cmd, but got %q", err)
	}

	errb := errbuf.Bytes()
	if len(errb) > 0 {
		t.Fatalf("Expected to have no errors but got %q", errb)
	}

	outb := outbuf.Bytes()
	if !slices.Equal(expected, outb) {
		t.Fatalf("Expected stdin and stdout to be equal: stdin: %q, stdout: %q", expected, outb)
	}

	for _, fname := range fnames {
		defer cleanup(fname)

		f, err := os.Open(fname)
		if err != nil {
			t.Fatalf("Expected to open the file %q, but got %q", fname, err)
		}

		rb := make([]byte, len(expected))
		if _, err = f.Read(rb); err != nil && !errors.Is(err, io.EOF) {
			t.Fatalf("Expected to read the file %q, but got %q", fname, err)
		}

		if !slices.Equal(expected, rb) {
			t.Fatalf("Expected stdin and file to be equal, stdin: %q, file: %q", expected, rb)
		}
	}
}

func TestExitCodes(t *testing.T) {
	cases := []struct {
		name string
		code int
		file func() string
	}{
		{
			name: "An error occured.",
			code: 1,
			file: func() string {
				dir, err := os.MkdirTemp("", "dir")
				if err != nil {
					t.Fatalf("Could not create a temp dir: %q", err)
				}

				return dir
			},
		},
		{
			name: "The standard input was successfully copied to all output files.",
			code: 0,
			file: func() string {
				f, err := os.CreateTemp("", "file")
				if err != nil {
					t.Fatalf("Could not create a temp dir: %q", err)
				}

				return f.Name()
			},
		},
	}

	for _, c := range cases {
		expected := []byte(c.name)

		f := c.file()
		defer cleanup(f)

		cmd := exec.Command(gtee, f)

		var outbuf bytes.Buffer
		var errbuf bytes.Buffer

		cmd.Stdin = bytes.NewReader(expected)
		cmd.Stdout = &outbuf
		cmd.Stderr = &errbuf

		err := cmd.Run()
		errb := errbuf.Bytes()

		if c.code == 0 {
			if err != nil {
				t.Fatalf("Expected to get no cmd err for code 0 but got %q", err)
			}

			if len(errb) > 0 {
				t.Fatalf("Expected stderr to be empty for code 0, but got %q", errb)
			}
		}

		if c.code > 0 {
			if err == nil {
				t.Fatalf("Expected to get a cmd err for code >0 but got %q", err)
			}

			if len(errb) == 0 {
				t.Fatal("Expected stderr to have a diagnostic message for code >0, but got nothing")
			}
		}
	}
}

func TestAppend(t *testing.T) {
	expected := []byte("Append the output to the files with -a.")

	f, err := os.Create("file")
	if err != nil {
		t.Fatalf("Could not create a temp file: %q", err)
	}
	defer cleanup(f.Name())

	if _, err = f.Write(expected); err != nil {
		t.Fatalf("Could not write on a temp file: %q", err)
	}

	f.Sync()
	f.Close()

	cmd := exec.Command(gtee, "-a", f.Name())

	var errbuf bytes.Buffer

	cmd.Stdin = bytes.NewReader(expected)
	cmd.Stdout = nil
	cmd.Stderr = &errbuf

	if err = cmd.Run(); err != nil {
		t.Fatalf("Expected to run the cmd, but got %q", err)
	}

	errb := errbuf.Bytes()
	if len(errb) > 0 {
		t.Fatalf("Expected to have no errors but got %q", errb)
	}

	tf, err := os.Open(f.Name())
	if err != nil {
		t.Fatalf("Could not open the temp file after appending it, but got %q", err)
	}

	rb := make([]byte, len(expected)*2)
	if _, err := tf.Read(rb); err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("Expected to read the temp file after appending it, but got %q", err)
	}

	trimmed := bytes.Trim(rb, "\x00")
	if len(trimmed) != len(rb) {
		t.Fatalf("Expected byte count %d to double after appending, but got %d", len(expected), len(trimmed))
	}
}

func TestIgnore(t *testing.T) {
	expected := []byte("If the -i option was specified, SIGINT shall be ignored.")

	cmd := exec.Command(gtee, "-i")

	var outbuf bytes.Buffer
	var errbuf bytes.Buffer
	r, w := io.Pipe()

	cmd.Stdin = r
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	// It is expected to ignore interrupts on long running processes,
	// Therefore a long running write and an interrupt signal are simulated
	// via context.WithTimeout.
	sigCtx, sigCancel := context.WithTimeout(context.Background(), time.Millisecond*250)
	wCtx, wCancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer sigCancel()
	defer wCancel()

	if err := cmd.Start(); err != nil {
		t.Fatalf("Expected to start the cmd but got %q", err)
	}

	go func() {
		defer w.Close()

		<-sigCtx.Done()
		cmd.Process.Signal(syscall.SIGINT)

		<-wCtx.Done()
		w.Write(expected)
	}()

	if err := cmd.Wait(); err != nil {
		t.Fatalf("Expected to run the cmd but got %q", err)
	}

	errb := errbuf.Bytes()
	if len(errb) > 0 {
		t.Fatalf("Expected to have no errors but got %q", errb)
	}

	outb := outbuf.Bytes()
	if len(outb) == len(expected) {
		t.Fatalf("Expected to have a message about SIGINT on stdout but got %q", outb)
	}
}
