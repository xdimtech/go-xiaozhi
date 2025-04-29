package openai

import (
	"encoding/json"
	"fmt"

	"github.com/xdimtech/go-xiaozhi/pkg/utils"
)

type ServerEventType string

const (
	ServerEventTypeError                                            ServerEventType = "error"
	ServerEventTypeSessionCreated                                   ServerEventType = "session.created"
	ServerEventTypeSessionUpdated                                   ServerEventType = "session.updated"
	ServerEventTypeConversationCreated                              ServerEventType = "conversation.created"
	ServerEventTypeInputAudioBufferCommitted                        ServerEventType = "input_audio_buffer.committed"
	ServerEventTypeInputAudioBufferCleared                          ServerEventType = "input_audio_buffer.cleared"
	ServerEventTypeInputAudioBufferSpeechStarted                    ServerEventType = "input_audio_buffer.speech_started"
	ServerEventTypeInputAudioBufferSpeechStopped                    ServerEventType = "input_audio_buffer.speech_stopped"
	ServerEventTypeConversationItemCreated                          ServerEventType = "conversation.item.created"
	ServerEventTypeConversationItemInputAudioTranscriptionCompleted ServerEventType = "conversation.item.input_audio_transcription.completed"
	ServerEventTypeConversationItemInputAudioTranscriptionFailed    ServerEventType = "conversation.item.input_audio_transcription.failed"
	ServerEventTypeConversationItemTruncated                        ServerEventType = "conversation.item.truncated"
	ServerEventTypeConversationItemDeleted                          ServerEventType = "conversation.item.deleted"
	ServerEventTypeResponseCreated                                  ServerEventType = "response.created"
	ServerEventTypeResponseCancelled                                ServerEventType = "response.cancelled"
	ServerEventTypeResponseDone                                     ServerEventType = "response.done"
	ServerEventTypeResponseOutputItemAdded                          ServerEventType = "response.output_item.added"
	ServerEventTypeResponseOutputItemDone                           ServerEventType = "response.output_item.done"
	ServerEventTypeResponseContentPartAdded                         ServerEventType = "response.content_part.added"
	ServerEventTypeResponseContentPartDone                          ServerEventType = "response.content_part.done"
	ServerEventTypeResponseTextDelta                                ServerEventType = "response.text.delta"
	ServerEventTypeResponseTextDone                                 ServerEventType = "response.text.done"
	ServerEventTypeResponseAudioTranscriptDelta                     ServerEventType = "response.audio_transcript.delta"
	ServerEventTypeResponseAudioTranscriptDone                      ServerEventType = "response.audio_transcript.done"
	ServerEventTypeResponseAudioDelta                               ServerEventType = "response.audio.delta"
	ServerEventTypeResponseAudioDone                                ServerEventType = "response.audio.done"
	ServerEventTypeResponseFunctionCallArgumentsDelta               ServerEventType = "response.function_call_arguments.delta"
	ServerEventTypeResponseFunctionCallArgumentsDone                ServerEventType = "response.function_call_arguments.done"
	ServerEventTypeRateLimitsUpdated                                ServerEventType = "rate_limits.updated"
)

// ServerEvent is the interface for server events.
type ServerEvent interface {
	ServerEventType() ServerEventType
	SetBaseEventType(t ServerEventType)
	SetEventId(eventId string)
	GetEventId() string
}

// ServerEventBase is the base struct for all server events.
type ServerEventBase struct {
	// The unique ID of the server event.
	EventID string `json:"event_id,omitempty"`
	// The type of the server event.
	Type ServerEventType `json:"type"`
}

func (m *ServerEventBase) ServerEventType() ServerEventType {
	return m.Type
}

func (m *ServerEventBase) SetBaseEventType(t ServerEventType) {
	m.Type = t
	if m.EventID == "" {
		m.EventID = utils.UniqueID()
	}
}

func (m *ServerEventBase) SetEventId(eventId string) {
	m.EventID = eventId
}

func (m *ServerEventBase) GetEventId() string {
	return m.EventID
}

// ErrorEvent is the event for error.
// Returned when an error occurs.
// See https://platform.openai.com/docs/api-reference/realtime-server-events/error
type ErrorEvent struct {
	ServerEventBase
	// Details of the error.
	Error Error `json:"error"`
}

// SessionCreatedEvent is the event for session created.
// Returned when a session is created. Emitted automatically when a new connection is established.
// See https://platform.openai.com/docs/api-reference/realtime-server-events/session/created
type SessionCreatedEvent struct {
	ServerEventBase
	// The session resource.
	Session ServerSession `json:"session"`
}

