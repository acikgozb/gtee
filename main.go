package main

import (
	"bufio"
	"context"
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
	signalSIGINTIsIgnored    = fmt.Sprintf("%s: the SIGINT signal is ignored", program)
	errCannotReadStdin       = fmt.Errorf("%s: cannot read stdin", program)
	errCannotOpenFileToWrite = fmt.Errorf("%s: cannot open file to write", program)
	errCannotWrite           = fmt.Errorf("%s: cannot write to file", program)
)

type config struct {
	ignore bool
	append bool
}

func (c *config) parse() {
	flag.BoolVar(&c.ignore, "i", c.ignore, "Ignore the SIGINT signal.")
	flag.BoolVar(&c.append, "a", c.append, "Append the output to the files rather than overwriting them.")
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

func gtee(ctx context.Context, append bool) (<-chan error, *sync.WaitGroup) {
	var wg sync.WaitGroup

	fc := len(flag.Args()) + 1
	bChans, errChan := readStdin(ctx, fc, &wg)

	for i, fname := range flag.Args() {
		f, err := openFile(fname, fileFlag(append))
		if err != nil {
			continue
		}
		writeFile(ctx, f, bChans[i], &wg)
	}

	writeFile(ctx, os.Stdout, bChans[len(bChans)-1], &wg)

	return errChan, &wg
}

func readStdin(ctx context.Context, fc int, wg *sync.WaitGroup) ([]chan []byte, <-chan error) {
	errChan := make(chan error, 1)
	bChans := make([]chan []byte, fc)
	for i := range bChans {
		bChans[i] = make(chan []byte)
	}

	close := func() {
		wg.Done()
		for _, bChan := range bChans {
			close(bChan)
		}
		close(errChan)
	}

	wg.Add(1)
	go func() {
		defer close()

		rbuf := make([]byte, readBufSize)
		bbuf := make([]byte, readBufSize)
		for {
			n, err := os.Stdin.Read(rbuf)
			if n == 0 && err == io.EOF {
				return
			}

			if ctx.Err() != nil {
				return
			}

			if err != nil {
				errChan <- fmt.Errorf("%s: %s", errCannotReadStdin, err)
				return
			}

			copy(bbuf, rbuf)
			for _, bChan := range bChans {
				bChan <- bbuf
			}
		}
	}()

	return bChans, errChan
}

func fileFlag(append bool) int {
	if append {
		return os.O_CREATE | os.O_APPEND | os.O_WRONLY
	}

	return os.O_CREATE | os.O_WRONLY
}

func openFile(fname string, flag int) (*os.File, error) {
	file, err := os.OpenFile(fname, flag, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s: %s", errCannotOpenFileToWrite, fname, err)
		return nil, err
	}

	return file, err
}

func writeFile(ctx context.Context, f *os.File, bChan <-chan []byte, wg *sync.WaitGroup) {
	wg.Add(1)

	go func(f *os.File, wg *sync.WaitGroup) {
		defer f.Close()
		defer wg.Done()

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
					fmt.Fprintf(os.Stderr, "%s: %s: %s", errCannotWrite, f.Name(), err)
					return
				}
			}
		}
	}(f, wg)
}

func main() {
	cfg := config{
		ignore: false,
		append: false,
	}
	cfg.parse()

	ctx, stopListening := listenSIGINT(cfg.ignore)
	defer stopListening()

	errCh, process := gtee(ctx, cfg.append)

	for err := range errCh {
		fmt.Fprint(os.Stderr, err.Error())
	}

	process.Wait()
}
