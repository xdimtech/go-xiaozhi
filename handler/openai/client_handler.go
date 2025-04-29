package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/xdimtech/go-xiaozhi/pkg/audio"
	"github.com/xdimtech/go-xiaozhi/pkg/config"

	"github.com/gorilla/websocket"
	"github.com/samber/lo"
	"github.com/xdimtech/go-xiaozhi/pkg/protocol/openai"
	"github.com/xdimtech/go-xiaozhi/pkg/protocol/xiaozhi"
	"github.com/xdimtech/go-xiaozhi/pkg/utils"
)

type XiaozhiHandler struct {
	ctx               context.Context
	cliConn           *websocket.Conn
	apiConn           *websocket.Conn
	sess              *ApiSession
	closed            atomic.Bool
	writeQueue        chan any
	audioConverter    *audio.Converter
	firstDeltaTs      int64
	totalOpusDuration int
}

func NewXiaozhiHandler(ctx context.Context, conn *websocket.Conn) (*XiaozhiHandler, error) {
	sess := NewApiSession(ctx, config.OpenAIConfig().Model)
	handler := &XiaozhiHandler{
		ctx:        ctx,
		sess:       sess,
		cliConn:    conn,
		writeQueue: make(chan any, WriteQueueSize),
	}
	if err := handler.InitProxy(ctx); err != nil {
		return nil, err
	}
	return handler, nil
}

func (r *XiaozhiHandler) resetFrameTs() {
	r.firstDeltaTs = 0
	r.totalOpusDuration = 0
}

func (r *XiaozhiHandler) setFrameTs() {
	if r.firstDeltaTs != 0 {
		return
	}
	r.firstDeltaTs = time.Now().UnixMilli()
}

func (r *XiaozhiHandler) getWait() int64 {
	now := time.Now().UnixMilli()
	if r.firstDeltaTs == 0 {
		return 0
	}
	if r.totalOpusDuration == 0 {
		return 0
	}
	if wait := int64(r.totalOpusDuration) - (now - r.firstDeltaTs); wait > 0 {
		return wait
	}
	return 0
}

func (r *XiaozhiHandler) addOpusDuration() {
	r.totalOpusDuration += r.sess.CliConfig.FrameDuration
}

func (r *XiaozhiHandler) GetSessionId() string {
	if r.sess != nil {
		return r.sess.GetSessionId()
	}
	return ""
}

func (r *XiaozhiHandler) Close(ctx context.Context) error {
	if r.sess != nil {
		r.sess.Close()
	}
	close(r.writeQueue)
	r.closed.Store(true)
	return nil
}

func (r *XiaozhiHandler) Recv(ctx context.Context) chan any {
	return r.writeQueue
}

func (r *XiaozhiHandler) MarshalServerEvent(ev any) ([]byte, error) {
	event, ok := ev.(xiaozhi.ServerEvent)
	if !ok {
		return nil, errors.New("invalid ServerEvent")
	}
	return json.Marshal(event)
}

func (r *XiaozhiHandler) UnmarshalClientTextEvent(data []byte) (any, error) {
	return xiaozhi.UnmarshalClientEvent(data)
}

func (r *XiaozhiHandler) UnmarshalClientBinEvent(data []byte) (any, error) {
	return xiaozhi.UnmarshalClientBinEvent(data)
}

func (r *XiaozhiHandler) DispatchClientEvent(ctx context.Context, ev any) (error, bool) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Errorf("dispatch client event panic, err:%v", err)
		}
	}()

	event, ok := ev.(xiaozhi.ClientEvent)
	if !ok {
		return errors.New("invalid RealtimeClientEvent"), false
	}

	var rtEvent openai.ClientEvent = nil
	var err error = nil
	switch ev := event.(type) {
	case *xiaozhi.ClientEventHello:
		rtEvent, err = r.handleHelloEvent(ctx, ev)
	case *xiaozhi.ClientEventListen:
		rtEvent, err = r.handleListenEvent(ctx, ev)
	case *xiaozhi.ClientEventAppendBuffer:
		rtEvent, err = r.handleInputAudioBufferAppend(ctx, ev)
	case *xiaozhi.ClientEventIot:
		rtEvent, err = r.handleIotEvent(ctx, ev)
	default:
		fmt.Errorf("unknown event type: %T", ev)
		return nil, false
	}

	if rtEvent == nil || err != nil {
		return err, false
	}
	if err := r.SendToRealtimeAPI(rtEvent); err != nil {
		fmt.Errorf("send to realtime api failed, err: %v", err)
		return err, false
	}

	return nil, false
}

