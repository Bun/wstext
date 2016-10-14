package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var (
	// TODO: support multiple backends on the same server
	path    = flag.String("path", "/ws", "WebSocket path (empty: any)")
	backend = flag.String("backend", "127.0.0.1:6600", "Backend address")

	static   = flag.String("static-dir", "", "Serve static files")
	bind     = flag.String("bind", ":13542", "Bind address")
	tls_key  = flag.String("key", "", "TLS key")
	tls_cert = flag.String("cert", "", "TLS certificate")
)

var upgrader = websocket.Upgrader{
	HandshakeTimeout: time.Second * 30,
	ReadBufferSize:   2048,
	WriteBufferSize:  2048,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ProxyHandler struct {
	fs http.Handler
}

func (p *ProxyHandler) wserror(w http.ResponseWriter, err error) {
	h := w.Header()
	h.Set("Content-Type", "text/plain; charset=utf8")
	w.WriteHeader(500)
	fmt.Fprintf(w, "Error: %s", err)
}

func (p *ProxyHandler) proxy(w http.ResponseWriter, r *http.Request) {
	// TODO: support Unix sockets
	b, err := net.DialTimeout("tcp", *backend, time.Second*5)
	if err != nil {
		p.wserror(w, err)
		return
	}
	defer b.Close()

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ws-upgrade:", err)
		p.wserror(w, err)
		return
	}
	defer c.Close()

	end := make(chan struct{}, 1)
	go func() {
		var buf [2048]byte
		var err error
		for {
			var i int
			i, err = b.Read(buf[:])
			if err != nil {
				break
			}
			err = c.WriteMessage(websocket.TextMessage, buf[:i])
		}
		if err != nil {
			log.Println("WS2B:", err)
		}
		end <- struct{}{}
	}()
	go func() {
		var err error
		for {
			mt, buf, err_ := c.ReadMessage()
			if err_ != nil {
				err = err_
				break
			} else if mt == websocket.CloseMessage {
				break
			} else if mt != websocket.TextMessage {
				log.Printf("WS2BK: Received non text message: %v", mt)
				continue
			}
			_, err = b.Write(buf)
			if err != nil {
				break
			}
		}
		if err != nil {
			log.Println("B2WS:", err)
		}
		end <- struct{}{}
	}()

	<-end
}

func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Serve:", r.URL.Path)

	if *path == "" || *path == r.URL.Path {
		p.proxy(w, r)
		return
	}

	if p.fs != nil {
		p.fs.ServeHTTP(w, r)
	} else {
		http.NotFound(w, r)
	}
}

func main() {
	flag.Parse()
	log.Println("wstext")

	var fs http.Handler
	if *static != "" {
		log.Println("Static files path:", *static)
		fs = http.FileServer(http.Dir(*static))
	}

	ws := &http.Server{
		Addr:           *bind,
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 16,
		Handler:        &ProxyHandler{fs},
	}

	var err error
	if *tls_key != "" || *tls_cert != "" {
		err = ws.ListenAndServeTLS(*tls_cert, *tls_key)
	} else {
		err = ws.ListenAndServe()
	}
	if err != nil {
		log.Fatalln("Server error:", err)
	}
}
