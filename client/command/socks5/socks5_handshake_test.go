package socks5

import (
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

// local pure SOCKS5 server used to validate our handshake encoding without implant.
func TestSocks5UserPassHandshakeAndConnectEncoding(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.Close()

		// greeting
		buf := make([]byte, 512)
		if _, err := io.ReadFull(conn, buf[:2]); err != nil {
			t.Errorf("read ver: %v", err)
			return
		}
		if buf[0] != 0x05 {
			t.Errorf("ver=%d", buf[0])
			return
		}
		n := int(buf[1])
		if _, err := io.ReadFull(conn, buf[:n]); err != nil {
			t.Errorf("methods: %v", err)
			return
		}
		_, _ = conn.Write([]byte{0x05, 0x02})

		// auth
		if _, err := io.ReadFull(conn, buf[:2]); err != nil {
			t.Errorf("auth hdr: %v", err)
			return
		}
		ulen := int(buf[1])
		if _, err := io.ReadFull(conn, buf[:ulen+1]); err != nil {
			t.Errorf("auth user: %v", err)
			return
		}
		user := string(buf[:ulen])
		plen := int(buf[ulen])
		if _, err := io.ReadFull(conn, buf[:plen]); err != nil {
			t.Errorf("auth pass: %v", err)
			return
		}
		pass := string(buf[:plen])
		if user != "admin" || pass != "secret" {
			t.Errorf("creds %q/%q", user, pass)
			_, _ = conn.Write([]byte{0x01, 0x01})
			return
		}
		_, _ = conn.Write([]byte{0x01, 0x00})

		// request
		if _, err := io.ReadFull(conn, buf[:4]); err != nil {
			t.Errorf("req: %v", err)
			return
		}
		if buf[0] != 0x05 || buf[1] != 0x01 || buf[3] != 0x03 {
			t.Errorf("bad req header %v", buf[:4])
			return
		}
		if _, err := io.ReadFull(conn, buf[:1]); err != nil {
			return
		}
		dlen := int(buf[0])
		if _, err := io.ReadFull(conn, buf[:dlen]); err != nil {
			return
		}
		host := string(buf[:dlen])
		if _, err := io.ReadFull(conn, buf[:2]); err != nil {
			return
		}
		port := binary.BigEndian.Uint16(buf[:2])
		if host != "example.invalid" || port != 80 {
			t.Errorf("host/port=%s:%d", host, port)
		}
		// success reply
		_, _ = conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		// echo one payload
		nread, _ := conn.Read(buf)
		_, _ = conn.Write(buf[:nread])
	}()

	// client side using std library style handshake
	c, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	// methods: user/pass only
	_, _ = c.Write([]byte{0x05, 0x01, 0x02})
	resp := make([]byte, 2)
	if _, err := io.ReadFull(c, resp); err != nil {
		t.Fatal(err)
	}
	if resp[0] != 0x05 || resp[1] != 0x02 {
		t.Fatalf("method select %v", resp)
	}
	// auth
	u, p := []byte("admin"), []byte("secret")
	msg := []byte{0x01, byte(len(u))}
	msg = append(msg, u...)
	msg = append(msg, byte(len(p)))
	msg = append(msg, p...)
	_, _ = c.Write(msg)
	if _, err := io.ReadFull(c, resp); err != nil {
		t.Fatal(err)
	}
	if resp[1] != 0x00 {
		t.Fatalf("auth failed %v", resp)
	}
	// connect domain example.invalid:80
	host := []byte("example.invalid")
	req := []byte{0x05, 0x01, 0x00, 0x03, byte(len(host))}
	req = append(req, host...)
	portb := make([]byte, 2)
	binary.BigEndian.PutUint16(portb, 80)
	req = append(req, portb...)
	_, _ = c.Write(req)
	reply := make([]byte, 10)
	if _, err := io.ReadFull(c, reply); err != nil {
		t.Fatal(err)
	}
	if reply[1] != 0x00 {
		t.Fatalf("connect rep=%d", reply[1])
	}
	_, _ = c.Write([]byte("ping"))
	out := make([]byte, 4)
	if _, err := io.ReadFull(c, out); err != nil {
		t.Fatal(err)
	}
	if string(out) != "ping" {
		t.Fatalf("echo=%q", out)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("server hang")
	}
}