// SessionUpdatedEvent is the event for session updated.
// Returned when a session is updated.
// See https://platform.openai.com/docs/api-reference/realtime-server-events/session/updated
type SessionUpdatedEvent struct {
	ServerEventBase
	// The updated session resource.
	Session ServerSession `json:"session"`
}

// ConversationCreatedEvent is the event for conversation created.
type ConversationCreatedEvent struct {
	ServerEventBase
	// The conversation resource.
	Conversation Conversation `json:"conversation"`
}

// InputAudioBufferCommittedEvent is the event for input audio buffer committed.
type InputAudioBufferCommittedEvent struct {
	ServerEventBase
	// The ID of the preceding item after which the new item will be inserted.
	PreviousItemID string `json:"previous_item_id,omitempty"`
	// The ID of the user message item that will be created.
	ItemID string `json:"item_id"`
}

// InputAudioBufferClearedEvent is the event for input audio buffer cleared.
type InputAudioBufferClearedEvent struct {
	ServerEventBase
}

// InputAudioBufferSpeechStartedEvent is the event for input audio buffer speech started.
// Returned in server turn detection mode when speech is detected.
// See https://platform.openai.com/docs/api-reference/realtime-server-events/input_audio_buffer/speech_started
type InputAudioBufferSpeechStartedEvent struct {
	ServerEventBase
	// Milliseconds since the session started when speech was detected.
	AudioStartMs int64 `json:"audio_start_ms"`
	// The ID of the user message item that will be created when speech stops.
	ItemID string `json:"item_id"`
}

// InputAudioBufferSpeechStoppedEvent is the event for input audio buffer speech stopped.
// Returned in server turn detection mode when speech stops.
// See https://platform.openai.com/docs/api-reference/realtime-server-events/input_audio_buffer/speech_stopped
type InputAudioBufferSpeechStoppedEvent struct {
	ServerEventBase
	// Milliseconds since the session started when speech stopped.
	AudioEndMs int64 `json:"audio_end_ms"`
	// The ID of the user message item that will be created.
	ItemID string `json:"item_id"`
}

type ConversationItemCreatedEvent struct {
	ServerEventBase
	PreviousItemID string              `json:"previous_item_id,omitempty"`
	Item           ResponseMessageItem `json:"item"`
}

type ConversationItemInputAudioTranscriptionCompletedEvent struct {
	ServerEventBase
	ItemID       string `json:"item_id"`
	ContentIndex int    `json:"content_index"`
	Transcript   string `json:"transcript"`
}

type ConversationItemInputAudioTranscriptionFailedEvent struct {
	ServerEventBase
	ItemID       string `json:"item_id"`
	ContentIndex int    `json:"content_index"`
	Error        Error  `json:"error"`
}

type ConversationItemTruncatedEvent struct {
	ServerEventBase
	ItemID       string `json:"item_id"`       // The ID of the assistant message item that was truncated.
	ContentIndex int    `json:"content_index"` // The index of the content part that was truncated.
	AudioEndMs   int    `json:"audio_end_ms"`  // The duration up to which the audio was truncated, in milliseconds.
}

type ConversationItemDeletedEvent struct {
	ServerEventBase
	ItemID string `json:"item_id"` // The ID of the item that was deleted.
}

// ResponseCreatedEvent is the event for response created.
// Returned when a new Response is created. The first event of response creation, where the response is in an initial state of "in_progress".
// See https://platform.openai.com/docs/api-reference/realtime-server-events/response/created
type ResponseCreatedEvent struct {
	ServerEventBase
	// The response resource.
	Response Response `json:"response"`
}

// ResponseDoneEvent is the event for response done.
// Returned when a Response is done streaming. Always emitted, no matter the final state.
// See https://platform.openai.com/docs/api-reference/realtime-server-events/response/done
type ResponseDoneEvent struct {
	ServerEventBase
	// The response resource.
	Response Response `json:"response"`
}

// ResponseCreatedEvent is the event for response created.
// Returned when a new Response is created. The first event of response creation, where the response is in an initial state of "in_progress".
// See https://platform.openai.com/docs/api-reference/realtime-server-events/response/created
type ResponseCancelledEvent struct {
	ServerEventBase
}

