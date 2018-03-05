package main

import (
	"github.com/moribellamy/gexe/runner"
	"log"
	"os"
	"path/filepath"
)

func main() {
	log.Printf("Started gexe, PID=%d PPID=%d", os.Getpid(), os.Getppid())
	deployment, err := filepath.Abs("./deployment")
	if err != nil {
		log.Fatal(err)
	}
	run, err := runner.NewRunner(deployment)
	if err != nil {
		log.Fatal(err)
	}
	run.Loop()

	select {}
}
