package xiaozhi

import (
	"encoding/json"
)

type ServerEventType string

const (
	ServerEventTypeError ServerEventType = "error"
	ServerEventTypeHello ServerEventType = "hello"
	ServerEventTypeSTT   ServerEventType = "stt"
	ServerEventTypeLLM   ServerEventType = "llm"
	ServerEventTypeTTS   ServerEventType = "tts"
	ServerEventTypeIot   ServerEventType = "iot"
)

type ServerTTSState string

const (
	ServerTTSStateStart         ServerTTSState = "start"
	ServerTTSStateStop          ServerTTSState = "stop"
	ServerTTSStateSentenceStart ServerTTSState = "sentence_start"
	ServerTTSStateSentenceEnd   ServerTTSState = "sentence_end"
)

type ServerEvent interface {
	GetType() ServerEventType
}

type ServerEventBase struct {
	Type      ServerEventType `json:"type"`
	SessionId string          `json:"session_id"`
}

type ServerEventError struct {
	ServerEventBase
	Error string `json:"error"`
}

func (e *ServerEventError) GetType() ServerEventType {
	return ServerEventTypeError
}

type ServerEventHello struct {
	ServerEventBase
	Transport   string      `json:"transport"`
	AudioParams AudioParams `json:"audio_params"`
}

func (e *ServerEventHello) GetType() ServerEventType {
	return ServerEventTypeHello
}

// {'type': 'hello', 'version': 1, 'transport': 'websocket', 'audio_params': {'format': 'opus', 'sample_rate': 24000, 'channels': 1, 'frame_duration': 20}, 'session_id': '9842a257'}

type ServerEventSTT struct {
	ServerEventBase
	Text string `json:"text"`
}

func (e *ServerEventSTT) GetType() ServerEventType {
	return ServerEventTypeSTT
}

// {'type': 'stt', 'text': 'are youOK。', 'session_id': '9842a257'}

type ServerEventLLM struct {
	ServerEventBase
	Text    string `json:"text"`
	Emotion string `json:"emotion"`
}

func (e *ServerEventLLM) GetType() ServerEventType {
	return ServerEventTypeLLM
}

// {'type': 'llm', 'text': '😊', 'emotion': 'happy', 'session_id': '9842a257'}

type ServerEventTTS struct {
	ServerEventBase
	State      ServerTTSState `json:"state"`
	Text       string         `json:"text"`
	SampleRate int            `json:"sample_rate"`
}

func (e *ServerEventTTS) GetType() ServerEventType {
	return ServerEventTypeTTS
}

// {'type': 'tts', 'state': 'start', 'sample_rate': 24000, 'session_id': '9842a257'}
// {'type': 'tts', 'state': 'stop', 'session_id': '9842a257'}
// {'type': 'tts', 'state': 'sentence_start', 'text': '有什么好玩的事吗？', 'session_id': '9842a257'}
// {'type': 'tts', 'state': 'sentence_end', 'text': '有什么好玩的事吗？', 'session_id': '9842a257'}

type ServerEventInterface interface {
	ServerEventHello | ServerEventSTT | ServerEventLLM | ServerEventTTS
}

func unmarshalServerEvent[T ServerEventInterface](data []byte) (*T, error) {
	var t T
	err := json.Unmarshal(data, &t)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func IsServerEvent(event any) (ServerEvent, bool) {
	ev, ok := event.(ServerEvent)
	if !ok {
		return nil, false
	}
	switch ev.GetType() {
	case ServerEventTypeError:
		return event.(*ServerEventError), true
	case ServerEventTypeHello:
		return event.(*ServerEventHello), true
	case ServerEventTypeSTT:
		return event.(*ServerEventSTT), true
	case ServerEventTypeLLM:
		return event.(*ServerEventLLM), true
	case ServerEventTypeTTS:
		return event.(*ServerEventTTS), true
	default:
		return nil, false
	}
}