// ResponseOutputItemAddedEvent is the event for response output item added.
// Returned when a new Item is created during response generation.
// See https://platform.openai.com/docs/api-reference/realtime-server-events/response/output_item/added
type ResponseOutputItemAddedEvent struct {
	ServerEventBase
	// The ID of the response to which the item belongs.
	ResponseID string `json:"response_id"`
	// The index of the output item in the response.
	OutputIndex int `json:"output_index"`
	// The item that was added.
	Item ResponseMessageItem `json:"item"`
}

// ResponseOutputItemDoneEvent is the event for response output item done.
// Returned when an Item is done streaming. Also emitted when a Response is interrupted, incomplete, or cancelled.
// See https://platform.openai.com/docs/api-reference/realtime-server-events/response/output_item/done
type ResponseOutputItemDoneEvent struct {
	ServerEventBase
	// The ID of the response to which the item belongs.
	ResponseID string `json:"response_id"`
	// The index of the output item in the response.
	OutputIndex int `json:"output_index"`
	// The completed item.
	Item ResponseMessageItem `json:"item"`
}

// ResponseContentPartAddedEvent is the event for response content part added.
// Returned when a new content part is added to an assistant message item during response generation.
// See https://platform.openai.com/docs/api-reference/realtime-server-events/response/content_part/added
type ResponseContentPartAddedEvent struct {
	ServerEventBase
	ResponseID   string             `json:"response_id"`
	ItemID       string             `json:"item_id"`
	OutputIndex  int                `json:"output_index"`
	ContentIndex int                `json:"content_index"`
	Part         MessageContentPart `json:"part"`
}

// ResponseContentPartDoneEvent is the event for response content part done.
// Returned when a content part is done streaming in an assistant message item. Also emitted when a Response is interrupted, incomplete, or cancelled.
// See https://platform.openai.com/docs/api-reference/realtime-server-events/response/content_part/done
type ResponseContentPartDoneEvent struct {
	ServerEventBase
	// The ID of the response.
	ResponseID string `json:"response_id"`
	// The ID of the item to which the content part was added.
	ItemID string `json:"item_id"`
	// The index of the output item in the response.
	OutputIndex int `json:"output_index"`
	// The index of the content part in the item's content array.
	ContentIndex int `json:"content_index"`
	// The content part that was added.
	Part MessageContentPart `json:"part"`
}

// ResponseTextDeltaEvent is the event for response text delta.
// Returned when the text value of a "text" content part is updated.
// See https://platform.openai.com/docs/api-reference/realtime-server-events/response/text/delta
type ResponseTextDeltaEvent struct {
	ServerEventBase
	ResponseID   string `json:"response_id"`
	ItemID       string `json:"item_id"`
	OutputIndex  int    `json:"output_index"`
	ContentIndex int    `json:"content_index"`
	Delta        string `json:"delta"`
}

// ResponseTextDoneEvent is the event for response text done.
// Returned when the text value of a "text" content part is done streaming. Also emitted when a Response is interrupted, incomplete, or cancelled.
// See https://platform.openai.com/docs/api-reference/realtime-server-events/response/text/done
type ResponseTextDoneEvent struct {
	ServerEventBase
	ResponseID   string `json:"response_id"`
	ItemID       string `json:"item_id"`
	OutputIndex  int    `json:"output_index"`
	ContentIndex int    `json:"content_index"`
	Text         string `json:"text"`
}

// ResponseAudioTranscriptDeltaEvent is the event for response audio transcript delta.
// Returned when the model-generated transcription of audio output is updated.
// See https://platform.openai.com/docs/api-reference/realtime-server-events/response/audio_transcript/delta
type ResponseAudioTranscriptDeltaEvent struct {
	ServerEventBase
	// The ID of the response.
	ResponseID string `json:"response_id"`
	// The ID of the item.
	ItemID string `json:"item_id"`
	// The index of the output item in the response.
	OutputIndex int `json:"output_index"`
	// The index of the content part in the item's content array.
	ContentIndex int `json:"content_index"`
	// The transcript delta.
	Delta string `json:"delta"`
}

// ResponseAudioTranscriptDoneEvent is the event for response audio transcript done.
// Returned when the model-generated transcription of audio output is done streaming. Also emitted when a Response is interrupted, incomplete, or cancelled.
// See https://platform.openai.com/docs/api-reference/realtime-server-events/response/audio_transcript/done
type ResponseAudioTranscriptDoneEvent struct {
	ServerEventBase
	// The ID of the response.
	ResponseID string `json:"response_id"`
	// The ID of the item.
	ItemID string `json:"item_id"`
	// The index of the output item in the response.
	OutputIndex int `json:"output_index"`
	// The index of the content part in the item's content array.
	ContentIndex int `json:"content_index"`
	// The final transcript of the audio.
	Transcript string `json:"transcript"`
}

