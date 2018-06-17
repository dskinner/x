package main

import (
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/mobile/event/paint"
)

var work = make(chan []string, 100)

func tryListenAndServe() error {
	hostname := ":6177"
	srv, err := net.Listen("tcp", hostname)
	if err != nil {
		return err
	}
	srv.Close()

	go func() {
		for args := range work {

			filter := args[:0]
			for _, arg := range args {
				if !walked[arg] {
					filter = append(filter, arg)
				}
			}
			if len(filter) == 0 {
				continue
			}

			gallery.view.transform = gallery.Model.Animator().At()
			_ = store.Cycle(gallery.view, 0) // save our current gallery.view before calling Walk!

			// i := len(store.views)
			i := store.r.Index()
			if err := store.Walk(i, filter...); err != nil {
				log.Println(err)
				continue
			}

			gallery.view = ZV
			// i := store.r.Index()
			// n := len(store.views)
			// stride := (n - 1) - i
			// glwidget.Cycle(stride)
			// glwidget.Cycle(0)
			Cycle(0)
			deque.Send(paint.Event{})
		}
	}()

	go func() {
		var pending []string
		var locker sync.Mutex
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			r.ParseForm()
			args := r.Form["args"]

			locker.Lock()
			defer locker.Unlock()

			pending = append(pending, args...)
			n := len(pending)

			go func(size int) {
				time.Sleep(time.Second) // wait for more
				locker.Lock()
				defer locker.Unlock()
				if len(pending) == size { // nothings changed; submit work
					p := make([]string, size)
					copy(p, pending)
					work <- p
					pending = pending[:0]
				}
			}(n)

			w.Write([]byte("OK"))
		})
		log.Fatal(http.ListenAndServe(hostname, nil))
	}()

	return nil
}
