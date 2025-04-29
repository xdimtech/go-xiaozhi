package xiaozhi

import (
	"encoding/json"
	"fmt"
)

// ClientEventType 客户端事件
type ClientEventType string

const (
	ClientEventTypeHello        ClientEventType = "hello"
	ClientEventTypeListen       ClientEventType = "listen"
	ClientEventTypeAppendBuffer ClientEventType = "append.buffer"
	ClientEventTypeAbort        ClientEventType = "abort"
	ClientEventTypeIot          ClientEventType = "iot"
)

// ClientState 客户端监听状态
type ClientState string

const (
	ClientStateIdle         ClientState = "idle"
	ClientStateListenStart  ClientState = "start"
	ClientStateListenStop   ClientState = "stop"
	ClientStateListenDetect ClientState = "detect" // 用于客户端向服务器告知检测到唤醒词
)

// ClientMode 客户端识别模式
type ClientMode string

const (
	ClientModeAuto      ClientMode = "auto"
	ClientModeManual    ClientMode = "manual"
	ClientModelRealtime ClientMode = "realtime"
)

type ClientEvent interface {
	ClientEventType() ClientEventType
	GetAudioParams() *AudioParams
	GetSessionID() string
}

type AudioParams struct {
	Format        string `json:"format"`
	SampleRate    int    `json:"sample_rate"`
	Channels      int    `json:"channels"`
	FrameDuration int    `json:"frame_duration"`
}

// ClientEventBase is the base struct for all client events.
type ClientEventBase struct {
	Type        ClientEventType `json:"type"`
	Version     int             `json:"version,omitempty"`
	Transport   string          `json:"transport,omitempty"` // websocket, rtc, iot
	SessionID   string          `json:"session_id,omitempty"`
	AudioParams *AudioParams    `json:"audio_params,omitempty"`
}

// ClientEventHello is the hello event.
type ClientEventHello struct {
	ClientEventBase
}

// {'type': 'hello', 'version': 1, 'transport': 'websocket', 'audio_params': {'format': 'opus', 'sample_rate': 16000, 'channels': 1, 'frame_duration': 20}}

// ClientEventListen is the listen event.
type ClientEventListen struct {
	ClientEventBase
	State ClientState `json:"state"`
	Mode  ClientMode  `json:"mode"`
}

type ClientEventAppendBuffer struct {
	ClientEventBase
	Bytes []byte `json:"bytes"` // opus编码的二进制数据
}

// {'type': 'listen','state': 'start','mode': 'auto'}   然后客户端开始发送二进制的音频数据
// {'type': 'listen','state':'stop'}   客户端停止发送二进制的音频数据
// {'type': 'listen','state':'detect'}   客户端检测到唤醒词，发送二进制的音频数据

// ClientEventAbort is the abort event.
type ClientEventAbort struct {
	ClientEventBase
	Reason string `json:"reason"`
}

type ClientEventIot struct {
	ClientEventBase
	Data string `json:"data"`
}

func (e *ClientEventBase) ClientEventType() ClientEventType {
	return e.Type
}
func (e *ClientEventBase) GetAudioParams() *AudioParams {
	return e.AudioParams
}

func (e *ClientEventBase) GetSessionID() string {
	return e.SessionID
}

func (e *ClientEventHello) ClientEventType() ClientEventType {
	return ClientEventTypeHello
}

func (e *ClientEventHello) GetAudioParams() *AudioParams {
	return e.AudioParams
}

func (e *ClientEventAppendBuffer) ClientEventType() ClientEventType {
	return ClientEventTypeAppendBuffer
}

func (e *ClientEventAppendBuffer) GetAudioParams() *AudioParams {
	return e.AudioParams
}
func (e *ClientEventHello) GetSessionID() string {
	return e.SessionID
}

func (e *ClientEventListen) ClientEventType() ClientEventType {
	return ClientEventTypeListen
}

func (e *ClientEventListen) GetAudioParams() *AudioParams {
	return e.AudioParams
}

func (e *ClientEventListen) GetSessionID() string {
	return e.SessionID
}

func (e *ClientEventAbort) ClientEventType() ClientEventType {
	return ClientEventTypeAbort
}

func (e *ClientEventAbort) GetAudioParams() *AudioParams {
	return e.AudioParams
}

func (e *ClientEventAbort) GetSessionID() string {
	return e.SessionID
}

func (e *ClientEventIot) ClientEventType() ClientEventType {
	return ClientEventTypeIot
}

func (e *ClientEventIot) GetAudioParams() *AudioParams {
	return e.AudioParams
}

func (e *ClientEventIot) GetSessionID() string {
	return e.SessionID
}

type ClientEventInterface interface {
	ClientEventHello | ClientEventListen | ClientEventAbort | ClientEventIot
}

func unmarshalClientEvent[T ClientEventInterface](data []byte) (*T, error) {
	var t T
	err := json.Unmarshal(data, &t)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// MarshalClientEvent marshals the client event to JSON.
func MarshalClientEvent(event ClientEvent) ([]byte, error) {
	return json.Marshal(event)
}

// UnmarshalClientEvent Unmarshal the server event from the given JSON data.
func UnmarshalClientEvent(data []byte) (ClientEvent, error) {
	var eventType struct {
		Type ClientEventType `json:"type"`
	}
	err := json.Unmarshal(data, &eventType)
	if err != nil {
		return nil, err
	}
	switch eventType.Type {
	case ClientEventTypeHello:
		return unmarshalClientEvent[ClientEventHello](data)
	case ClientEventTypeListen:
		return unmarshalClientEvent[ClientEventListen](data)
	case ClientEventTypeAbort:
		return unmarshalClientEvent[ClientEventAbort](data)
	case ClientEventTypeIot:
		return unmarshalClientEvent[ClientEventIot](data)
	default:
		return nil, fmt.Errorf("unknown client event type: %s", eventType.Type)
	}
}

func UnmarshalClientBinEvent(data []byte) (ClientEvent, error) {
	return &ClientEventAppendBuffer{
		ClientEventBase: ClientEventBase{
			Type: ClientEventTypeAppendBuffer,
		},
		Bytes: data,
	}, nil
}
