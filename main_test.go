package main_test

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
)

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

func cleanup() error {
	if err := os.Remove(gtee); err != nil {
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

	if err := cleanup(); err != nil {
		fmt.Fprintf(os.Stderr, "cannot remove the binary: %s\n", err)
		os.Exit(1)
	}

	os.Exit(code)
}

func TestGtee(t *testing.T) {
	fmt.Println("super complicated test right here.")
}
