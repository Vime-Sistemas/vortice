package main

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewProxyPool_Routes(t *testing.T) {
	// start two simple backend servers
	b1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("one"))
	}))
	defer b1.Close()
	b2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("two"))
	}))
	defer b2.Close()

	pool := NewProxyPool([]string{b1.URL, b2.URL}, "round_robin", [][2]int{{0, 1}, {0, 1}}, "")

	// create listener on ephemeral port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()

	server := &http.Server{Handler: pool}
	go server.Serve(ln)
	defer server.Close()

	// give server a moment
	time.Sleep(50 * time.Millisecond)

	addr := ln.Addr().String()
	resp, err := http.Get("http://" + addr + "/")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "one" && string(body) != "two" {
		t.Fatalf("unexpected body: %s", string(body))
	}
}
