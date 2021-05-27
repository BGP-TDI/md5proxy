package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

var (
	optListen         = flag.String("src", "127.0.0.1:1790", "TCP address to listen on (use [] for IPv6)")
	optConnect        = flag.String("dst", "127.0.0.1:179", "TCP address to connect to (use [] for IPv6)")
	optPassword       = flag.String("md5", "", "TCP-MD5 password")
	optTimeout        = flag.Int("timeout", 10, "connection timeout (seconds)")
)

func md5dialer() *net.Dialer {
	var d net.Dialer

	// is addr ipv6?
	ipv6 := (*optConnect)[0] == '['

	// socket addr storage
	var storage unix.SockaddrStorage
	if ipv6 {
		storage.Family = unix.AF_INET6
	} else {
		storage.Family = unix.AF_INET
	}

	// tcp sig needed?
	if l := len(*optPassword); l > 0 {
		var key [80]byte
		copy(key[0:], []byte(*optPassword))

		sig := unix.TCPMD5Sig{
			Addr:      storage,
			Flags:     unix.TCP_MD5SIG_FLAG_PREFIX,
			Prefixlen: 0,
			Keylen:    uint16(l),
			Key:       key,
		}

		d.Control = func(network, address string, c syscall.RawConn) error {
			var err error
			c.Control(func(fd uintptr) {
				b := *(*[unsafe.Sizeof(sig)]byte)(unsafe.Pointer(&sig))
				err = unix.SetsockoptString(int(fd), unix.IPPROTO_TCP, unix.TCP_MD5SIG_EXT, string(b[:]))
			})
			return err
		}
	}

	return &d
}

func main() {
	flag.Parse()

	dialer := md5dialer()

	srvconn, err := net.Listen("tcp", *optListen)
	if err != nil { log.Fatal(err) }

	log.Printf("listening on %s, target %s", srvconn.Addr(), *optConnect)

	for {
		newconn, err := srvconn.Accept()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("accepted %s", newconn.RemoteAddr())

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		md5conn, err := dialer.DialContext(ctx, "tcp", *optConnect)
		if err != nil {
			log.Println(err)
			newconn.Close()
			cancel()
			continue
		}
		log.Printf("connected to %s", md5conn.RemoteAddr())

		// copy from client to BGP
		go func() {
			n, err := io.Copy(md5conn, newconn)
			switch {
			case err == nil, errors.Is(err, net.ErrClosed):
				log.Printf("sent %d bytes", n)
			default:
				log.Printf("sent %d bytes: %s", n, err)
			}

			newconn.Close()
			md5conn.Close()
		}()

		// copy from BGP to client
		n, err := io.Copy(newconn, md5conn)
		switch {
		case err == nil, errors.Is(err, net.ErrClosed):
			log.Printf("received %d bytes", n)
		default:
			log.Printf("received %d bytes: %s", n, err)
		}

		newconn.Close()
		md5conn.Close()
		cancel()
	}
}
