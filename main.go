package main

import (
	"fmt"
	"net/http"

	"rtmp/danmaku_server"
	"rtmp/rtmp_server"
	"rtmp/webui_server"

	"github.com/nareix/joy4/format"
)

func init() {
	format.RegisterAll()
}

func main() {
	http_mux := http.NewServeMux()
	server, l, channels := rtmp_server.InitRtmpServer()
	rtmp_server.InitHttpRoutes(http_mux, l, channels)
	danmaku_server.InitHttpRoutes(http_mux)
	go danmaku_server.RxAndBroadcast()
	webui_server.InitHttpRoutes(http_mux)

	go http.ListenAndServe(":8089", http_mux)
	server.ListenAndServe()
	fmt.Println("HTTP server started on port 8089")
	fmt.Println("RTMP server started on port 1935")
}
