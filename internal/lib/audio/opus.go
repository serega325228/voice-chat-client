package audio

import "github.com/thesyncim/gopus"

const (
	DefaultOpusSampleRate = 48000
	DefaultOpusChannels   = 1
	DefaultOpusBitrate    = 32000
	DefaultOpusFrameSize  = 480
	MaxOpusPacketSize     = 4000
)

type OpusConfig struct {
	SampleRate int
	Channels   int
	Bitrate    int
	FrameSize  int
	EnableFEC  bool
}

func DefaultOpusConfig() OpusConfig {
	return OpusConfig{
		SampleRate: DefaultOpusSampleRate,
		Channels:   DefaultOpusChannels,
		Bitrate:    DefaultOpusBitrate,
		FrameSize:  DefaultOpusFrameSize,
		EnableFEC:  true,
	}
}

type OpusEncoder struct {
	encoder *gopus.Encoder
	config  OpusConfig
}

func NewOpusEncoder(config OpusConfig) (*OpusEncoder, error) {
	config = normalizeOpusConfig(config)

	encoder, err := gopus.NewEncoder(gopus.EncoderConfig{
		SampleRate:  config.SampleRate,
		Channels:    config.Channels,
		Application: gopus.ApplicationVoIP,
	})
	if err != nil {
		return nil, err
	}

	if err := encoder.SetBitrate(config.Bitrate); err != nil {
		return nil, err
	}
	if err := encoder.SetFrameSize(config.FrameSize); err != nil {
		return nil, err
	}
	if err := encoder.SetExpertFrameDuration(gopus.ExpertFrameDuration10Ms); err != nil {
		return nil, err
	}

	encoder.SetFEC(config.EnableFEC)

	return &OpusEncoder{
		encoder: encoder,
		config:  config,
	}, nil
}

func (e *OpusEncoder) Encode(samples Samples, packetBuffer []byte) (int, error) {
	return e.encoder.Encode(samples, packetBuffer)
}

func (e *OpusEncoder) Config() OpusConfig {
	return e.config
}

type OpusDecoder struct {
	decoder   *gopus.Decoder
	config    OpusConfig
	pcmBuffer []float32
}

func NewOpusDecoder(config OpusConfig) (*OpusDecoder, error) {
	config = normalizeOpusConfig(config)

	decoderConfig := gopus.DefaultDecoderConfig(config.SampleRate, config.Channels)
	decoder, err := gopus.NewDecoder(decoderConfig)
	if err != nil {
		return nil, err
	}

	return &OpusDecoder{
		decoder:   decoder,
		config:    config,
		pcmBuffer: make([]float32, decoderConfig.MaxPacketSamples*decoderConfig.Channels),
	}, nil
}

func (d *OpusDecoder) Decode(packet []byte) (Samples, error) {
	sampleCount, err := d.decoder.Decode(packet, d.pcmBuffer)
	if err != nil || sampleCount == 0 {
		return nil, err
	}

	return append(Samples(nil), d.pcmBuffer[:sampleCount*d.config.Channels]...), nil
}

func (d *OpusDecoder) DecodeFEC(packet []byte) (Samples, error) {
	sampleCount, err := d.decoder.DecodeWithFEC(packet, d.pcmBuffer, true)
	if err != nil || sampleCount == 0 {
		return nil, err
	}

	return append(Samples(nil), d.pcmBuffer[:sampleCount*d.config.Channels]...), nil
}

func (d *OpusDecoder) DecodePLC() (Samples, error) {
	return d.Decode(nil)
}

func (d *OpusDecoder) Config() OpusConfig {
	return d.config
}

func normalizeOpusConfig(config OpusConfig) OpusConfig {
	defaults := DefaultOpusConfig()

	if config.SampleRate == 0 {
		config.SampleRate = defaults.SampleRate
	}
	if config.Channels == 0 {
		config.Channels = defaults.Channels
	}
	if config.Bitrate == 0 {
		config.Bitrate = defaults.Bitrate
	}
	if config.FrameSize == 0 {
		config.FrameSize = defaults.FrameSize
	}

	return config
}
