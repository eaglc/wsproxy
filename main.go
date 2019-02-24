package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"wsproxy/proxy"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

var gAddr = flag.String("addr", "", "Listen address.")
var gModuleId = flag.Int("module_id", 0, "ModuleId for wsproxys.")

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func eaglc(w http.ResponseWriter, r *http.Request) {

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		glog.Warningf("upgrade:%v", err)
		return
	}
	defer c.Close()
	r.ParseForm()
	host := r.FormValue("host")
	port := r.FormValue("port")

	if host == "" || port == "" {
		glog.Warningf("host or port invalid. host: %s port: %s", host, port)
		return
	}

	glog.Infof("request with backend host: %v port: %v", host, port)

	cn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", host, port))

	if err != nil {
		glog.Errorf("connect to backend failed. err: %v host: %s port:%s ", err, host, port)
		return
	}
	defer cn.Close()

	proxy := proxy.NewWsProxy(cn, *c)
	proxy.Process()

	glog.Infof("end with backend host: %v port: %v", host, port)
	glog.Flush()
}

// -log_dir=./log_dir -module_id=0
func main() {
	flag.Parse()

	defer glog.Flush()

	if *gAddr == "" {
		glog.Fatalf("listen addr can not empty.")
		return
	}

	glog.Infof("start with addr: %s module_id: %d", *gAddr, *gModuleId)

	http.HandleFunc("/ws", eaglc)
	http.ListenAndServe(*gAddr, nil)
	//log.Fatal(http.ListenAndServeTLS(*addr, "server.crt", "server.key", nil))
}
