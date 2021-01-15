package util

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func ServeFiles(dir, recommendedPort string, silentMode bool, goldsVersion string) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		log.Fatal(err)
	}

	port := recommendedPort
	defaultPort, err := strconv.Atoi(recommendedPort)
	if err != nil {
		if recommendedPort != "" {
			log.Printf("Invalid port: %s. A new one will be selected automatically.", recommendedPort)
		}
		defaultPort = 9999
		port = strconv.Itoa(defaultPort)
	}

	if defaultPort > 65535 {
		defaultPort = 65535
	} else if defaultPort < 1024 {
		defaultPort = 1024
	}

NextTry:
	l, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		if strings.Index(err.Error(), "bind: address already in use") >= 0 {
			if defaultPort == 1024 {
				defaultPort = 65536
			}
			defaultPort--
			port = strconv.Itoa(defaultPort)
			//port = strconv.Itoa(50000 + 1 + rand.Int()%9999)
			goto NextTry
		}
		// ToDo: random port
		log.Fatal(err)
	}

	// http://stackoverflow.com/questions/33880343/go-webserver-dont-cache-files-using-timestamp
	var epoch = time.Unix(0, 0).Format(time.RFC1123)
	var noCacheHeaders = map[string]string{
		"Expires":         epoch,
		"Cache-Control":   "no-cache, private, max-age=0",
		"Pragma":          "no-cache",
		"X-Accel-Expires": "0",
	}
	var etagHeaders = []string{
		"ETag",
		"If-Modified-Since",
		"If-Match",
		"If-None-Match",
		"If-Range",
		"If-Unmodified-Since",
	}

	NoCacheHandler := func(h http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			for _, v := range etagHeaders {
				if r.Header.Get(v) != "" {
					r.Header.Del(v)
				}
			}
			for k, v := range noCacheHeaders {
				w.Header().Set(k, v)
			}
			h.ServeHTTP(w, r)
		}
	}

	go func() {
		time.Sleep(time.Second)

		log.Println("Serving directory:")
		log.Print("   ", dir)
		log.Println("Running at:")
		log.Print("   http://localhost:", port)

		// ToDo: show the list in every html page.
		if addrs, err := net.InterfaceAddrs(); err == nil {
			for _, a := range addrs {
				if ipnet, ok := a.(*net.IPNet); ok && ipnet.IP.To4() != nil {
					log.Println("   http://" + ipnet.IP.String() + ":" + port)
				}
			}
		}

		if !silentMode {
			if err = OpenBrowser("http://localhost:" + port); err != nil {
				log.Println(err)
			}
		}
	}()

	handler := NoCacheHandler(http.FileServer(http.Dir(dir)))
	if err = http.Serve(l, handler); err != nil {
		log.Printf("Failed to start server: %v\n", err)
	}
}
