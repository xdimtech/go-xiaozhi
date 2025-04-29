package openai

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/samber/lo"

	"github.com/gorilla/websocket"
	"github.com/xdimtech/go-xiaozhi/pkg/config"
	"github.com/xdimtech/go-xiaozhi/pkg/protocol/openai"
	"github.com/xdimtech/go-xiaozhi/pkg/protocol/xiaozhi"
	"github.com/xdimtech/go-xiaozhi/pkg/utils"
)

func (w *XiaozhiHandler) InitProxy(ctx context.Context) error {
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+config.OpenAIConfig().APIKey)
	wsUrl := config.OpenAIConfig().BaseURL + "?model=" + config.OpenAIConfig().Model
	conn, _, err := websocket.DefaultDialer.Dial(wsUrl, headers)
	if err != nil {
		fmt.Errorf("connect to step openai api failed, err: %v", err)
		return err
	}

	w.apiConn = conn
	go func() {
		for {
			msgType, message, err := conn.ReadMessage()
			if err != nil {
				fmt.Errorf("read message from openai api failed, err: %v", err)
				return
			}
			_ = w.handleRealtimeApiEvent(msgType, message)
		}
	}()

	return nil
}

func (w *XiaozhiHandler) SendToRealtimeAPI(event openai.ClientEvent) error {
	if event == nil {
		fmt.Errorf("send to openai api event is nil")
		return nil
	}

	switch event.ClientEventType() {
	case openai.ClientEventTypeSessionUpdate,
		openai.ClientEventTypeInputAudioBufferAppend,
		openai.ClientEventTypeInputAudioBufferCommit:
	default:

	}
	return w.apiConn.WriteJSON(event)
}

func (w *XiaozhiHandler) logServerEvent(event openai.ServerEvent) {
	switch event.ServerEventType() {
	case openai.ServerEventTypeResponseAudioDelta,
		openai.ServerEventTypeResponseAudioDone:
	default:
	}
}

func (w *XiaozhiHandler) handleRealtimeApiEvent(messageType int, p []byte) error {
	event, err := openai.UnmarshalServerEvent(p)
	if err != nil {
		fmt.Errorf("unmarshal server event failed, err: %v", err)
		return err
	}
	w.logServerEvent(event)
	var ev xiaozhi.ServerEvent = nil
	switch event.ServerEventType() {
	case openai.ServerEventTypeError:
		ev, err = w.handleServerError(w.ctx, event)
	case openai.ServerEventTypeSessionCreated:
		ev, err = w.handleSessionCreateOrUpdate(w.ctx, event, false)
	case openai.ServerEventTypeSessionUpdated:
		ev, err = w.handleSessionCreateOrUpdate(w.ctx, event, true)
	case openai.ServerEventTypeInputAudioBufferCommitted:
		ev, err = w.handleInputAudioBufferCommitted(w.ctx, event)
	case openai.ServerEventTypeInputAudioBufferCleared:
		ev, err = w.handleInputAudioBufferCleared(w.ctx, event)
	case openai.ServerEventTypeInputAudioBufferSpeechStarted:
		ev, err = w.handleInputAudioBufferSpeechStarted(w.ctx, event)
	case openai.ServerEventTypeInputAudioBufferSpeechStopped:
		ev, err = w.handleInputAudioBufferSpeechStopped(w.ctx, event)
	case openai.ServerEventTypeResponseCreated:
		ev, err = w.handleResponseCreated(w.ctx, event)
	case openai.ServerEventTypeResponseContentPartAdded:
		ev, err = w.handleContentPartAdded(w.ctx, event)
	case openai.ServerEventTypeConversationItemInputAudioTranscriptionCompleted:
		ev, err = w.handleAsrDone(w.ctx, event)
	case openai.ServerEventTypeResponseAudioTranscriptDelta:
		ev, err = w.handleResponseAudioTranscriptDelta(w.ctx, event)
	case openai.ServerEventTypeResponseAudioTranscriptDone:
		ev, err = w.handleResponseAudioTranscriptDone(w.ctx, event)
	case openai.ServerEventTypeResponseAudioDelta:
		ev, err = w.handleAudioDelta(w.ctx, event)
	case openai.ServerEventTypeResponseAudioDone:
		ev, err = w.handleAudioDone(w.ctx, event)
	case openai.ServerEventTypeResponseContentPartDone:
		ev, err = w.handleContentPartDone(w.ctx, event)
	case openai.ServerEventTypeResponseDone:
		ev, err = w.handleResponseDone(w.ctx, event)
	case openai.ServerEventTypeResponseOutputItemDone:
		ev, err = w.handleResponseOutputItemDone(w.ctx, event)
	case openai.ServerEventTypeConversationItemCreated:
		ev, err = w.handleConversationItemCreated(w.ctx, event)
	}

	if ev != nil {
		_ = w.WriteRespEvent(w.ctx, ev)
	}
	return nil
}

