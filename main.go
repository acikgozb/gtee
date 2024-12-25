package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	go func(ctx context.Context) {
		<-ctx.Done()
		stop()

		fmt.Printf("gtee: sig %s\n", os.Interrupt)
		os.Exit(1)
	}(ctx)

	run()
}

func run() {
	if _, err := io.Copy(os.Stdout, os.Stdin); err != nil {
		log.Fatal(err)
	}
}
