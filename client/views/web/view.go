// interactive web user interface
package web

import (
	"cngrok/client/mvc"
	"cngrok/log"
	"cngrok/proto"
	"cngrok/util"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"net/http"
	"path"
)

type WebView struct {
	log.Logger

	ctl mvc.Controller

	// messages sent over this broadcast are sent to all websocket connections
	wsMessages *util.Broadcast
}

func NewWebView(ctl mvc.Controller, addr string) *WebView {
	wv := &WebView{
		Logger:     log.NewPrefixLogger("view", "web"),
		wsMessages: util.NewBroadcast(),
		ctl:        ctl,
	}

	// for now, always redirect to the http view
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/http/in", 302)
	})

	// handle web socket connections
	http.HandleFunc("/_ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)

		if err != nil {
			http.Error(w, "WebSocket升级失败", 400)
			wv.Warn("WebSocket升级失败: %v", err)
			return
		}

		msgs := wv.wsMessages.Reg()
		defer wv.wsMessages.UnReg(msgs)
		for m := range msgs {
			err := conn.WriteMessage(websocket.TextMessage, m.([]byte))
			if err != nil {
				// connection is closed
				break
			}
		}
	})

	// serve static assets
	http.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(path.Join(r.URL.Path[1:]))
		if err != nil {
			wv.Warn("服务静态文件的错误: %s", err.Error())
			http.NotFound(w, r)
			return
		}
		w.Write(buf)
	})

	wv.Info("Web服务接口 %s", addr)
	wv.ctl.Go(func() { http.ListenAndServe(addr, nil) })
	return wv
}

func (wv *WebView) NewHttpView(proto *proto.Http) *WebHttpView {
	return newWebHttpView(wv.ctl, wv, proto)
}

func (wv *WebView) Shutdown() {
}
