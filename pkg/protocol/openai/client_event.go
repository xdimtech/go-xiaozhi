package openai

import (
	"encoding/json"
	"fmt"
)

// ClientEventType is the type of client event. See https://platform.openai.com/docs/guides/realtime/client-events
type ClientEventType string

const (
	ClientEventTypeSessionUpdate              ClientEventType = "session.update"
	ClientEventTypeInputAudioBufferAppend     ClientEventType = "input_audio_buffer.append"
	ClientEventTypeInputAudioBufferCommit     ClientEventType = "input_audio_buffer.commit"
	ClientEventTypeInputAudioBufferClear      ClientEventType = "input_audio_buffer.clear"
	ClientEventTypeInputAudioBufferTranscript ClientEventType = "input_audio_buffer.transcript"
	ClientEventTypeInputTextBufferTranscript  ClientEventType = "input_text_buffer.transcript"
	ClientEventTypeConversationItemCreate     ClientEventType = "conversation.item.create"
	ClientEventTypeConversationItemTruncate   ClientEventType = "conversation.item.truncate"
	ClientEventTypeConversationItemDelete     ClientEventType = "conversation.item.delete"
	ClientEventTypeResponseCreate             ClientEventType = "response.create"
	ClientEventTypeResponseCancel             ClientEventType = "response.cancel"
)

// ClientEvent is the interface for client event.
type ClientEvent interface {
	ClientEventType() ClientEventType
	GetExtra() *ClientExtraParams
	GetEventID() string
}

type ClientExtraParams struct {
	TurnId string `json:"turn_id"`
}

// ClientEventBase is the base struct for all client events.
type ClientEventBase struct {
	EventID string             `json:"event_id,omitempty"`
	Type    ClientEventType    `json:"type"`
	Extra   *ClientExtraParams `json:"extra,omitempty"`
}

func (e ClientEventBase) ClientEventType() ClientEventType {
	return e.Type
}

func (e ClientEventBase) GetExtra() *ClientExtraParams {
	return e.Extra
}

func (e ClientEventBase) GetEventID() string {
	return e.EventID
}

type ClientSession struct {
	// The set of modalities the model can respond with. To disable audio, set this to ["text"].
	Modalities []Modality `json:"modalities,omitempty"`
	// The default system instructions prepended to model calls.
	Instructions *string `json:"instructions,omitempty"`
	// The voice the model uses to respond - one of alloy, echo, or shimmer. Cannot be changed once the model has responded with audio at least once.
	Voice *Voice `json:"voice,omitempty"`
	// The format of input audio. Options are "pcm16", "g711_ulaw", or "g711_alaw".
	InputAudioFormat *AudioFormat `json:"input_audio_format,omitempty"`
	// The format of output audio. Options are "pcm16", "g711_ulaw", or "g711_alaw".
	OutputAudioFormat *AudioFormat `json:"output_audio_format,omitempty"`
	// Configuration for input audio transcription. Can be set to `nil` to turn off.
	InputAudioTranscription *InputAudioTranscription `json:"input_audio_transcription,omitempty"`
	// Configuration for turn detection. Can be set to `nil` to turn off.
	TurnDetection *TurnDetection `json:"turn_detection"`
	// Tools (functions) available to the model.
	Tools []Tool `json:"tools,omitempty"`
	// How the model chooses tools. Options are "auto", "none", "required", or specify a function.
	ToolChoice interface{} `json:"tool_choice,omitempty"`
	// Sampling temperature for the model.
	Temperature *float32 `json:"temperature,omitempty"`
	// Maximum number of output tokens for a single assistant response, inclusive of tool calls. Provide an integer between 1 and 4096 to limit output tokens, or "inf" for the maximum available tokens for a given model. Defaults to "inf".
	MaxOutputTokens *IntOrInf `json:"max_response_output_tokens,omitempty"`

	History []MessageItem `json:"history,omitempty"`

	BuiltInTools []string `json:"built_in_tools,omitempty"`
}

// SessionUpdateEvent is the event for session update.
// Send this event to update the session’s default configuration.
// See https://platform.openai.com/docs/api-reference/realtime-client-events/session/update
type SessionUpdateEvent struct {
	ClientEventBase
	// Session configuration to update.
	Session ClientSession `json:"session"`
}

func (m SessionUpdateEvent) ClientEventType() ClientEventType {
	return ClientEventTypeSessionUpdate
}

// InputAudioBufferAppendEvent is the event for input audio buffer append.
type InputAudioBufferAppendEvent struct {
	ClientEventBase
	Audio string `json:"audio"` // Base64-encoded audio bytes.
}

func (m InputAudioBufferAppendEvent) ClientEventType() ClientEventType {
	return ClientEventTypeInputAudioBufferAppend
}

// InputAudioBufferCommitEvent is the event for input audio buffer commit.
type InputAudioBufferCommitEvent struct {
	ClientEventBase
	Transcript string `json:"transcript"`
	Language   string `json:"language"`
}

func (m InputAudioBufferCommitEvent) ClientEventType() ClientEventType {
	return ClientEventTypeInputAudioBufferCommit
}

// InputAudioBufferClearEvent is the event for input audio buffer clear.
type InputAudioBufferClearEvent struct {
	ClientEventBase
}

