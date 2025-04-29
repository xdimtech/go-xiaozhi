package audio

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"gopkg.in/hraban/opus.v2"
)

const (
	DeviceOpusRate24k = 24000
	DeviceOpusRate48k = 48000
	DefaultDownPcmSR  = 24000
	DefaultUpPcmSR    = 24000
)

type AudioGainConfig struct {
	DefaultGain    float32
	MinGain        float32
	MaxGain        float32
	OpusGain       float32
	AutoGainEnable bool
}

func DefaultAudioGainConfig() AudioGainConfig {
	return AudioGainConfig{
		DefaultGain:    3.0,
		MinGain:        0.1,
		MaxGain:        10.0,
		OpusGain:       3.0,
		AutoGainEnable: false,
	}
}

type Converter struct {
	SampleRate     int
	DownSampleRate int
	Channels       int
	FrameSize      int
	FrameDuration  int
	DownDuration   int
	upReSampler    ResampleOperator
	downReSampler  ResampleOperator
	delta          []int16
	Encoder        *opus.Encoder
	Decoder        *opus.Decoder
	cb             Callback
	gainConfig     AudioGainConfig
}

type Callback func(ctx context.Context, data any) error

func NewConverter(sampleRate, channels, frameDuration, frameSize int, cb Callback) *Converter {
	upSampler, err := NewGoResampler(channels, sampleRate, DefaultUpPcmSR)
	if err != nil {
		panic(err)
	}
	downSampler, err := NewGoResampler(channels, DefaultDownPcmSR, DeviceOpusRate24k)
	if err != nil {
		panic(err)
	}
	enc, err := opus.NewEncoder(DeviceOpusRate24k, 1, opus.Application(opus.AppVoIP))
	if err != nil {
		panic(err)
	}
	dec, err := opus.NewDecoder(DeviceOpusRate24k, 1)
	if err != nil {
		panic(err)
	}
	return &Converter{
		SampleRate:     sampleRate,
		DownSampleRate: DeviceOpusRate24k,
		Channels:       channels,
		FrameDuration:  frameDuration,
		DownDuration:   frameDuration,
		FrameSize:      frameSize,
		upReSampler:    upSampler,
		downReSampler:  downSampler,
		Encoder:        enc,
		Decoder:        dec,
		cb:             cb,
	}
}

func (c *Converter) SetGainConfig(config AudioGainConfig) error {
	if config.MinGain < 0 || config.MaxGain > 20 {
		return fmt.Errorf("invalid gain range: min=%v, max=%v", config.MinGain, config.MaxGain)
	}
	c.gainConfig = config
	return nil
}

func (c *Converter) OpusToPcmBase64(opusData []byte) (string, error) {
	if len(opusData) == 0 {
		return "", errors.New("empty opus data")
	}
	// 解码为pcm
	pcmData, err := c.opus2pcm(opusData)
	if err != nil {
		return "", errors.New("opus decode failed, err: " + err.Error())
	}
	// 重采样为24k
	resampledData, err := c.upReSampler.Handle(pcmData)
	if err != nil {
		return "", errors.New("upReSampler handle failed, err: " + err.Error())
	}
	// 编码为base64
	encodedData := base64.StdEncoding.EncodeToString(resampledData)
	return encodedData, nil
}

func (c *Converter) opus2pcm(audioData []byte) ([]byte, error) {
	if len(audioData) == 0 {
		return nil, nil
	}
	pcm := make([]int16, 4096)
	dec, err := opus.NewDecoder(c.SampleRate, 1)
	if err != nil {
		return nil, err
	}
	size, err := dec.Decode(audioData, pcm)
	if err != nil {
		return nil, err
	}
	pcm = pcm[:size]
	return c.int16ToBytes(c.gain(pcm, 3)), nil
}

func (c *Converter) int16ToBytes(s []int16) []byte {
	var buf bytes.Buffer
	for _, v := range s {
		err := binary.Write(&buf, binary.LittleEndian, v)
		if err != nil {
			return nil
		}
	}
	return buf.Bytes()
}

