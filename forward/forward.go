package forward

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	buffers   = sync.Pool{New: func() interface{} { return make([]byte, 1<<16) }}
	listeners = make(map[string]net.Listener)
)

func Close(bind string) error {
	l, ok := listeners[bind]
	if !ok {
		return fmt.Errorf("Not listening on %q", bind)
	}
	return l.Close()
}

func ListenAndServe(bind string) error {
	l, err := net.Listen("tcp", bind)
	if err != nil {
		return fmt.Errorf("Failed to listen on %q: %v", bind, err)
	}
	listeners[bind] = l
	log.Printf("forward: listening on %s\n", l.Addr())
	go listenAndServe(l.(*net.TCPListener))
	return nil
}

func listenAndServe(l *net.TCPListener) {
	defer l.Close()
	var delay time.Duration
	for {
		c, err := l.AcceptTCP()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if delay == 0 {
					delay = 5 * time.Millisecond
				} else if delay *= 2; delay > time.Second {
					delay = time.Second
				}
				log.Printf("forward: %v; retrying in %v", err, delay)
				time.Sleep(delay)
				continue
			}
			log.Printf("forward: stopped listening on %s: %v\n", l.Addr().String(), err)
			return
		}

		delay = 0
		c.SetKeepAlive(true)
		c.SetKeepAlivePeriod(3 * time.Minute)
		go serve(c)
	}
}

func serve(c *net.TCPConn) {
	cli := &client{
		TCPConn: c,
		Reader:  bufio.NewReader(c),
		buf:     new(bytes.Buffer),
	}
	cli.log = log.New(io.MultiWriter(os.Stdout, cli.buf), "", 0)

	defer func() {
		if err := recover(); err != nil {
			cli.log.Printf("panic serving\n%s", string(runtime.ReadTrace()))
		}
	}()

	cli.log.Printf("accepted %v\n", cli.RemoteAddr())
	defer cli.log.Printf("disconnected %v\n", cli.RemoteAddr())

	if err := cli.serve(); err != nil {
		cli.log.Printf("%v\n", err)
	}
}

type client struct {
	*net.TCPConn
	*bufio.Reader

	log *log.Logger
	buf *bytes.Buffer
}

func (cli *client) Read(p []byte) (n int, err error) { return cli.Reader.Read(p) }

func (cli *client) serve() error {
	var (
		requestLine []byte
		err         error
	)
	for { // read request-line; RFC-2616#4.1 discard prefixed empty lines.
		requestLine, err = cli.ReadBytes('\n')
		if err != nil {
			return fmt.Errorf("failed to read request target: %v", err)
		}
		if len(bytes.TrimSpace(requestLine)) != 0 {
			break
		}
	}

	p := strings.Split(string(requestLine), " ")
	if len(p) < 2 {
		return fmt.Errorf("invalid request-line: %q", string(requestLine))
	}
	method, target := p[0], p[1]

	addr, err := hostport(target)
	if err != nil {
		return fmt.Errorf("get host:port of %q failed with request-line %q: %v", target, string(requestLine), err)
	}

	srv, err := dialTCP(addr)
	if err != nil {
		return fmt.Errorf("dial target host %q failed with request-line %q: %v", addr, string(requestLine), err)
	}
	cli.log.Println(string(bytes.TrimSpace(requestLine)))

	switch method {
	case "CONNECT": // RFC-7231#4.3.6
		for { // line request up to next response
			line, err := cli.ReadBytes('\n')
			if err != nil {
				return fmt.Errorf("failed to line up next client request: %v", err)
			}
			if len(bytes.TrimSpace(line)) == 0 {
				break
			}
		}
		if _, err := fmt.Fprint(cli, "HTTP/1.1 200 OK\r\n\r\n"); err != nil {
			return fmt.Errorf("failed to deliver 200 response to client CONNECT request: %v", err)
		}
	default:
		if _, err := srv.Write(requestLine); err != nil {
			return fmt.Errorf("err writing request-line: %v", err)
		}
	}

	srvClosed := make(chan struct{})
	cliClosed := make(chan struct{})

	go copyBuffer(srv, cli, cliClosed, cli.log)
	go copyBuffer(cli, srv, srvClosed, cli.log)

	select {
	case <-cliClosed:
		srv.SetLinger(0) // client closed first; recycle port.
		srv.CloseRead()
		<-srvClosed
	case <-srvClosed:
		cli.CloseRead()
		<-cliClosed
	}

	return nil
}

func copyBuffer(dst io.Writer, src io.ReadCloser, srcClosed chan struct{}, logger *log.Logger) {
	buf := buffers.Get().([]byte)
	defer buffers.Put(buf)
	if _, err := io.CopyBuffer(dst, src, buf); err != nil {
		logger.Printf("copyBuffer: %v\n", err)
	}
	if err := src.Close(); err != nil {
		logger.Printf("close source: %v\n", err)
	}
	close(srcClosed)
}

// hostport returns authority-form of target; RFC-7230#5.3
func hostport(target string) (string, error) {
	if len(target) == 0 {
		return "", errors.New("empty string argument")
	}
	switch target[0] {
	case '/':
		return "", errors.New("origin-form not yet supported")
	case '*':
		return "", errors.New("asterisk-form not yet supported")
	}
	if strings.Index(target, "://") == -1 {
		return target, nil // already in authority-form, use as-is.
	}

	u, err := url.Parse(target)
	if err != nil {
		return "", fmt.Errorf("absolute-form parse error: %v", err)
	}
	port := u.Port()
	if port == "" {
		port = "80"
	}
	return fmt.Sprintf("%s:%s", u.Hostname(), port), nil
}

func dialTCP(addr string) (*net.TCPConn, error) {
	d := net.Dialer{Timeout: 10 * time.Second, KeepAlive: 3 * time.Minute}
	conn, err := d.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return conn.(*net.TCPConn), nil
}