func (m InputAudioBufferClearEvent) ClientEventType() ClientEventType {
	return ClientEventTypeInputAudioBufferClear
}

// ConversationItemCreateEvent is the event for conversation item create.
type ConversationItemCreateEvent struct {
	ClientEventBase
	// The ID of the preceding item after which the new item will be inserted.
	PreviousItemID string `json:"previous_item_id,omitempty"`
	// The item to add to the conversation.
	Item MessageItem `json:"item"`
}

func (m ConversationItemCreateEvent) ClientEventType() ClientEventType {
	return ClientEventTypeConversationItemCreate
}

// ConversationItemTruncateEvent is the event for conversation item truncate.
type ConversationItemTruncateEvent struct {
	ClientEventBase
	// The ID of the assistant message item to truncate.
	ItemID string `json:"item_id"`
	// The index of the content part to truncate.
	ContentIndex int `json:"content_index"`
	// Inclusive duration up to which audio is truncated, in milliseconds.
	AudioEndMs int `json:"audio_end_ms"`
}

func (m ConversationItemTruncateEvent) ClientEventType() ClientEventType {
	return ClientEventTypeConversationItemTruncate
}

// ConversationItemDeleteEvent is the event for conversation item delete.
type ConversationItemDeleteEvent struct {
	ClientEventBase
	// The ID of the item to delete.
	ItemID string `json:"item_id"`
}

func (m ConversationItemDeleteEvent) ClientEventType() ClientEventType {
	return ClientEventTypeConversationItemDelete
}

// ResponseCreateEvent is the event for response create.
type ResponseCreateEvent struct {
	ClientEventBase
	// Configuration for the response.
	Response ResponseCreateParams `json:"response"`
}

func (m ResponseCreateEvent) ClientEventType() ClientEventType {
	return ClientEventTypeResponseCreate
}

// ResponseCancelEvent is the event for response cancel.
type ResponseCancelEvent struct {
	ClientEventBase
}

func (m ResponseCancelEvent) ClientEventType() ClientEventType {
	return ClientEventTypeResponseCancel
}

type ClientEventInterface interface {
	SessionUpdateEvent |
		InputAudioBufferAppendEvent |
		InputAudioBufferCommitEvent |
		InputAudioBufferClearEvent |
		ConversationItemCreateEvent |
		ConversationItemTruncateEvent |
		ConversationItemDeleteEvent |
		ResponseCreateEvent |
		ResponseCancelEvent
}

// 该函数follow openai的session.update的定义
// 会根据用户传入的turn_detection字段, 调整手动或自动模式
// 如果用户传入的turn_detection字段为nil, 需要切到手动模式
func unmarshalSessionUpdateEvent(data []byte) (*SessionUpdateEvent, error) {
	type tmpSessionUpdateEvent struct {
		Session map[string]interface{} `json:"session"`
	}
	var tmp tmpSessionUpdateEvent
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return nil, err
	}

	var t SessionUpdateEvent
	err = json.Unmarshal(data, &t)
	if err != nil {
		return nil, err
	}

	// 如果不存在turn_detection字段，认为什么也不做
	if _, ok := tmp.Session["turn_detection"]; !ok {
		return &t, nil
	} else if tmp.Session["turn_detection"] == nil {
		// 如果用户传入的turn_detection字段为nil，则需要调为手动模式
		t.Session.TurnDetection = &TurnDetection{
			Type: ClientTurnDetectionTypeUnspecified,
		}
		return &t, nil
	} else {
		// 否则直接使用户传入的turn_detection字段,
		// 直接返回SessionUpdateEvent序列化后的结果即可
		return &t, nil
	}
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

// UnmarshalServerEvent unmarshals the server event from the given JSON data.
func UnmarshalClientEvent(data []byte) (ClientEvent, error) {
	var eventType struct {
		Type ClientEventType `json:"type"`
	}
	err := json.Unmarshal(data, &eventType)
	if err != nil {
		return nil, err
	}

	switch eventType.Type {
	case ClientEventTypeSessionUpdate:
		return unmarshalSessionUpdateEvent(data)
	case ClientEventTypeInputAudioBufferAppend:
		return unmarshalClientEvent[InputAudioBufferAppendEvent](data)
	case ClientEventTypeInputAudioBufferCommit:
		return unmarshalClientEvent[InputAudioBufferCommitEvent](data)
	case ClientEventTypeInputAudioBufferClear:
		return unmarshalClientEvent[InputAudioBufferClearEvent](data)
	case ClientEventTypeConversationItemCreate:
		return unmarshalClientEvent[ConversationItemCreateEvent](data)
	case ClientEventTypeConversationItemTruncate:
		return unmarshalClientEvent[ConversationItemTruncateEvent](data)
	case ClientEventTypeConversationItemDelete:
		return unmarshalClientEvent[ConversationItemDeleteEvent](data)
	case ClientEventTypeResponseCreate:
		return unmarshalClientEvent[ResponseCreateEvent](data)
	case ClientEventTypeResponseCancel:
		return unmarshalClientEvent[ResponseCancelEvent](data)
	default:
		return nil, fmt.Errorf("unknown client event type: %s", eventType.Type)
	}
}