func (w *XiaozhiHandler) closeRealtimeAPI() {
	if w.apiConn == nil {
		return
	}
	err := w.apiConn.Close()
	if err != nil {
		fmt.Errorf("close session fail, err: %v", err)
	}
	w.apiConn = nil
}

func (w *XiaozhiHandler) handleServerError(
	ctx context.Context, event openai.ServerEvent) (xiaozhi.ServerEvent, error) {

	_ev := event.(*openai.ErrorEvent)
	msg := utils.MustToJSON(_ev.Error)
	return &xiaozhi.ServerEventError{
		ServerEventBase: xiaozhi.ServerEventBase{
			Type:      xiaozhi.ServerEventTypeError,
			SessionId: w.GetSessionId(),
		},
		Error: msg,
	}, nil
}

func (w *XiaozhiHandler) handleSessionCreateOrUpdate(ctx context.Context,
	event openai.ServerEvent, update bool) (xiaozhi.ServerEvent, error) {
	if update {
		w.sess.Update(&event.(*openai.SessionUpdatedEvent).Session)
		return &xiaozhi.ServerEventHello{
			ServerEventBase: xiaozhi.ServerEventBase{
				Type:      xiaozhi.ServerEventTypeHello,
				SessionId: w.GetSessionId(),
			},
			Transport: config.Xiaozhi().Transport,
			AudioParams: xiaozhi.AudioParams{
				Format:        w.sess.CliConfig.Format,
				SampleRate:    24000,
				Channels:      1,
				FrameDuration: 60,
			},
		}, nil
	}
	w.sess.Update(&event.(*openai.SessionCreatedEvent).Session)
	return nil, nil
}

func (w *XiaozhiHandler) handleInputAudioBufferCommitted(
	ctx context.Context, event openai.ServerEvent) (xiaozhi.ServerEvent, error) {
	return &xiaozhi.ServerEventTTS{
		ServerEventBase: xiaozhi.ServerEventBase{
			Type:      xiaozhi.ServerEventTypeTTS,
			SessionId: w.GetSessionId(),
		},
		State: xiaozhi.ServerTTSStateStart,
	}, nil
}

func (w *XiaozhiHandler) handleInputAudioBufferCleared(
	ctx context.Context, event openai.ServerEvent) (xiaozhi.ServerEvent, error) {
	return nil, nil
}

func (w *XiaozhiHandler) handleInputAudioBufferSpeechStarted(
	ctx context.Context, event openai.ServerEvent) (xiaozhi.ServerEvent, error) {
	return nil, nil
}

func (w *XiaozhiHandler) handleInputAudioBufferSpeechStopped(
	ctx context.Context, event openai.ServerEvent) (xiaozhi.ServerEvent, error) {
	return nil, nil
}

func (w *XiaozhiHandler) handleResponseCreated(
	ctx context.Context, event openai.ServerEvent) (xiaozhi.ServerEvent, error) {
	return nil, nil
}

