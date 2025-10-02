package rtmp_server

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/nareix/joy4/av/avutil"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/nareix/joy4/format/rtmp"
	"github.com/nareix/joy4/format/ts"
)

type Channel struct {
	que *pubsub.Queue
}

func InitRtmpServer() (*rtmp.Server, *sync.RWMutex, map[string]*Channel) {
	server := &rtmp.Server{}

	l := &sync.RWMutex{}
	channels := map[string]*Channel{}

	server.HandlePlay = func(conn *rtmp.Conn) {
		l.RLock()

		parts := strings.Split(conn.URL.Path, "/")

		dst_name := "/" + parts[1]
		if len(parts) == 3 {
			dst_name += "/" + parts[2]
		}
		fmt.Printf("New RTMP stream access at %s\n", dst_name)

		ch := channels[dst_name]
		l.RUnlock()

		if ch != nil {
			cursor := ch.que.Latest()
			avutil.CopyFile(conn, cursor)
		}
	}

	server.HandlePublish = func(conn *rtmp.Conn) {
		streams, _ := conn.Streams()

		l.Lock()

		parts := strings.Split(conn.URL.Path, "/")

		dst_name := "/" + parts[1]
		if len(parts) == 3 {
			dst_name += "/" + parts[2]
		}
		fmt.Printf("New RTMP stream access at %s\n", dst_name)

		ch := channels[dst_name]

		if ch == nil {
			ch = &Channel{}
			ch.que = pubsub.NewQueue()
			ch.que.WriteHeader(streams)
			channels[conn.URL.Path] = ch
		} else {
			ch = nil
		}
		l.Unlock()
		if ch == nil {
			return
		}

		avutil.CopyPackets(ch.que, conn)

		l.Lock()
		delete(channels, conn.URL.Path)
		l.Unlock()
		ch.que.Close()
	}

	return server, l, channels
}

type writeFlusher struct {
	httpflusher http.Flusher
	io.Writer
}

func (self writeFlusher) Flush() error {
	self.httpflusher.Flush()
	return nil
}

func InitHttpRoutes(http_mux *http.ServeMux, l *sync.RWMutex, channels map[string]*Channel) {
	http_mux.HandleFunc("/stream/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
		parts := strings.Split(r.URL.Path, "/")

		dst_name := "/" + parts[2]
		if len(parts) == 4 {
			dst_name += "/" + parts[3]
		}
		fmt.Printf("New HTTP TS stream access at %s\n", dst_name)
		l.RLock()
		ch := channels[dst_name]
		l.RUnlock()

		if ch != nil {
			w.Header().Set("Content-Type", "video/MP2T")
			w.Header().Set("Transfer-Encoding", "chunked")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(200)
			flusher := w.(http.Flusher)
			flusher.Flush()

			ts_muxer := ts.NewMuxer(writeFlusher{httpflusher: flusher, Writer: w})
			cursor := ch.que.Latest()

			avutil.CopyFile(ts_muxer, cursor)
		} else {
			http.NotFound(w, r)
		}
	})
}
