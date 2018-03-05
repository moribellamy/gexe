package application

import (
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// An application handles the running, updating, and archiving of a user's software. Each application folder has
// this structure:
//
//  myapplication     # A top-level directory, conceptually an "Application"
//  └── archive       # Old releases
//  └── current
//      └── foo.exe   # Exactly one executable file
//      └── any.txt   # Other resources for foo.exe, ignored by gexe
//      └── any.dat
//  └── release       # Contents of a new release
//  └── command       # Files that appear here are interpreted as commands to gexe
type Application struct {
	dir     string            // absolute path on the filesystem to the corresponding app under management
	binary  os.FileInfo       // Resolved binary inside "current" dir
	watcher *fsnotify.Watcher // For the command directory
	daemon  *Daemon
}

// Return true if the given file seems to be an executable. TODO: Platform specific impls.
func isExecutable(file os.FileInfo) bool {
	const executable = os.FileMode(1)

	// HACK: Windows doesn't have the notion of executable files, it just tries to execute any file that
	// has the right file association.
	if strings.HasSuffix(file.Name(), ".exe") && runtime.GOOS == "windows" {
		return true
	}
	return file.Mode()&1 == executable && !file.IsDir()
}

// Return a slice of all files that seem to be executable in the given directory.
func getBinaries(dir string) ([]os.FileInfo, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var retval []os.FileInfo

	for _, file := range files {
		if isExecutable(file) {
			retval = append(retval, file)
		}
	}

	return retval, nil
}

func oneBinary(dir string) (os.FileInfo, error) {
	binaries, err := getBinaries(dir)
	if err != nil {
		return nil, fmt.Errorf("could not read directory %v: %v", dir, err)
	}
	if len(binaries) != 1 {
		return nil, fmt.Errorf("applicaton '%s' has missing or ambiguous binary", dir)
	}
	return binaries[0], nil
}

// Main control flow for an active application. The only way to leave this loop is if the program is terminated
// by the OS.
func (app *Application) Loop() {
	app.daemon = NewDaemon(app.BinaryPath())
	done := app.daemon.Loop()
	for {
		select {
		case event := <-app.watcher.Events:
			log.Printf("FSEVENT: %v\n", event)
			basename := filepath.Base(event.Name)
			if event.Op == fsnotify.Create {
				switch basename {
				case "release":
					app.daemon.Kill()
					<-done
					if err := app.doRelease(); err != nil {
						log.Printf("ERROR: Could not do release: %v", err)
					}
					app.refreshBinary()
					app.daemon = NewDaemon(app.BinaryPath())
					done = app.daemon.Loop()
				case "stop":
					app.daemon.Stop()
				case "start":
					app.refreshBinary()
					app.daemon.Start()
				default:
					log.Printf("Ignored unknown command: '%v' for app '%v'\n", basename, app.Name())
				}
				os.Remove(event.Name)
			}
		}
	}
}

func tryMove(source string, dest string) error {
	var err error
	log.Printf("Moving %v to %v\n", source, dest)
	for i := 0; i < 5; i++ {
		err = os.Rename(source, dest)
		if err == nil {
			break
		} else {
			log.Printf("ERROR moving files. Trying again soon...\n")
			time.Sleep(1 * time.Second)
		}
	}
	return err
}

// Does the folder shuffling that constitutes a release.
func (app *Application) doRelease() error {
	log.Printf("Running release on app: '%v'", app.Name())
	currentDir := filepath.Join(app.dir, "current")
	releaseDir := filepath.Join(app.dir, "release")
	archiveDir := filepath.Join(app.dir, "archive")
	if _, err := os.Stat(releaseDir); os.IsNotExist(err) {
		return errors.New("Release directory does not exist.\n")
	}
	if _, err := oneBinary(releaseDir); err != nil {
		return err
	}

	if _, err := os.Stat(archiveDir); os.IsNotExist(err) {
		log.Printf("Archive directory missing. Creating it...")
		os.Mkdir(archiveDir, os.ModePerm)
	}

	err := tryMove(currentDir, filepath.Join(archiveDir, fmt.Sprintf("%v-%d", app.Name(), time.Now().Unix())))
	if err != nil {
		return fmt.Errorf("could not archive: %v", err)
	}
	err = tryMove(releaseDir, currentDir)
	if err != nil {
		return fmt.Errorf("could not stage release: %v", err)
	}
	return nil
}

// Get the path (on local disk) of this application's binary.
func (app *Application) BinaryPath() string {
	return filepath.Join(app.dir, "current", app.binary.Name())
}

// Get the human friendly name for this application.
func (app *Application) Name() string {
	return filepath.Base(app.dir)
}

func (app *Application) refreshBinary() error {
	binary, err := oneBinary(filepath.Join(app.dir, "current"))
	if err != nil {
		return err
	}
	app.binary = binary
	return nil
}

// Construct a new Application based on the given directory (see godoc for Application struct).
func NewApplication(dir string) (*Application, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Cannot init file watcher: '%v'", err)
	}
	commandPath := filepath.Join(dir, "command")
	if _, err := os.Stat(commandPath); os.IsNotExist(err) {
		os.Mkdir(commandPath, os.ModePerm)
	} else {
		files, _ := ioutil.ReadDir(commandPath)
		for _, file := range files {
			if !file.IsDir() {
				os.Remove(filepath.Join(commandPath, file.Name()))
			}
		}
	}
	watcher.Add(commandPath)

	retval := &Application{
		dir:     dir,
		watcher: watcher,
	}
	if err := retval.refreshBinary(); err != nil {
		return nil, err
	}

	log.Printf("Initialized app '%v' with binary '%v'\n", retval.Name(), retval.BinaryPath())
	return retval, nil
}
