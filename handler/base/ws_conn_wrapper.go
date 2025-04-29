package base

import "context"

type WsConnWrapper interface {
	Ping(data string) error
	Pong(data string) error
	Close() error
	WriteLoop(ctx context.Context)
	ReadLoop(ctx context.Context) error
	WatchIdle(ctx context.Context)
}