func (r *XiaozhiHandler) handleHelloEvent(ctx context.Context,
	event *xiaozhi.ClientEventHello) (openai.ClientEvent, error) {

	if event.GetAudioParams() == nil ||
		event.GetAudioParams().SampleRate == 0 ||
		event.GetAudioParams().Channels == 0 ||
		event.GetAudioParams().Format == "" ||
		event.GetAudioParams().FrameDuration == 0 {
		return nil, errors.New("invalid audio params")
	}

	frameSize := event.GetAudioParams().FrameDuration * event.GetAudioParams().SampleRate / 1000
	r.audioConverter = audio.NewConverter(event.GetAudioParams().SampleRate,
		event.GetAudioParams().Channels, event.GetAudioParams().FrameDuration, frameSize, r.WriteRespEvent)
	r.sess.CliConfig = &ClientConfig{
		Format:        event.GetAudioParams().Format,
		SampleRate:    event.GetAudioParams().SampleRate,
		Channels:      event.GetAudioParams().Channels,
		FrameDuration: event.GetAudioParams().FrameDuration,
		FrameSize:     frameSize,
	}

	systemPrompt := config.OpenAIConfig().SystemPrompt
	maxToken := openai.IntOrInf(4096)
	eventID := utils.UniqueID()
	pbEvent := &openai.SessionUpdateEvent{
		ClientEventBase: openai.ClientEventBase{
			EventID: eventID,
			Type:    openai.ClientEventTypeSessionUpdate,
		},
		Session: openai.ClientSession{
			Instructions: lo.ToPtr(systemPrompt),
			Modalities: []openai.Modality{
				openai.ModalityText,
				openai.ModalityAudio,
			},
			Voice:             lo.ToPtr(openai.Voice(r.sess.defaultVoice)),
			InputAudioFormat:  lo.ToPtr(openai.AudioFormatPcm16),
			OutputAudioFormat: lo.ToPtr(openai.AudioFormatPcm16),
			ToolChoice:        openai.ToolChoiceRequired,
			MaxOutputTokens:   lo.ToPtr(maxToken),
			TurnDetection: &openai.TurnDetection{
				Type: openai.ClientTurnDetectionTypeServerVad,
			},
			BuiltInTools: []string{
				"web_search",
			},
		},
	}

	return pbEvent, nil
}

func (r *XiaozhiHandler) handleListenEvent(ctx context.Context,
	event *xiaozhi.ClientEventListen) (openai.ClientEvent, error) {
	if event.State == xiaozhi.ClientStateListenStart {
		// 新的一轮对话开始
	} else if event.State == xiaozhi.ClientStateListenStop {
		// 对话结束
	} else if event.State == xiaozhi.ClientStateListenDetect {

	} else if event.State == xiaozhi.ClientStateIdle {
		// 空闲状态
	}

	return nil, nil
}

func (r *XiaozhiHandler) handleInputAudioBufferAppend(ctx context.Context,
	event *xiaozhi.ClientEventAppendBuffer) (*openai.InputAudioBufferAppendEvent, error) {

	audioData := event.Bytes
	if len(audioData) == 0 {
		return nil, nil
	}

	b64Data, err := r.audioConverter.OpusToPcmBase64(audioData)
	if err != nil {
		return nil, err
	}
	pbEvent := &openai.InputAudioBufferAppendEvent{
		ClientEventBase: openai.ClientEventBase{
			EventID: utils.UniqueID(),
			Type:    openai.ClientEventTypeInputAudioBufferAppend,
		},
		Audio: b64Data,
	}
	return pbEvent, nil
}

func (r *XiaozhiHandler) handleIotEvent(
	ctx context.Context, ev *xiaozhi.ClientEventIot) (openai.ClientEvent, error) {
	return nil, nil
}

func (w *XiaozhiHandler) BuildErrorEvent(ctx context.Context, err error) interface{} {
	return &xiaozhi.ServerEventError{
		ServerEventBase: xiaozhi.ServerEventBase{
			Type: xiaozhi.ServerEventTypeError,
		},
		Error: err.Error(),
	}
}

func (w *XiaozhiHandler) WriteRespEvent(ctx context.Context, event any) error {
	if w.closed.Load() {
		return errors.New("write queue closed")
	}
	if len(w.writeQueue) > WriteQueueSize {
		fmt.Errorf("write queue is full, len: %d", len(w.writeQueue))
	}
	if ev, ok := xiaozhi.IsServerEvent(event); ok {
		if !strings.HasSuffix(string(ev.GetType()), ".delta") {
		}
		w.writeQueue <- event
		return nil
	}

	w.addOpusDuration()
	w.writeQueue <- event
	return nil
}

func (w *XiaozhiHandler) Done() <-chan struct{} {
	return w.ctx.Done()
}

func (w *XiaozhiHandler) logEvent(ctx context.Context, eventName string) {
	return
}
