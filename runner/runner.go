package runner

import (
	"fmt"
	"github.com/moribellamy/gexe/application"
	"io/ioutil"
	"log"
	"path/filepath"
)

type Runner struct {
	dir string // Top level directory, containing multiple gexe/application/Application objects. Must be absolute.
}

func NewRunner(dir string) (*Runner, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if abs != dir {
		return nil, fmt.Errorf("given directory '%v' is not absolute", dir)
	}
	return &Runner{
		dir: dir,
	}, nil
}

func (runner *Runner) Loop() {
	files, err := ioutil.ReadDir(runner.dir)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		app, err := application.NewApplication(filepath.Join(runner.dir, f.Name()))
		if err != nil {
			log.Println(err)
		} else {
			go app.Loop()
		}
	}
}
