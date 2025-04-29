package openai

import (
	"context"

	"github.com/xdimtech/go-xiaozhi/pkg/config"
	"github.com/xdimtech/go-xiaozhi/pkg/protocol/openai"
)

type ClientConfig struct {
	Format        string
	SampleRate    int
	Channels      int
	FrameDuration int
	FrameSize     int
}

type ApiSession struct {
	ctx          context.Context
	cancel       context.CancelFunc
	CliConfig    *ClientConfig
	defaultVoice string
	modelId      string
	ID           string
	Object       string
	RtSession    *openai.ServerSession
}

func NewApiSession(ctx context.Context, modelId string) *ApiSession {
	ctx, cancel := context.WithCancel(ctx)
	return &ApiSession{
		ctx:          ctx,
		cancel:       cancel,
		modelId:      modelId,
		defaultVoice: config.OpenAIConfig().Voice,
		Object:       openai.ObjectRealtimeSession,
	}
}

func (s *ApiSession) Update(sess *openai.ServerSession) {
	s.RtSession = sess
	s.ID = sess.ID
}

func (s *ApiSession) GetSessionId() string {
	return s.ID
}

func (s *ApiSession) Close() {
	if s.cancel != nil {
		s.cancel()
	}
}
