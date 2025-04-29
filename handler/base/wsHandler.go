package base

import "context"

type WsHandler interface {
	InitProxy(ctx context.Context) error
	Recv(ctx context.Context) chan any
	Close(ctx context.Context) error
	Done() <-chan struct{}
	UnmarshalClientTextEvent(msg []byte) (any, error)
	UnmarshalClientBinEvent(msg []byte) (any, error)
	DispatchClientEvent(ctx context.Context, event any) (error, bool)
	MarshalServerEvent(event any) ([]byte, error)
	BuildErrorEvent(ctx context.Context, err error) interface{}
}