func (w *XiaozhiHandler) handleAsrDone(
	ctx context.Context, event openai.ServerEvent) (xiaozhi.ServerEvent, error) {
	_event := event.(*openai.ConversationItemInputAudioTranscriptionCompletedEvent)
	sttEvent := &xiaozhi.ServerEventSTT{
		ServerEventBase: xiaozhi.ServerEventBase{
			Type:      xiaozhi.ServerEventTypeSTT,
			SessionId: w.GetSessionId(),
		},
		Text: _event.Transcript,
	}
	_ = w.WriteRespEvent(w.ctx, sttEvent)

	llmEvent := &xiaozhi.ServerEventLLM{
		ServerEventBase: xiaozhi.ServerEventBase{
			Type:      xiaozhi.ServerEventTypeLLM,
			SessionId: w.GetSessionId(),
		},
		Text:    strconv.Itoa('😊'),
		Emotion: "happy",
	}
	_ = w.WriteRespEvent(w.ctx, llmEvent)
	return nil, nil
}

func (w *XiaozhiHandler) handleContentPartAdded(
	ctx context.Context, event openai.ServerEvent) (xiaozhi.ServerEvent, error) {
	return nil, nil
}

func (w *XiaozhiHandler) handleContentPartDone(
	ctx context.Context, event openai.ServerEvent) (xiaozhi.ServerEvent, error) {

	_event := event.(*openai.ResponseContentPartDoneEvent)
	return &xiaozhi.ServerEventTTS{
		ServerEventBase: xiaozhi.ServerEventBase{
			Type:      xiaozhi.ServerEventTypeTTS,
			SessionId: w.GetSessionId(),
		},
		State: xiaozhi.ServerTTSStateSentenceEnd,
		Text:  lo.FromPtr(_event.Part.Transcript),
	}, nil
}

func (w *XiaozhiHandler) handleResponseDone(
	ctx context.Context, event openai.ServerEvent) (xiaozhi.ServerEvent, error) {
	if w.getWait() > 0 {
		time.Sleep(time.Duration(w.getWait()) * time.Millisecond)
	}
	w.resetFrameTs()
	return &xiaozhi.ServerEventTTS{
		ServerEventBase: xiaozhi.ServerEventBase{
			Type:      xiaozhi.ServerEventTypeTTS,
			SessionId: w.GetSessionId(),
		},
		State: xiaozhi.ServerTTSStateStop,
	}, nil
}

func (w *XiaozhiHandler) handleResponseOutputItemDone(
	ctx context.Context, event openai.ServerEvent) (xiaozhi.ServerEvent, error) {
	return nil, nil
}

func (w *XiaozhiHandler) handleConversationItemCreated(
	ctx context.Context, event openai.ServerEvent) (xiaozhi.ServerEvent, error) {
	return nil, nil
}

func (w *XiaozhiHandler) handleResponseAudioTranscriptDelta(
	ctx context.Context, event openai.ServerEvent) (xiaozhi.ServerEvent, error) {
	return nil, nil
}

func (w *XiaozhiHandler) handleResponseAudioTranscriptDone(
	ctx context.Context, event openai.ServerEvent) (xiaozhi.ServerEvent, error) {
	_event := event.(*openai.ResponseAudioTranscriptDoneEvent)
	return &xiaozhi.ServerEventTTS{
		ServerEventBase: xiaozhi.ServerEventBase{
			Type:      xiaozhi.ServerEventTypeTTS,
			SessionId: w.GetSessionId(),
		},
		State: xiaozhi.ServerTTSStateSentenceStart,
		Text:  _event.Transcript,
	}, nil
}

func (w *XiaozhiHandler) handleAudioDelta(
	ctx context.Context, event openai.ServerEvent) (xiaozhi.ServerEvent, error) {
	_event := event.(*openai.ResponseAudioDeltaEvent)
	w.setFrameTs()
	err := w.audioConverter.ResolvePCM(_event.Delta)
	if err != nil {
		fmt.Errorf("pcm base64 to opus failed, err: %v", err)
		return nil, err
	}
	return nil, nil
}

func (w *XiaozhiHandler) handleAudioDone(
	ctx context.Context, event openai.ServerEvent) (xiaozhi.ServerEvent, error) {
	return nil, nil
}
