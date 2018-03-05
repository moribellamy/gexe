package application

import (
	"log"
	"os/exec"
	"path/filepath"
	"time"
)

type Daemon struct {
	binary string
	cmd    exec.Cmd
	exit   chan error
	stop   chan bool
	start  chan bool
	kill   chan bool
}

func NewDaemon(binary string) *Daemon {
	return &Daemon{
		binary: binary,
		start:  make(chan bool),
		stop:   make(chan bool),
		exit:   make(chan error),
		kill:   make(chan bool),
	}
}

func (daemon *Daemon) name() string {
	return filepath.Base(daemon.binary)
}

// Main activity loop for a daemon. Returns a channel to wait on when Kill() is called.
func (daemon *Daemon) Loop() chan bool {
	done := make(chan bool)
	go func() {
		daemon.loop()
		done <- true
	}()
	return done
}

// Run in a goroutine. Exits when the daemon has been killed by a call to Kill() below.
// Auto restarts on application termination.
func (daemon *Daemon) loop() {
	go daemon.Start()
	running := false
	requestKill := false
	for {
		select {
		case <-daemon.start:
			if running {
				log.Printf("Cannot start daemon '%v' which is already running.\n", daemon.name())
				continue
			}
			daemon.cmd = exec.Cmd{
				Path: daemon.binary,
				Dir:  filepath.Dir(daemon.binary),
			}
			daemon.cmd.Start()
			log.Printf("Started daemon '%v', PID=%v\n", daemon.name(), daemon.cmd.Process.Pid)
			running = true
			go func() { daemon.exit <- daemon.cmd.Wait() }()
		case err := <-daemon.exit:
			log.Printf("Daemon '%v' exited with '%v'\n", daemon.name(), err)
			running = false
			go func() {
				log.Printf("Requesting restart in 15s...\n")
				time.Sleep(15 * time.Second)
				daemon.Start()
			}()
		case <-daemon.kill:
			// Exit select loop. All channels now have no subscribers.
			log.Printf("Killing daemon for '%v'\n", daemon.name())
			requestKill = true
			daemon.Stop()
		case <-daemon.stop:
			log.Printf("Trying to stop '%v'\n", daemon.name())
			if running {
				if err := daemon.cmd.Process.Kill(); err != nil {
					log.Printf("Could not stop daemon process '%v': %v\n", daemon.name(), err)
				} else {
					log.Printf("Sent stop request to '%v', waiting for termination...\n", daemon.name())
					// Consume the termination -- we meant to stop it!
					<-daemon.exit
					log.Printf("'%v' terminated.", daemon.name())
				}
				running = false
			} else {
				log.Printf("Cannot stop daemon '%v' which has not started.", daemon.name())
			}
			if requestKill {
				return
			}
		}
	}
}

func (daemon *Daemon) Start() {
	go func() { daemon.start <- true }()
}

func (daemon *Daemon) Stop() {
	go func() { daemon.stop <- true }()
}

func (daemon *Daemon) Kill() {
	go func() { daemon.kill <- true }()
}