// ResponseAudioDeltaEvent is the event for response audio delta.
// Returned when the model-generated audio is updated.
// See https://platform.openai.com/docs/api-reference/realtime-server-events/response/audio/delta
type ResponseAudioDeltaEvent struct {
	ServerEventBase
	// The ID of the response.
	ResponseID string `json:"response_id"`
	// The ID of the item.
	ItemID string `json:"item_id"`
	// The index of the output item in the response.
	OutputIndex int `json:"output_index"`
	// The index of the content part in the item's content array.
	ContentIndex int `json:"content_index"`
	// Base64-encoded audio data delta.
	Delta string `json:"delta"`
}

// ResponseAudioDoneEvent is the event for response audio done.
// Returned when the model-generated audio is done. Also emitted when a Response is interrupted, incomplete, or cancelled.
// See https://platform.openai.com/docs/api-reference/realtime-server-events/response/audio/done
type ResponseAudioDoneEvent struct {
	ServerEventBase
	// The ID of the response.
	ResponseID string `json:"response_id"`
	// The ID of the item.
	ItemID string `json:"item_id"`
	// The index of the output item in the response.
	OutputIndex int `json:"output_index"`
	// The index of the content part in the item's content array.
	ContentIndex int `json:"content_index"`
}

// ResponseFunctionCallArgumentsDeltaEvent is the event for response function call arguments delta.
// Returned when the model-generated function call arguments are updated.
// See https://platform.openai.com/docs/api-reference/realtime-server-events/response/function_call_arguments/delta
type ResponseFunctionCallArgumentsDeltaEvent struct {
	ServerEventBase
	// The ID of the response.
	ResponseID string `json:"response_id"`
	// The ID of the item.
	ItemID string `json:"item_id"`
	// The index of the output item in the response.
	OutputIndex int `json:"output_index"`
	// The ID of the function call.
	CallID string `json:"call_id"`
	// The arguments delta as a JSON string.
	Delta string `json:"delta"`
}

// ResponseFunctionCallArgumentsDoneEvent is the event for response function call arguments done.
// Returned when the model-generated function call arguments are done streaming. Also emitted when a Response is interrupted, incomplete, or cancelled.
// See https://platform.openai.com/docs/api-reference/realtime-server-events/response/function_call_arguments/done
type ResponseFunctionCallArgumentsDoneEvent struct {
	ServerEventBase
	// The ID of the response.
	ResponseID string `json:"response_id"`
	// The ID of the item.
	ItemID string `json:"item_id"`
	// The index of the output item in the response.
	OutputIndex int `json:"output_index"`
	// The ID of the function call.
	CallID string `json:"call_id"`
	// The final arguments as a JSON string.
	Arguments string `json:"arguments"`
	// The name of the function. Not shown in API reference but present in the actual event.
	Name string `json:"name"`
}

// RateLimitsUpdatedEvent is the event for rate limits updated.
// Emitted after every "response.done" event to indicate the updated rate limits.
// See https://platform.openai.com/docs/api-reference/realtime-server-events/rate_limits/updated
type RateLimitsUpdatedEvent struct {
	ServerEventBase
	// List of rate limit information.
	RateLimits []RateLimit `json:"rate_limits"`
}

type ServerEventInterface interface {
	ErrorEvent |
		SessionCreatedEvent |
		SessionUpdatedEvent |
		ConversationCreatedEvent |
		InputAudioBufferCommittedEvent |
		InputAudioBufferClearedEvent |
		InputAudioBufferSpeechStartedEvent |
		InputAudioBufferSpeechStoppedEvent |
		ConversationItemCreatedEvent |
		ConversationItemInputAudioTranscriptionCompletedEvent |
		ConversationItemInputAudioTranscriptionFailedEvent |
		ConversationItemTruncatedEvent |
		ConversationItemDeletedEvent |
		ResponseCancelledEvent |
		ResponseCreatedEvent |
		ResponseDoneEvent |
		ResponseOutputItemAddedEvent |
		ResponseOutputItemDoneEvent |
		ResponseContentPartAddedEvent |
		ResponseContentPartDoneEvent |
		ResponseTextDeltaEvent |
		ResponseTextDoneEvent |
		ResponseAudioTranscriptDeltaEvent |
		ResponseAudioTranscriptDoneEvent |
		ResponseAudioDeltaEvent |
		ResponseAudioDoneEvent |
		ResponseFunctionCallArgumentsDeltaEvent |
		ResponseFunctionCallArgumentsDoneEvent |
		RateLimitsUpdatedEvent
}

