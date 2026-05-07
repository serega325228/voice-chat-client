package service

import (
	"context"
	"fmt"
	"selfcord/internal/lib/audio"
	pipeline "selfcord/internal/pipeline"

	"github.com/pion/webrtc/v4"
)

type managedSessionPeer struct {
	webrtc       *WebRTCService
	sound        *SoundService
	captureGate  *audio.MuteableSampleSink
	pipelineStop context.CancelFunc
}

func NewManagedSessionPeer(webrtcConfig WebRTCConfig, soundConfig SoundConfig) (WebRTCPeer, error) {
	pipelineCtx, pipelineStop := context.WithCancel(context.Background())

	captureSink, captureSource := pipeline.New(
		pipelineCtx,
		pipeline.RNNoiseWorker,
	)
	captureGate := audio.NewMuteableSampleSink(captureSink)

	playbackSource, playbackSink := audio.NewSampleChannel(8)

	webrtcService, err := NewWebRTCService(webrtcConfig, captureSource, playbackSink)
	if err != nil {
		pipelineStop()
		return nil, err
	}

	soundService := NewSoundServiceWithConfig(soundConfig, playbackSource, captureGate)
	if err := soundService.Start(); err != nil {
		pipelineStop()
		_ = webrtcService.Close()
		return nil, fmt.Errorf("sound: start session audio: %w", err)
	}

	return &managedSessionPeer{
		webrtc:       webrtcService,
		sound:        soundService,
		captureGate:  captureGate,
		pipelineStop: pipelineStop,
	}, nil
}

func (p *managedSessionPeer) CreateOffer() (webrtc.SessionDescription, error) {
	return p.webrtc.CreateOffer()
}

func (p *managedSessionPeer) CreateAnswer(offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	return p.webrtc.CreateAnswer(offer)
}

func (p *managedSessionPeer) SetAnswer(answer webrtc.SessionDescription) error {
	return p.webrtc.SetAnswer(answer)
}

func (p *managedSessionPeer) AddRemoteCandidate(candidate webrtc.ICECandidateInit) error {
	return p.webrtc.AddRemoteCandidate(candidate)
}

func (p *managedSessionPeer) DrainLocalCandidates() []webrtc.ICECandidateInit {
	return p.webrtc.DrainLocalCandidates()
}

func (p *managedSessionPeer) SetMuted(muted bool) {
	p.captureGate.SetMuted(muted)
}

func (p *managedSessionPeer) IsMuted() bool {
	return p.captureGate.IsMuted()
}

func (p *managedSessionPeer) Close() error {
	if p.pipelineStop != nil {
		p.pipelineStop()
	}

	if p.sound != nil {
		if err := p.sound.Close(); err != nil {
			return err
		}
	}

	if p.webrtc != nil {
		if err := p.webrtc.Close(); err != nil {
			return err
		}
	}

	return nil
}
