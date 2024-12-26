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

func (c *config) parse() {
	flag.BoolVar(&c.ignore, "i", c.ignore, "Ignore SIGINT")
	flag.Parse()
}

func ignoreSIGINT() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		for {
			switch s := <-c; s {
			case os.Interrupt:
				fmt.Println("\ngtee: SIGINT is suppressed.")
			case syscall.SIGTERM:
				close(c)
				os.Exit(1)
			}
		}
	}()
}

func gtee() {
	if _, err := io.Copy(os.Stdout, os.Stdin); err != nil {
		log.Fatal(err)
	}
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

	gtee()
}
