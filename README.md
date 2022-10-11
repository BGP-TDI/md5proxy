# md5proxy

This is a simple TCP-MD5 proxy implemented in 2021 for BGP research.

It listens on a (non-MD5) TCP socket, accepts new connections, and connects on another, TCP-MD5-enabled socket to the target machine.

You need [Go](https://golang.org) to compile this using `go build`.

## Usage

```
$ ./md5proxy -src 192.168.10.1:31337 -dst "[2001:19f0:ffff::1]:179" -md5 solarwinds123
```

## License and contact

GNU GPL v3

pjf@foremski.pl, 2021