func unmarshalServerEvent[T ServerEventInterface](data []byte) (*T, error) {
	var t T
	err := json.Unmarshal(data, &t)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func MarshalServerEvent(event ServerEvent) ([]byte, error) {
	switch ev := event.(type) {
	case *ErrorEvent:
		ev.SetBaseEventType(ServerEventTypeError)
	case *SessionCreatedEvent:
		ev.SetBaseEventType(ServerEventTypeSessionCreated)
	case *SessionUpdatedEvent:
		ev.SetBaseEventType(ServerEventTypeSessionUpdated)
	case *ConversationCreatedEvent:
		ev.SetBaseEventType(ServerEventTypeConversationCreated)
	case *InputAudioBufferCommittedEvent:
		ev.SetBaseEventType(ServerEventTypeInputAudioBufferCommitted)
	case *InputAudioBufferClearedEvent:
		ev.SetBaseEventType(ServerEventTypeInputAudioBufferCleared)
	case *InputAudioBufferSpeechStartedEvent:
		ev.SetBaseEventType(ServerEventTypeInputAudioBufferSpeechStarted)
	case *InputAudioBufferSpeechStoppedEvent:
		ev.SetBaseEventType(ServerEventTypeInputAudioBufferSpeechStopped)
	case *ConversationItemCreatedEvent:
		ev.SetBaseEventType(ServerEventTypeConversationItemCreated)
	case *ConversationItemInputAudioTranscriptionCompletedEvent:
		ev.SetBaseEventType(ServerEventTypeConversationItemInputAudioTranscriptionCompleted)
	case *ConversationItemInputAudioTranscriptionFailedEvent:
		ev.SetBaseEventType(ServerEventTypeConversationItemInputAudioTranscriptionFailed)
	case *ConversationItemTruncatedEvent:
		ev.SetBaseEventType(ServerEventTypeConversationItemTruncated)
	case *ConversationItemDeletedEvent:
		ev.SetBaseEventType(ServerEventTypeConversationItemDeleted)
	case *ResponseCancelledEvent:
		ev.SetBaseEventType(ServerEventTypeResponseCancelled)
	case *ResponseCreatedEvent:
		ev.SetBaseEventType(ServerEventTypeResponseCreated)
	case *ResponseDoneEvent:
		ev.SetBaseEventType(ServerEventTypeResponseDone)
	case *ResponseOutputItemAddedEvent:
		ev.SetBaseEventType(ServerEventTypeResponseOutputItemAdded)
	case *ResponseOutputItemDoneEvent:
		ev.SetBaseEventType(ServerEventTypeResponseOutputItemDone)
	case *ResponseContentPartAddedEvent:
		ev.SetBaseEventType(ServerEventTypeResponseContentPartAdded)
	case *ResponseContentPartDoneEvent:
		ev.SetBaseEventType(ServerEventTypeResponseContentPartDone)
	case *ResponseTextDeltaEvent:
		ev.SetBaseEventType(ServerEventTypeResponseTextDelta)
	case *ResponseTextDoneEvent:
		ev.SetBaseEventType(ServerEventTypeResponseTextDone)
	case *ResponseAudioTranscriptDeltaEvent:
		ev.SetBaseEventType(ServerEventTypeResponseAudioTranscriptDelta)
	case *ResponseAudioTranscriptDoneEvent:
		ev.SetBaseEventType(ServerEventTypeResponseAudioTranscriptDone)
	case *ResponseAudioDeltaEvent:
		ev.SetBaseEventType(ServerEventTypeResponseAudioDelta)
	case *ResponseAudioDoneEvent:
		ev.SetBaseEventType(ServerEventTypeResponseAudioDone)
	case *ResponseFunctionCallArgumentsDeltaEvent:
		ev.SetBaseEventType(ServerEventTypeResponseFunctionCallArgumentsDelta)
	case *ResponseFunctionCallArgumentsDoneEvent:
		ev.SetBaseEventType(ServerEventTypeResponseFunctionCallArgumentsDone)
	case *RateLimitsUpdatedEvent:
		ev.SetBaseEventType(ServerEventTypeRateLimitsUpdated)
	default:
		// This should never happen.
		return nil, fmt.Errorf("unknown server event type: %s", ev.ServerEventType())
	}
	return json.Marshal(event)
}

// UnmarshalServerEvent unmarshals the server event from the given JSON data.
func UnmarshalServerEvent(data []byte) (ServerEvent, error) { //nolint:funlen,cyclop // TODO: optimize
	var eventType struct {
		Type ServerEventType `json:"type"`
	}
	err := json.Unmarshal(data, &eventType)
	if err != nil {
		return nil, err
	}
	switch eventType.Type {
	case ServerEventTypeError:
		return unmarshalServerEvent[ErrorEvent](data)
	case ServerEventTypeSessionCreated:
		return unmarshalServerEvent[SessionCreatedEvent](data)
	case ServerEventTypeSessionUpdated:
		return unmarshalServerEvent[SessionUpdatedEvent](data)
	case ServerEventTypeConversationCreated:
		return unmarshalServerEvent[ConversationCreatedEvent](data)
	case ServerEventTypeInputAudioBufferCommitted:
		return unmarshalServerEvent[InputAudioBufferCommittedEvent](data)
	case ServerEventTypeInputAudioBufferCleared:
		return unmarshalServerEvent[InputAudioBufferClearedEvent](data)
	case ServerEventTypeInputAudioBufferSpeechStarted:
		return unmarshalServerEvent[InputAudioBufferSpeechStartedEvent](data)
	case ServerEventTypeInputAudioBufferSpeechStopped:
		return unmarshalServerEvent[InputAudioBufferSpeechStoppedEvent](data)
	case ServerEventTypeConversationItemCreated:
		return unmarshalServerEvent[ConversationItemCreatedEvent](data)
	case ServerEventTypeConversationItemInputAudioTranscriptionCompleted:
		return unmarshalServerEvent[ConversationItemInputAudioTranscriptionCompletedEvent](data)
	case ServerEventTypeConversationItemInputAudioTranscriptionFailed:
		return unmarshalServerEvent[ConversationItemInputAudioTranscriptionFailedEvent](data)
	case ServerEventTypeConversationItemTruncated:
		return unmarshalServerEvent[ConversationItemTruncatedEvent](data)
	case ServerEventTypeConversationItemDeleted:
		return unmarshalServerEvent[ConversationItemDeletedEvent](data)
	case ServerEventTypeResponseCreated:
		return unmarshalServerEvent[ResponseCreatedEvent](data)
	case ServerEventTypeResponseDone:
		return unmarshalServerEvent[ResponseDoneEvent](data)
	case ServerEventTypeResponseOutputItemAdded:
		return unmarshalServerEvent[ResponseOutputItemAddedEvent](data)
	case ServerEventTypeResponseOutputItemDone:
		return unmarshalServerEvent[ResponseOutputItemDoneEvent](data)
	case ServerEventTypeResponseContentPartAdded:
		return unmarshalServerEvent[ResponseContentPartAddedEvent](data)
	case ServerEventTypeResponseContentPartDone:
		return unmarshalServerEvent[ResponseContentPartDoneEvent](data)
	case ServerEventTypeResponseTextDelta:
		return unmarshalServerEvent[ResponseTextDeltaEvent](data)
	case ServerEventTypeResponseTextDone:
		return unmarshalServerEvent[ResponseTextDoneEvent](data)
	case ServerEventTypeResponseAudioTranscriptDelta:
		return unmarshalServerEvent[ResponseAudioTranscriptDeltaEvent](data)
	case ServerEventTypeResponseAudioTranscriptDone:
		return unmarshalServerEvent[ResponseAudioTranscriptDoneEvent](data)
	case ServerEventTypeResponseAudioDelta:
		return unmarshalServerEvent[ResponseAudioDeltaEvent](data)
	case ServerEventTypeResponseAudioDone:
		return unmarshalServerEvent[ResponseAudioDoneEvent](data)
	case ServerEventTypeResponseFunctionCallArgumentsDelta:
		return unmarshalServerEvent[ResponseFunctionCallArgumentsDeltaEvent](data)
	case ServerEventTypeResponseFunctionCallArgumentsDone:
		return unmarshalServerEvent[ResponseFunctionCallArgumentsDoneEvent](data)
	case ServerEventTypeRateLimitsUpdated:
		return unmarshalServerEvent[RateLimitsUpdatedEvent](data)
	default:
		// This should never happen.
		return nil, fmt.Errorf("unknown server event type: %s", eventType.Type)
	}
}
