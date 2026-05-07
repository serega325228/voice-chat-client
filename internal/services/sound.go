package service

import (
	"fmt"
	"selfcord/internal/lib/audio"

	"github.com/gen2brain/malgo"
)

const (
	defaultSampleRate uint32 = 48000
	defaultChannels   uint32 = 1
)

type SoundConfig struct {
	SampleRate uint32
	Channels   uint32
}

func DefaultSoundConfig() SoundConfig {
	return SoundConfig{
		SampleRate: defaultSampleRate,
		Channels:   defaultChannels,
	}
}

type SoundService struct {
	config         SoundConfig
	playbackSource audio.SampleSource
	captureSink    audio.SampleSink

	ctx    *malgo.AllocatedContext
	device *malgo.Device
}

func NewSoundService(playbackSource audio.SampleSource, captureSink audio.SampleSink) *SoundService {
	return &SoundService{
		config:         DefaultSoundConfig(),
		playbackSource: wrapPlaybackSource(playbackSource),
		captureSink:    captureSink,
	}
}

func NewSoundServiceWithConfig(
	config SoundConfig,
	playbackSource audio.SampleSource,
	captureSink audio.SampleSink,
) *SoundService {
	if config.SampleRate == 0 {
		config.SampleRate = defaultSampleRate
	}
	if config.Channels == 0 {
		config.Channels = defaultChannels
	}

	return &SoundService{
		config:         config,
		playbackSource: wrapPlaybackSource(playbackSource),
		captureSink:    captureSink,
	}
}

func wrapPlaybackSource(playbackSource audio.SampleSource) audio.SampleSource {
	channelSource, ok := playbackSource.(audio.ChannelSampleSource)
	if !ok {
		return playbackSource
	}

	return audio.NewBufferedSampleSource(channelSource)
}

func (s *SoundService) Start() error {
	if s.device != nil {
		return nil
	}

	if s.config.SampleRate == 0 {
		return fmt.Errorf("sound: sample rate must be greater than zero")
	}
	if s.config.Channels == 0 {
		return fmt.Errorf("sound: channels must be greater than zero")
	}

	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return fmt.Errorf("sound: init context: %w", err)
	}

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Duplex)
	deviceConfig.Capture.Format = malgo.FormatF32
	deviceConfig.Capture.Channels = s.config.Channels
	deviceConfig.Playback.Format = malgo.FormatF32
	deviceConfig.Playback.Channels = s.config.Channels
	deviceConfig.SampleRate = s.config.SampleRate

	device, err := malgo.InitDevice(ctx.Context, deviceConfig, malgo.DeviceCallbacks{
		Data: s.onData,
	})
	if err != nil {
		ctx.Free()
		return fmt.Errorf("sound: init device: %w", err)
	}

	if err := device.Start(); err != nil {
		device.Uninit()
		ctx.Free()
		return fmt.Errorf("sound: start device: %w", err)
	}

	s.ctx = ctx
	s.device = device

	return nil
}

func (s *SoundService) Close() error {
	var stopErr error

	if s.device != nil {
		if s.device.IsStarted() {
			stopErr = s.device.Stop()
		}
		s.device.Uninit()
		s.device = nil
	}

	if s.ctx != nil {
		s.ctx.Free()
		s.ctx = nil
	}

	if stopErr != nil {
		return fmt.Errorf("sound: stop device: %w", stopErr)
	}

	return nil
}

func (s *SoundService) onData(output, input []byte, _ uint32) {
	if len(input) > 0 && s.captureSink != nil {
		s.captureSink.WriteSamples(audio.BytesToSamples(input))
	}

	if len(output) == 0 {
		return
	}

	clear(output)

	if s.playbackSource == nil {
		return
	}

	sampleCount := len(output) / audio.BytesPerSample
	samples := s.playbackSource.ReadSamples(sampleCount)
	audio.WriteSamplesToBytes(output, samples)
}
