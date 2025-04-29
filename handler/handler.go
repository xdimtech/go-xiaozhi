package handler

import (
	"context"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/xdimtech/go-xiaozhi/handler/base"
	"github.com/xdimtech/go-xiaozhi/handler/openai"
	"github.com/xdimtech/go-xiaozhi/handler/xiaozhi"
	"github.com/xdimtech/go-xiaozhi/pkg/config"

	"github.com/gorilla/websocket"
	"github.com/xdimtech/go-xiaozhi/pkg/utils"
)

type WebSocketServer struct {
	requestCounter atomic.Int64
}

func NewWebSocketServer() *WebSocketServer {
	return &WebSocketServer{}
}

func (s *WebSocketServer) Start(addr string) error {
	http.HandleFunc("/xiaozhi/v1/", s.RealTime)
	log.Printf("Server started at local: ws://127.0.0.1%s\n", addr)
	ip, _ := utils.GetLocalIP()
	log.Printf("Server started at public: ws://%s%s\n", ip, addr)
	return http.ListenAndServe(addr, nil)
}

func (s *WebSocketServer) wsConnect(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	upgrader := &websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			return true
			// 开发环境下允许所有来源
			// 生产环境建议使用更严格的检查:
			// origin := r.Header.Get("Origin")
			// return origin == "http://localhost:8080" ||
			//        origin == "http://your-allowed-domain.com"
		},
	}
	conn, err := upgrader.Upgrade(w, r, w.Header())
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (s *WebSocketServer) RealTime(w http.ResponseWriter, r *http.Request) {

	var err error
	conn, err := s.wsConnect(w, r)
	if err != nil {
		panic(err)
	}

	defer func() {
		_ = conn.Close()
		conn = nil
	}()

	ctx := r.Context()
	connWrapper, err := s.NewConnWrapper(ctx, conn, r)
	if err != nil {
		panic(err)
	}

	_ = connWrapper.ReadLoop(ctx)
}

func (s *WebSocketServer) NewConnWrapper(
	ctx context.Context, conn *websocket.Conn, r *http.Request) (base.WsConnWrapper, error) {
	if config.Provider().Name == "openai" {
		return openai.NewConnWrapper(ctx, conn)
	}
	return xiaozhi.NewConnWrapper(ctx, conn, xiaozhi.WithOriginReq(r))
}
