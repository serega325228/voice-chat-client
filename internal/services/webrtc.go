package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"selfcord/internal/lib/audio"
	"strings"
	"sync"
	"time"

	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

const (
	webrtcFrameDuration  = 10 * time.Millisecond
	webrtcFrameSamples   = audio.DefaultOpusFrameSize
	pipelineFrameSamples = audio.DefaultOpusFrameSize
)

type WebRTCConfig struct {
	ICEServers []webrtc.ICEServer
}

type WebRTCService struct {
	peerConnection *webrtc.PeerConnection
	localTrack     *webrtc.TrackLocalStaticSample
	audioSource    audio.ChannelSampleSource
	playbackSink   audio.SampleSink

	ctx    context.Context
	cancel context.CancelFunc

	mu                      sync.Mutex
	localCandidates         []webrtc.ICECandidateInit
	pendingRemoteCandidates []webrtc.ICECandidateInit
	remoteDescriptionSet    bool
}

func NewWebRTCService(
	config WebRTCConfig,
	audioSource audio.ChannelSampleSource,
	playbackSink audio.SampleSink,
) (*WebRTCService, error) {
	mediaEngine := &webrtc.MediaEngine{}
	if err := registerAudioCodecs(mediaEngine); err != nil {
		return nil, fmt.Errorf("webrtc: register codecs: %w", err)
	}

	interceptorRegistry := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(mediaEngine, interceptorRegistry); err != nil {
		return nil, fmt.Errorf("webrtc: register interceptors: %w", err)
	}

	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithInterceptorRegistry(interceptorRegistry),
	)

	peerConnection, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: config.ICEServers,
	})
	if err != nil {
		return nil, fmt.Errorf("webrtc: create peer connection: %w", err)
	}

	localTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{
			MimeType:    webrtc.MimeTypeOpus,
			ClockRate:   audio.DefaultOpusSampleRate,
			Channels:    audio.DefaultOpusChannels,
			SDPFmtpLine: "minptime=10;useinbandfec=1",
		},
		"audio",
		"selfcord",
	)
	if err != nil {
		peerConnection.Close()
		return nil, fmt.Errorf("webrtc: create local track: %w", err)
	}

	sender, err := peerConnection.AddTrack(localTrack)
	if err != nil {
		peerConnection.Close()
		return nil, fmt.Errorf("webrtc: add local track: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	service := &WebRTCService{
		peerConnection: peerConnection,
		localTrack:     localTrack,
		audioSource:    audioSource,
		playbackSink:   playbackSink,
		ctx:            ctx,
		cancel:         cancel,
	}

	peerConnection.OnICECandidate(service.onICECandidate)
	peerConnection.OnTrack(service.onTrack)

	go service.readSenderRTCP(sender)
	go service.forwardPipelineToTrack()

	return service, nil
}

func (s *WebRTCService) CreateOffer() (webrtc.SessionDescription, error) {
	offer, err := s.peerConnection.CreateOffer(nil)
	if err != nil {
		return webrtc.SessionDescription{}, fmt.Errorf("webrtc: create offer: %w", err)
	}

	if err := s.peerConnection.SetLocalDescription(offer); err != nil {
		return webrtc.SessionDescription{}, fmt.Errorf("webrtc: set local description: %w", err)
	}

	return offer, nil
}

func (s *WebRTCService) CreateAnswer(offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	if offer.Type != webrtc.SDPTypeOffer {
		return webrtc.SessionDescription{}, fmt.Errorf("webrtc: expected offer, got %s", offer.Type.String())
	}

	if err := s.setRemoteDescription(offer); err != nil {
		return webrtc.SessionDescription{}, err
	}

	answer, err := s.peerConnection.CreateAnswer(nil)
	if err != nil {
		return webrtc.SessionDescription{}, fmt.Errorf("webrtc: create answer: %w", err)
	}

	if err := s.peerConnection.SetLocalDescription(answer); err != nil {
		return webrtc.SessionDescription{}, fmt.Errorf("webrtc: set local description: %w", err)
	}

	return answer, nil
}

func (s *WebRTCService) DrainLocalCandidates() []webrtc.ICECandidateInit {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.localCandidates) == 0 {
		return nil
	}

	candidates := append([]webrtc.ICECandidateInit(nil), s.localCandidates...)
	s.localCandidates = s.localCandidates[:0]

	return candidates
}

func (s *WebRTCService) SetAnswer(answer webrtc.SessionDescription) error {
	if answer.Type != webrtc.SDPTypeAnswer {
		return fmt.Errorf("webrtc: expected answer, got %s", answer.Type.String())
	}

	return s.setRemoteDescription(answer)
}

