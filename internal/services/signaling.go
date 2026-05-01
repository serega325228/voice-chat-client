package service

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"
	client "voice-chat-client/internal/clients"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

const localCandidateFlushInterval = 100 * time.Millisecond

type WebRTCPeer interface {
	CreateOffer() (webrtc.SessionDescription, error)
	CreateAnswer(offer webrtc.SessionDescription) (webrtc.SessionDescription, error)
	SetAnswer(answer webrtc.SessionDescription) error
	AddRemoteCandidate(candidate webrtc.ICECandidateInit) error
	DrainLocalCandidates() []webrtc.ICECandidateInit
	SetMuted(muted bool)
	IsMuted() bool
	Close() error
}

type WebRTCPeerFactory func() (WebRTCPeer, error)

type SessionEventKind string

const (
	SessionEventStarted     SessionEventKind = "started"
	SessionEventLeft        SessionEventKind = "left"
	SessionEventFailed      SessionEventKind = "failed"
	SessionEventMuteChanged SessionEventKind = "mute_changed"
)

type SessionEvent struct {
	Kind      SessionEventKind `json:"kind"`
	SessionID string           `json:"sessionId,omitempty"`
	IsMuted   bool             `json:"isMuted"`
	Message   string           `json:"message,omitempty"`
}

type SessionEventHandler func(SessionEvent)

type SignalingService struct {
	client        *client.SignalingClient
	newWebRTCPeer WebRTCPeerFactory

	mu         sync.Mutex
	session    *client.Session
	peer       WebRTCPeer
	loopCancel context.CancelFunc
	onEvent    SessionEventHandler
}

func NewSignalingService(signalingClient *client.SignalingClient, newWebRTCPeer WebRTCPeerFactory) *SignalingService {
	return &SignalingService{
		client:        signalingClient,
		newWebRTCPeer: newWebRTCPeer,
	}
}

func (s *SignalingService) CreateSession() (*client.Session, error) {
	if err := s.closeCurrentSession(); err != nil {
		return nil, err
	}

	session, err := s.client.CreateSession()
	if err != nil {
		return nil, err
	}

	if err := s.startSession(session, true); err != nil {
		_ = s.closeCurrentSession()
		return nil, err
	}

	return session, nil
}

func (s *SignalingService) JoinSession(sessionID string) (*client.Session, error) {
	if err := s.closeCurrentSession(); err != nil {
		return nil, err
	}

	session, err := s.client.JoinSession(sessionID)
	if err != nil {
		return nil, err
	}

	if err := s.startSession(session, false); err != nil {
		_ = s.closeCurrentSession()
		return nil, err
	}

	return session, nil
}

func (s *SignalingService) LeaveSession() error {
	s.mu.Lock()
	sessionID := ""
	if s.session != nil {
		sessionID = s.session.ID
	}
	s.mu.Unlock()

	if err := s.closeCurrentSession(); err != nil {
		return err
	}

	if sessionID != "" {
		s.emitEvent(SessionEvent{
			Kind:      SessionEventLeft,
			SessionID: sessionID,
		})
	}

	return nil
}

func (s *SignalingService) startSession(session *client.Session, sendOffer bool) error {
	if s.newWebRTCPeer == nil {
		return errors.New("signaling: webrtc peer factory is not configured")
	}

	peer, err := s.newWebRTCPeer()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())

	s.mu.Lock()
	s.session = session
	s.peer = peer
	s.loopCancel = cancel
	s.mu.Unlock()

	go s.forwardLocalCandidates(ctx, session.ID, peer)
	go s.consumeSignalingMessages(ctx, session.ID, peer)

	if sendOffer {
		offer, err := peer.CreateOffer()
		if err != nil {
			cancel()
			s.client.Close()
			peer.Close()
			return err
		}

		if err := s.client.Send(client.SignalingMessage{
			Type:      client.SignalingEventOffer,
			SessionID: session.ID,
			Offer:     &offer,
		}); err != nil {
			cancel()
			s.client.Close()
			peer.Close()
			return err
		}
	}

	s.flushLocalCandidates(session.ID, peer)
	s.emitEvent(SessionEvent{
		Kind:      SessionEventStarted,
		SessionID: session.ID,
		IsMuted:   peer.IsMuted(),
		Message:   "Аудиоканал и signaling запущены",
	})

	return nil
}

