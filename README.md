LFS Test Server
======

[rel]: https://github.com/github/lfs-test-server/releases
[lfs]: https://github.com/github/git-lfs
[api]: https://github.com/github/git-lfs/blob/master/docs/api.md

LFS Test Server is an example server that implements the [Git LFS API][api]. It
is intended to be used for testing the [Git LFS][lfs] client and is not in a
production ready state.

LFS Test Server is written in Go, with pre-compiled binaries available for Mac,
Windows, Linux, and FreeBSD.

See [CONTRIBUTING.md](CONTRIBUTING.md) for info on working on LFS Test Server and
sending patches.

## Installing

Download the [latest version][rel]. It is a single binary file.

Alternatively, use the Go installer:

```
  $ go install github.com/github/lfs-test-server
```


## Building

To build from source, use the Go tools:

```
  $ go get github.com/github/lfs-test-server
```


## Running

Running the binary will start an LFS server on `localhost:8080` by default.
There are few things that can be configured via environment variables:

	LFS_LISTEN      # The address:port the server listens on, default: "tcp://:8080"
	LFS_HOST        # The host used when the server generates URLs, default: "localhost:8080"
	LFS_METADB      # The database file the server uses to store meta information, default: "lfs.db"
	LFS_CONTENTPATH # The path where LFS files are store, default: "lfs-content"
	LFS_ADMINUSER   # An administrator username, default: unset
	LFS_ADMINPASS   # An administrator password, default: unset

If the `LFS_ADMINUSER` and `LFS_ADMINPASS` variables are set, a
rudimentary admin interface can be accessed via
`$LFS_SCHEME://$LFS_HOST/mgmt`. Here you can add and remove users.

To use the LFS test server with the Git LFS client, configure it in the repository's `.gitconfig` file:

```
  [lfs]
    url = "http://localhost:8080/janedoe/lfsrepo"
```
