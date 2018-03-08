# gexe

[![Documentation](https://godoc.org/github.com/yangwenmai/how-to-add-badge-in-github-readme?status.svg)](https://godoc.org/github.com/moribellamy/gexe)

gexe is a simple, cross-platform daemon to keep your binaries running. It
also supports a simple release flow.

At this time, gexe is _only_ recommended for hobby use (**not** production).
You might consider gexe if your binary isn't production critical, or if
you don't want to become expert in a new nanny proc every time you run on a new
platform.

## Usage

```
$ go get github.com/moribellamy/gexe
$ cp $GOPATH/src/github.com/moribellamy/gexe </next/to/your/deployment/folder>
$ ./gexe
```

## Alternatives

Consider your platform's main daemonization solution for production use.
Ubuntu has upstart, older linux has SYSV init, Windows has NSSM, Mac has
`brew services`, etc...

YAJSW seems to be cross-platform and mature.

There are also various devops solutions, like kubernetes, your cloud platform.


## The model

gexe manages a server "deployment" in a folder. The deployment folder is made
up of multiple "applications". Each application is a folder having this structure:
```
myapplication     # A top-level directory, conceptually an "Application"
└── archive       # Old releases
└── current
    └── foo.exe   # Exactly one executable file
    └── any.txt   # Other resources for foo.exe, ignored by gexe
    └── any.dat
└── release       # Contents of a new release
└── command       # Files that appear here are interpreted as commands to gexe
```

* The `current` dir is where your executable, and any files needed by it, live.
The executable is invoked with `CWD=<current>`
* The `release` dir is where you can stage a new release. Since it will
eventually be moved to `current`, you also want exactly one executable in
this folder.
* The `command` directory is for simple interface with the running gexe daemon.
To issue a command, create a file named...
  * `stop` to temporarily halt your app
  * `start` to restart your app after a call to stop
  * `release` to stop your app, archive it, stage the contents of the "release"
dir to the "current" dir, then start

## TODO
* Integration with per-system nannys, so gexe will restart even after a crash.
* Unit tests
* Entire component tests with vagrant
* Per application configuration