func (s *SignalingService) SetEventHandler(handler SessionEventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.onEvent = handler
}

func (s *SignalingService) SetMuted(muted bool) error {
	s.mu.Lock()
	peer := s.peer
	sessionID := ""
	if s.session != nil {
		sessionID = s.session.ID
	}
	s.mu.Unlock()

	if peer == nil {
		return errors.New("signaling: no active session")
	}

	peer.SetMuted(muted)
	s.emitEvent(SessionEvent{
		Kind:      SessionEventMuteChanged,
		SessionID: sessionID,
		IsMuted:   peer.IsMuted(),
		Message:   mutedStateMessage(peer.IsMuted()),
	})

	return nil
}

func (s *SignalingService) IsMuted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.peer == nil {
		return false
	}

	return s.peer.IsMuted()
}

func (s *SignalingService) forwardLocalCandidates(ctx context.Context, sessionID string, peer WebRTCPeer) {
	ticker := time.NewTicker(localCandidateFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.flushLocalCandidates(sessionID, peer)
		}
	}
}

func (s *SignalingService) consumeSignalingMessages(ctx context.Context, sessionID string, peer WebRTCPeer) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		message, err := s.client.Receive()
		if err != nil {
			if errors.Is(err, client.ErrSignalingAcknowledgement) {
				continue
			}
			if errors.Is(err, io.EOF) || websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return
			}
			s.reportAsyncError(err)
			return
		}

		if message.SessionID != "" && message.SessionID != sessionID {
			continue
		}

		s.handleSignalingMessage(sessionID, peer, message)
	}
}

func (s *SignalingService) handleSignalingMessage(sessionID string, peer WebRTCPeer, message client.SignalingMessage) {
	switch message.Type {
	case client.SignalingEventCandidate:
		if message.Candidate != nil {
			if err := peer.AddRemoteCandidate(*message.Candidate); err != nil {
				s.reportAsyncError(err)
			}
		}
	case client.SignalingEventAnswer:
		if message.Answer != nil {
			if err := peer.SetAnswer(*message.Answer); err != nil {
				s.reportAsyncError(err)
			}
		}
	case client.SignalingEventOffer, client.SignalingEventRenegotiationNeeded:
		if message.Offer == nil {
			return
		}

		answer, err := peer.CreateAnswer(*message.Offer)
		if err != nil {
			s.reportAsyncError(err)
			return
		}

		if err := s.client.Send(client.SignalingMessage{
			Type:      client.SignalingEventAnswer,
			SessionID: sessionID,
			Answer:    &answer,
		}); err != nil {
			s.reportAsyncError(err)
			return
		}
		s.flushLocalCandidates(sessionID, peer)
	}
}

func (s *SignalingService) flushLocalCandidates(sessionID string, peer WebRTCPeer) {
	for _, candidate := range peer.DrainLocalCandidates() {
		candidate := candidate
		if err := s.client.Send(client.SignalingMessage{
			Type:      client.SignalingEventCandidate,
			SessionID: sessionID,
			Candidate: &candidate,
		}); err != nil {
			s.reportAsyncError(err)
			return
		}
	}
}

func (s *SignalingService) closeCurrentSession() error {
	s.mu.Lock()
	peer := s.peer
	cancel := s.loopCancel
	s.session = nil
	s.peer = nil
	s.loopCancel = nil
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	if err := s.client.Close(); err != nil {
		return err
	}

	if peer != nil {
		if err := peer.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (s *SignalingService) reportAsyncError(err error) {
	if err == nil {
		return
	}

	s.mu.Lock()
	active := s.session != nil
	sessionID := ""
	if s.session != nil {
		sessionID = s.session.ID
	}
	s.mu.Unlock()

	if !active {
		return
	}

	_ = s.closeCurrentSession()
	s.emitEvent(SessionEvent{
		Kind:      SessionEventFailed,
		SessionID: sessionID,
		Message:   err.Error(),
	})
}

func (s *SignalingService) emitEvent(event SessionEvent) {
	s.mu.Lock()
	handler := s.onEvent
	s.mu.Unlock()

	if handler != nil {
		handler(event)
	}
}

func mutedStateMessage(muted bool) string {
	if muted {
		return "Микрофон выключен"
	}

	return "Микрофон включен"
}
