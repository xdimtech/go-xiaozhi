package openai

import (
	"context"
	"errors"
	"time"

	"github.com/xdimtech/go-xiaozhi/handler/base"

	"github.com/gorilla/websocket"
	xiaozhiapi "github.com/xdimtech/go-xiaozhi/pkg/protocol/xiaozhi"
	"golang.org/x/sync/errgroup"
)

const (
	WriteQueueSize = 1024
	PingTick       = 1 * time.Second
)

type ConnWrapper struct {
	ctx         context.Context
	conn        *websocket.Conn
	handler     base.WsHandler
	done        chan struct{}
	idleTimeout time.Duration
	idleTimer   *time.Timer
}

type WsConnOption func(*ConnWrapper)

func WithIdleTimeout(idleTimeout time.Duration) WsConnOption {
	return func(w *ConnWrapper) {
		w.idleTimeout = idleTimeout
		w.idleTimer = time.NewTimer(w.idleTimeout)
	}
}

func WithProxyHandler(handler base.WsHandler) WsConnOption {
	return func(w *ConnWrapper) {
		w.handler = handler
	}
}

func NewConnWrapper(ctx context.Context, conn *websocket.Conn, ops ...WsConnOption) (*ConnWrapper, error) {

	hdl, err := NewXiaozhiHandler(ctx, conn)
	if err != nil {
		return nil, err
	}

	wsConn := &ConnWrapper{
		ctx:         ctx,
		conn:        conn,
		handler:     hdl,
		done:        make(chan struct{}),
		idleTimeout: 0,
	}

	for _, op := range ops {
		op(wsConn)
	}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { wsConn.WriteLoop(ctx); return nil })
	g.Go(func() error { wsConn.WatchIdle(ctx); return nil })
	g.Go(func() error { wsConn.WatchHandlerDone(ctx); return nil })

	conn.SetPingHandler(wsConn.Pong)
	return wsConn, nil
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
	if err := w.handler.Close(w.ctx); err != nil {
		return err
	}
	if w.idleTimer != nil {
		w.idleTimer.Stop()
	}
	return nil
}

func (w *ConnWrapper) WriteLoop(ctx context.Context) {
	activeTimer := time.NewTimer(PingTick)
	for {
		select {
		case event, ok := <-w.handler.Recv(ctx):
			if !ok {
				continue
			}
			if w.conn == nil {
				return
			}
			if _, ok := xiaozhiapi.IsServerEvent(event); ok {
				writeBuf, err := w.handler.MarshalServerEvent(event)
				if err != nil {
					continue
				}
				_ = w.conn.WriteMessage(websocket.TextMessage, writeBuf)
			} else {
				binData, ok := event.([]byte)
				if !ok {
					continue
				}
				_ = w.conn.WriteMessage(websocket.BinaryMessage, binData)
			}
			w.resetIdleTimer()
		case <-w.done:
			return
		case <-activeTimer.C:
			activeTimer.Reset(PingTick)
			_ = w.Ping("done")
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

		var event any
		switch msgType {
		case websocket.TextMessage:
			event, err = w.handler.UnmarshalClientTextEvent(msg)
		case websocket.BinaryMessage:
			event, err = w.handler.UnmarshalClientBinEvent(msg)
		}

		if err != nil {
			errEvent := w.handler.BuildErrorEvent(ctx, errors.New("invalid event format"))
			_ = w.conn.WriteJSON(errEvent)
			continue
		}

		quit := false
		err, quit = w.handler.DispatchClientEvent(ctx, event)
		if err != nil {
			errEvent := w.handler.BuildErrorEvent(ctx, err)
			_ = w.conn.WriteJSON(errEvent)
		}

		if quit {
			w.done <- struct{}{}
			return err
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
	_ = w.conn.WriteJSON(w.handler.BuildErrorEvent(w.ctx, errors.New("too long without operation")))
	_ = w.conn.Close()
}

func (w *ConnWrapper) resetIdleTimer() {
	if w.idleTimeout == 0 {
		return
	}
	w.idleTimer.Reset(w.idleTimeout)
}

func (w *ConnWrapper) WatchHandlerDone(ctx context.Context) {
	<-w.handler.Done()
	if w.conn != nil {
		_ = w.conn.Close()
	}
}
