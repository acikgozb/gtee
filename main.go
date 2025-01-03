package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	program                  = "gtee"
	readBufSize              = bufio.MaxScanTokenSize
	flagIgnoreDescription    = "Ignore the SIGINT signal."
	flagAppendDescription    = "Append the output to the files rather than overwriting them."
	signalSIGINTIsIgnored    = fmt.Sprintf("%s: the SIGINT signal is ignored\n", program)
	errCannotReadStdin       = fmt.Errorf("%s: cannot read stdin", program)
	errCannotOpenFileToWrite = fmt.Errorf("%s: cannot open file to write", program)
	errCannotWrite           = fmt.Errorf("%s: cannot write to file", program)
)

type config struct {
	ignore bool
	append bool
}

func (c *config) parse() {
	flag.BoolVar(&c.ignore, "i", c.ignore, flagIgnoreDescription)
	flag.BoolVar(&c.ignore, "ignore", c.ignore, flagIgnoreDescription)
	flag.BoolVar(&c.append, "a", c.append, flagAppendDescription)
	flag.BoolVar(&c.append, "append", c.append, flagAppendDescription)

	bold := func(s string) string {
		return fmt.Sprintf("\033[1m%s\033[0m", s)
	}

	flag.Usage = func() {
		fmt.Printf("%s", bold("NAME"))
		fmt.Printf("\n")
		fmt.Printf("\t%s - Duplicate standard input.\n", program)
		fmt.Printf("\n")
		fmt.Printf("%s", bold("SYNOPSIS"))
		fmt.Printf("\n")
		fmt.Printf("\t%s [-a] [-i] [file ...]\n", bold(program))
		fmt.Printf("\n")
		fmt.Printf("%s\n", bold("OPTIONS"))
		fmt.Printf("\t%s\t%s\n", bold("-i, --i, -ignore, --ignore"), flagIgnoreDescription)
		fmt.Printf("\t%s\t%s\n", bold("-a, --a, -append, --append"), flagAppendDescription)
		fmt.Printf("\n")
		fmt.Printf("%s\n", bold("EXAMPLES"))
		fmt.Printf("Send the echoed message to both stdout and a file called greetings.txt:\n\n")
		fmt.Printf("\t$ echo \"Hello\" | %s greetings.txt\n", program)
		fmt.Printf("\tHello\n")
	}

	flag.Parse()
}

func listenSIGINT(ignore bool) (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT)

	stop := func() {
		signal.Stop(sigc)
		close(sigc)
	}

	go func(sigc chan os.Signal) {
		for range sigc {
			if ignore {
				fmt.Println(signalSIGINTIsIgnored)
				continue
			}

			cancel()
			stop()
		}
	}(sigc)

	return ctx, stop
}

func run(ctx context.Context, append bool) <-chan error {
	var wg sync.WaitGroup

	errChan := make(chan error)

	fnames := getFnames(flag.Args())
	fs := openFiles(fnames, getFlag(append), errChan, &wg)

	bChans := readStdin(ctx, len(fs), errChan, &wg)

	for i, f := range fs {
		writeFile(ctx, f, bChans[i], errChan, &wg)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	return errChan
}

func openFiles(fnames []string, flag int, errChan chan<- error, wg *sync.WaitGroup) []*os.File {
	fs := make([]*os.File, 0)

	for _, fname := range fnames {
		f, err := os.OpenFile(fname, flag, 0644)
		if err != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				errChan <- fmt.Errorf("%s: %s: %s", errCannotOpenFileToWrite, fname, err)
			}()

			continue
		}
		fs = append(fs, f)
	}

	return append(fs, os.Stdout)
}

func readStdin(ctx context.Context, fc int, errChan chan<- error, wg *sync.WaitGroup) []chan []byte {
	bChans := make([]chan []byte, fc)
	for i := range bChans {
		bChans[i] = make(chan []byte)
	}

	close := func() {
		for _, bChan := range bChans {
			close(bChan)
		}
		wg.Done()
	}

	wg.Add(1)
	go func() {
		defer close()

		rbuf := make([]byte, readBufSize)
		bbuf := make([]byte, readBufSize)
		for {
			n, err := os.Stdin.Read(rbuf)
			if n == 0 && errors.Is(err, io.EOF) {
				return
			}

			if ctx.Err() != nil {
				return
			}

			if err != nil && !errors.Is(err, io.EOF) {
				errChan <- fmt.Errorf("%s: %s", errCannotReadStdin, err)
				return
			}

			copy(bbuf, rbuf)

			if n < len(bbuf) {
				bbuf = bytes.Trim(bbuf, "\x00")
			}

			for _, bChan := range bChans {
				bChan <- bbuf
			}
		}
	}()

	return bChans
}

func getFnames(args []string) []string {
	fnames := make([]string, 0)
	m := make(map[string]bool, 0)

	for _, fname := range args {
		if _, ok := m[fname]; !ok {
			m[fname] = true
			fnames = append(fnames, fname)
		}
	}
	return fnames
}

func getFlag(append bool) int {
	if append {
		return os.O_CREATE | os.O_APPEND | os.O_WRONLY
	}
	return os.O_CREATE | os.O_WRONLY
}

func writeFile(ctx context.Context, f *os.File, bChan <-chan []byte, errChan chan<- error, wg *sync.WaitGroup) {
	wg.Add(1)

	go func(f *os.File, wg *sync.WaitGroup) {
		defer wg.Done()
		defer f.Close()
		defer f.Sync()

		for {
			select {
			case <-ctx.Done():
				return
			case b, ok := <-bChan:
				if !ok {
					return
				}

				_, err := f.Write(b)
				if err != nil {
					errChan <- fmt.Errorf("%s: %s: %s", errCannotWrite, f.Name(), err)
				}
			}
		}
	}(f, wg)
}

func gtee() int {
	var code int

	cfg := config{
		ignore: false,
		append: false,
	}
	cfg.parse()

	ctx, stopListening := listenSIGINT(cfg.ignore)
	defer stopListening()

	errCh := run(ctx, cfg.append)

	for err := range errCh {
		fmt.Fprint(os.Stderr, err.Error())
		code = 1
	}

	return code
}

func main() {
	os.Exit(gtee())
}