func (s *WebRTCService) AddRemoteCandidate(candidate webrtc.ICECandidateInit) error {
	s.mu.Lock()
	remoteDescriptionSet := s.remoteDescriptionSet
	if !remoteDescriptionSet {
		s.pendingRemoteCandidates = append(s.pendingRemoteCandidates, candidate)
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	if err := s.peerConnection.AddICECandidate(candidate); err != nil {
		return fmt.Errorf("webrtc: add remote candidate: %w", err)
	}

	return nil
}

func (s *WebRTCService) Close() error {
	s.cancel()

	if err := s.peerConnection.Close(); err != nil && !errors.Is(err, io.ErrClosedPipe) {
		return fmt.Errorf("webrtc: close peer connection: %w", err)
	}

	return nil
}

func (s *WebRTCService) onICECandidate(candidate *webrtc.ICECandidate) {
	if candidate == nil {
		return
	}

	s.mu.Lock()
	s.localCandidates = append(s.localCandidates, candidate.ToJSON())
	s.mu.Unlock()
}

func (s *WebRTCService) onTrack(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
	if track.Kind() != webrtc.RTPCodecTypeAudio {
		return
	}

	if !strings.EqualFold(track.Codec().MimeType, webrtc.MimeTypeOpus) {
		return
	}

	go s.forwardTrackToPlayback(track)
}

func (s *WebRTCService) readSenderRTCP(sender *webrtc.RTPSender) {
	buffer := make([]byte, 1500)

	for {
		if _, _, err := sender.Read(buffer); err != nil {
			return
		}
	}
}

func (s *WebRTCService) forwardPipelineToTrack() {
	if s.audioSource == nil {
		return
	}

	frame := make(audio.Samples, 0, pipelineFrameSamples)
	packetBuffer := make([]byte, audio.MaxOpusPacketSize)
	encoder, err := audio.NewOpusEncoder(audio.DefaultOpusConfig())
	if err != nil {
		return
	}

	for {
		select {
		case <-s.ctx.Done():
			return
		case samples, ok := <-s.audioSource:
			if !ok {
				return
			}

			for len(samples) > 0 {
				missing := pipelineFrameSamples - len(frame)
				if missing > len(samples) {
					missing = len(samples)
				}

				frame = append(frame, samples[:missing]...)
				samples = samples[missing:]

				if len(frame) < pipelineFrameSamples {
					continue
				}

				s.writeAudioFrame(encoder, packetBuffer, frame)
				frame = frame[:0]
			}
		}
	}
}

func (s *WebRTCService) writeAudioFrame(
	encoder *audio.OpusEncoder,
	packetBuffer []byte,
	frame audio.Samples,
) {
	if len(frame) != webrtcFrameSamples {
		return
	}

	packetSize, err := encoder.Encode(frame, packetBuffer)
	if err != nil || packetSize == 0 {
		return
	}

	packet := append([]byte(nil), packetBuffer[:packetSize]...)
	err = s.localTrack.WriteSample(media.Sample{
		Data:     packet,
		Duration: webrtcFrameDuration,
	})
	if err != nil && !errors.Is(err, io.ErrClosedPipe) {
		return
	}
}

func (s *WebRTCService) forwardTrackToPlayback(track *webrtc.TrackRemote) {
	if s.playbackSink == nil {
		return
	}

	decoder, err := audio.NewOpusDecoder(audio.DefaultOpusConfig())
	if err != nil {
		return
	}

	var expectedSequence uint16
	var hasExpectedSequence bool

	for {
		packet, _, err := track.ReadRTP()
		if err != nil {
			return
		}

		if len(packet.Payload) == 0 {
			continue
		}

		if hasExpectedSequence {
			diff := int(int16(packet.SequenceNumber - expectedSequence))
			if diff < 0 {
				continue
			}

			switch {
			case diff == 1:
				s.decodeAndWrite(decoder, packet.Payload, true)
			case diff > 1:
				for i := 0; i < diff-1; i++ {
					s.decodeAndWrite(decoder, nil, false)
				}
				s.decodeAndWrite(decoder, packet.Payload, true)
			}
		}

		s.decodeAndWrite(decoder, packet.Payload, false)
		expectedSequence = packet.SequenceNumber + 1
		hasExpectedSequence = true
	}
}

func registerAudioCodecs(mediaEngine *webrtc.MediaEngine) error {
	return mediaEngine.RegisterCodec(
		webrtc.RTPCodecParameters{
			RTPCodecCapability: webrtc.RTPCodecCapability{
				MimeType:    webrtc.MimeTypeOpus,
				ClockRate:   audio.DefaultOpusSampleRate,
				Channels:    audio.DefaultOpusChannels,
				SDPFmtpLine: "minptime=10;useinbandfec=1",
			},
			PayloadType: 111,
		},
		webrtc.RTPCodecTypeAudio,
	)
}

func (s *WebRTCService) decodeAndWrite(
	decoder *audio.OpusDecoder,
	packet []byte,
	useFEC bool,
) {
	var (
		samples audio.Samples
		err     error
	)

	if useFEC {
		samples, err = decoder.DecodeFEC(packet)
	} else if packet == nil {
		samples, err = decoder.DecodePLC()
	} else {
		samples, err = decoder.Decode(packet)
	}
	if err != nil || len(samples) == 0 {
		return
	}

	s.playbackSink.WriteSamples(samples)
}

func (s *WebRTCService) setRemoteDescription(description webrtc.SessionDescription) error {
	if err := s.peerConnection.SetRemoteDescription(description); err != nil {
		return fmt.Errorf("webrtc: set remote description: %w", err)
	}

	s.mu.Lock()
	pending := append([]webrtc.ICECandidateInit(nil), s.pendingRemoteCandidates...)
	s.pendingRemoteCandidates = nil
	s.remoteDescriptionSet = true
	s.mu.Unlock()

	for _, candidate := range pending {
		if err := s.peerConnection.AddICECandidate(candidate); err != nil {
			return fmt.Errorf("webrtc: add pending remote candidate: %w", err)
		}
	}

	return nil
}