func (c *Converter) gain(input []int16, f float32) []int16 {
	output := make([]int16, len(input))
	for i, sample := range input {
		adjustedSample := float32(sample) * f
		if adjustedSample > math.MaxInt16 {
			output[i] = math.MaxInt16
		} else if adjustedSample < math.MinInt16 {
			output[i] = math.MinInt16
		} else {
			output[i] = int16(adjustedSample)
		}
	}
	return output
}

func (c *Converter) ResolvePCM(base64Str string) error {
	delta, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return errors.New("base64 decode failed, err: " + err.Error())
	}
	c.parseFrames(delta)
	return nil
}

func (c *Converter) parseFrames(audioDelta []byte) {
	c.delta = append(c.delta, c.gain(c.bytesToInt16(audioDelta), 8)...)
	chunk := c.DownDuration * c.DownSampleRate / 1000

	var rest []int16
	if len(c.delta)%chunk != 0 {
		rest = c.delta[len(c.delta)-len(c.delta)%chunk:]
		c.delta = c.delta[:len(c.delta)-len(c.delta)%chunk]
	}

	for i := 0; i < len(c.delta); i += chunk {
		data := make([]byte, 2048)
		n, err := c.Encoder.Encode(c.delta[i:i+chunk], data)
		if err != nil {
			fmt.Println("Error encoding Opus frame:", err)
		}
		c.cb(nil, data[:n])
	}
	c.delta = rest
}

func (c *Converter) bytesToInt16(b []byte) []int16 {
	buf := bytes.NewReader(b)
	var result []int16
	for {
		var num int16
		err := binary.Read(buf, binary.LittleEndian, &num)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil
		}
		result = append(result, num)
	}
	return result
}

func (c *Converter) pcm2opus(audioData []byte) ([][]byte, error) {
	if len(audioData) == 0 || len(audioData)%2 != 0 {
		return nil, errors.New("invalid pcm data")
	}
	enc, err := opus.NewEncoder(c.DownSampleRate, c.Channels, opus.Application(opus.AppVoIP))
	if err != nil {
		return nil, fmt.Errorf("failed to create encoder: %w", err)
	}
	samples := make([]int16, len(audioData)/2)
	for i := 0; i < len(audioData); i += 2 {
		samples[i/2] = int16(binary.LittleEndian.Uint16(audioData[i:]))
	}

	numFrames := (len(samples) + c.FrameSize - 1) / c.FrameSize
	opusFrames := make([][]byte, 0, numFrames)
	maxOpusFrameSize := 1275

	for i := 0; i < numFrames; i++ {
		start := i * c.FrameSize
		end := start + c.FrameSize

		var frameToEncode []int16
		if end >= len(samples) {
			frameToEncode = make([]int16, c.FrameSize)
			copy(frameToEncode, samples[start:])
		} else {
			frameToEncode = samples[start:end]
		}

		frameData := make([]byte, maxOpusFrameSize)
		n, err := enc.Encode(frameToEncode, frameData)
		if err != nil {
			return nil, fmt.Errorf("failed to encode frame %d: %w", i, err)
		}

		if n > 0 {
			opusFrames = append(opusFrames, frameData[:n])
		}
	}

	return opusFrames, nil
}

func (c *Converter) Pcm16encode(data []byte) []int16 {
	var pcm []int16
	buf := bytes.NewBuffer(data)
	for {
		var sample int16
		err := binary.Read(buf, binary.LittleEndian, &sample)
		if err != nil {
			break
		}
		pcm = append(pcm, sample)
	}
	return pcm
}

func (c *Converter) Pcm16decode(pcm []int16) []byte {
	buf := new(bytes.Buffer)
	for _, sample := range pcm {
		_ = binary.Write(buf, binary.LittleEndian, sample)
	}
	return buf.Bytes()
}
