Harbour
======

Harbour is a simple sever that implements the [Git LFS](https://github.com/github/git-lfs) API. Harbour is intended
to be used for testing the Git LFS client and is not in a production ready
state.

Harbour is written in Go, with pre-compiled binaries available vor Mac,
Windows, Linux, and FreeBSD.

See [CONTRIBUTING.md](CONTRIBUTING.md) for info on working on Harbour and
sending patches.

## Installing

Download the [latest version][rel]. It is a single binary file.

Alternatively, use the Go installer:

  $ go install github.com/github/harbour


## Building

To build from source, use the Go tools:

  $ go get github.com/github/harbour


## Running

Running the binary will start a Harbour service on `localhost:8080` by default.
There are few things that can be configured via environment variables:

	HARBOUR_LISTEN      # The address:port harbour listens on, default: "tcp://:8080"
	HARBOUR_HOST        # The host used when harbour generates URLs, default: "localhost:8080"
	HARBOUR_SCHEME      # The scheme used when harbour generates URLs, default: "https"
	HARBOUR_METADB      # The database file harbour uses to store meta information, default: "lfs.db"
	HARBOUR_CONTENTPATH # The path where LFS files are store, default: "lfs-content"
	HARBOUR_ADMINUSER   # An administrator username, default: unset
	HARBOUR_ADMINPASS   # An administrator password, default: unset

If the `HARBOUR_ADMINUSER` and `HARBOUR_ADMINPASS` variables are set, a
rudimentary admin interface can be accessed via
`$HARBOUR_SCHEME://$HARBOUR_HOST/mgmt`. Here you can add and remove users.

To use the harbour server with the Git LFS client, configure it in the repository's `.gitconfig` file:

  [lfs]
    url = "http://localhost:8080/janedoe/lfsrepo

