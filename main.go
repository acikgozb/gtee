package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
)

type config struct {
	ignore bool
	append bool
}

var (
	name          = "gtee"
	readBufSize   = bufio.MaxScanTokenSize
	ignoredSIGINT = fmt.Sprintf("%s: %s", name, "The SIGINT signal is ignored.")
)

func (c *config) parse() {
	flag.BoolVar(&c.ignore, "i", c.ignore, "Ignore the SIGINT signal.")
	flag.Parse()
}

func main() {
	cfg := config{
		ignore: false,
		append: false,
	}
	cfg.parse()

	if cfg.ignore {
		ignoreSIGINT()
	}

	gtee(&cfg)
}

func ignoreSIGINT() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		for {
			switch s := <-c; s {
			case os.Interrupt:
				fmt.Println(ignoredSIGINT)
			case syscall.SIGTERM:
				close(c)
				os.Exit(1)
			}
		}
	}()
}

func gtee(cfg *config) {
	fnames := flag.Args()
	fnames = append(fnames, os.Stdout.Name())


	var wg sync.WaitGroup
	rc := make(chan []byte)
	ec := make(chan error)

	for range fnames {
		wg.Add(1)
		go func(rc <-chan []byte, wg *sync.WaitGroup) {
			for b := range rc {
				fmt.Printf("Got chunk: %d", len(b))
			}
			wg.Done()
		}(rc, &wg)

	}

	wg.Add(1)
	go readStdin(rc, ec, &wg)

	wg.Wait()

}

func main() {
	cfg := config{
		ignore: false,
		append: false,
	}
	cfg.parse()
func readStdin(rc chan<- []byte, ec chan<- error, wg *sync.WaitGroup) {
	sc := bufio.NewScanner(os.Stdin)
	sc.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF {
			// Data may not be empty: https://pkg.go.dev/bufio#SplitFunc
			return 0, data, bufio.ErrFinalToken
		}

		if len(data) > readBufSize {
			return readBufSize, data[:readBufSize], nil
		}

		if len(data) <= readBufSize {
			return len(data), data, nil
		}

		return 0, nil, nil
	})

	for sc.Scan() {
		b := sc.Bytes()
		rc <- b
	}

	err := sc.Err()
	if err != nil {
		ec <- err
	}
	close(rc)
	wg.Done()
}
