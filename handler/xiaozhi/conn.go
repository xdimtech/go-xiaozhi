package xiaozhi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/xdimtech/go-xiaozhi/pkg/config"

	"github.com/gorilla/websocket"
	xiaozhiapi "github.com/xdimtech/go-xiaozhi/pkg/protocol/xiaozhi"
	"golang.org/x/sync/errgroup"
)

type ConnWrapper struct {
	ctx         context.Context
	conn        *websocket.Conn
	proxyConn   *websocket.Conn
	done        chan struct{}
	idleTimeout time.Duration
	idleTimer   *time.Timer
	originReq   *http.Request
}

type WsConnOption func(*ConnWrapper)

func WithIdleTimeout(idleTimeout time.Duration) WsConnOption {
	return func(w *ConnWrapper) {
		w.idleTimeout = idleTimeout
		w.idleTimer = time.NewTimer(w.idleTimeout)
	}
}

func WithOriginReq(originReq *http.Request) WsConnOption {
	return func(w *ConnWrapper) {
		w.originReq = originReq
	}
}

func NewConnWrapper(ctx context.Context, conn *websocket.Conn, ops ...WsConnOption) (*ConnWrapper, error) {
	wsConn := &ConnWrapper{
		ctx:         ctx,
		conn:        conn,
		done:        make(chan struct{}),
		idleTimeout: 0,
	}
	for _, op := range ops {
		op(wsConn)
	}

	if wsConn.originReq == nil {
		return nil, errors.New("origin request is nil")
	}

	if err := wsConn.ConnectProxy(); err != nil {
		return nil, err
	}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { wsConn.WriteLoop(ctx); return nil })
	g.Go(func() error { wsConn.WatchIdle(ctx); return nil })

	conn.SetPingHandler(wsConn.Pong)
	return wsConn, nil
}

func (w *ConnWrapper) ConnectProxy() error {
	baseUrl := w.originReq.URL.String()
	if !strings.Contains(w.originReq.URL.String(), "wss://") {
		baseUrl = config.Provider().Xiaozhi.BaseURL
	}

	headers := http.Header{}
	headers.Add("Authorization", w.originReq.Header.Get("Authorization"))
	headers.Add("Protocol-Version", w.originReq.Header.Get("Protocol-Version"))
	headers.Add("Device-Id", w.originReq.Header.Get("Device-Id"))
	headers.Add("Client-Id", w.originReq.Header.Get("Client-Id"))
	proxyConn, resp, err := websocket.DefaultDialer.Dial(baseUrl, headers)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		return errors.New("invalid request")
	}
	w.proxyConn = proxyConn
	return nil
}

func (w *ConnWrapper) Ping(data string) error {
	if w.conn == nil {
		return nil
	}
	w.resetIdleTimer()
	return w.conn.WriteControl(websocket.PingMessage, []byte(data), time.Now().Add(time.Second))
}

func (w *ConnWrapper) Pong(data string) error {
	if w.conn == nil {
		return nil
	}
	w.resetIdleTimer()
	return w.conn.WriteControl(websocket.PongMessage, []byte(data), time.Now().Add(time.Second))
}

func (w *ConnWrapper) Close() error {
	_ = w.conn.Close()
	if w.idleTimer != nil {
		w.idleTimer.Stop()
	}
	return nil
}

func (w *ConnWrapper) WriteLoop(ctx context.Context) {
	for {
		msgType, msg, merr := w.proxyConn.ReadMessage()
		if merr != nil {
			w.done <- struct{}{}
			break
		}

		var err error
		switch msgType {
		case websocket.TextMessage:
			err = w.conn.WriteMessage(websocket.TextMessage, msg)
		case websocket.BinaryMessage:
			err = w.conn.WriteMessage(websocket.BinaryMessage, msg)
		}
		if err != nil {
			errEvent := w.errorEvent(ctx, errors.New("invalid event format"))
			_ = w.conn.WriteJSON(errEvent)
			continue
		}
	}
}

func (w *ConnWrapper) ReadLoop(ctx context.Context) (err error) {

	defer func() {
		_ = w.Close()
	}()

	for {
		msgType, msg, merr := w.conn.ReadMessage()
		if merr != nil {
			w.done <- struct{}{}
			break
		}

		switch msgType {
		case websocket.TextMessage:
			err = w.proxyConn.WriteMessage(websocket.TextMessage, msg)
		case websocket.BinaryMessage:
			err = w.proxyConn.WriteMessage(websocket.BinaryMessage, msg)
		}
		if err != nil {
			errEvent := w.errorEvent(ctx, errors.New("invalid event format"))
			_ = w.conn.WriteJSON(errEvent)
			continue
		}
		w.resetIdleTimer()
	}
	return err
}

func (w *ConnWrapper) WatchIdle(ctx context.Context) {
	if w.idleTimeout == 0 {
		return
	}
	if w.idleTimer == nil {
		return
	}

	<-w.idleTimer.C
	_ = w.conn.WriteJSON(w.errorEvent(w.ctx, errors.New("too long without operation")))
	_ = w.conn.Close()
}

func (w *ConnWrapper) resetIdleTimer() {
	if w.idleTimeout == 0 {
		return
	}
	w.idleTimer.Reset(w.idleTimeout)
}

func (w *ConnWrapper) errorEvent(ctx context.Context, err error) xiaozhiapi.ServerEvent {
	return &xiaozhiapi.ServerEventError{
		ServerEventBase: xiaozhiapi.ServerEventBase{
			Type: xiaozhiapi.ServerEventTypeError,
		},
		Error: err.Error(),
	}
}
