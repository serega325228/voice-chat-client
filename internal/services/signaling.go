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
	Close() error
}

type WebRTCPeerFactory func() (WebRTCPeer, error)

type SignalingService struct {
	client        *client.SignalingClient
	newWebRTCPeer WebRTCPeerFactory

	mu         sync.Mutex
	session    *client.Session
	peer       WebRTCPeer
	loopCancel context.CancelFunc
}

func NewSignalingService(signalingClient *client.SignalingClient, newWebRTCPeer WebRTCPeerFactory) *SignalingService {
	return &SignalingService{
		client:        signalingClient,
		newWebRTCPeer: newWebRTCPeer,
	}
}

func (s *SignalingService) CreateSession() (*client.Session, error) {
	if err := s.closeCurrentSession(true); err != nil {
		return nil, err
	}

	session, err := s.client.CreateSession()
	if err != nil {
		return nil, err
	}

	s.storeSession(session)

	return session, nil
}

func (s *SignalingService) JoinSession(sessionID string) (*client.Session, error) {
	if err := s.closeCurrentSession(true); err != nil {
		return nil, err
	}

	session, err := s.client.JoinSession(sessionID)
	if err != nil {
		return nil, err
	}

	s.storeSession(session)

	return session, nil
}

func (s *SignalingService) LeaveSession() error {
	return s.closeCurrentSession(true)
}

func (s *SignalingService) startSession(session *client.Session, sendOffer bool) error {
	if s.newWebRTCPeer == nil {
		return errors.New("signaling: webrtc peer factory is not configured")
	}

	peer, err := s.newWebRTCPeer()
	if err != nil {
		return err
	}

	if err := s.client.Connect(session); err != nil {
		peer.Close()
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

	return nil
}

func (s *SignalingService) storeSession(session *client.Session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.session = session
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
			if errors.Is(err, io.EOF) || websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return
			}
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
			peer.AddRemoteCandidate(*message.Candidate)
		}
	case client.SignalingEventAnswer:
		if message.Answer != nil {
			peer.SetAnswer(*message.Answer)
		}
	case client.SignalingEventOffer, client.SignalingEventRenegotiationNeeded:
		if message.Offer == nil {
			return
		}

		answer, err := peer.CreateAnswer(*message.Offer)
		if err != nil {
			return
		}

		s.client.Send(client.SignalingMessage{
			Type:      client.SignalingEventAnswer,
			SessionID: sessionID,
			Answer:    &answer,
		})
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
			return
		}
	}
}

func (s *SignalingService) closeCurrentSession(notifyServer bool) error {
	s.mu.Lock()
	session := s.session
	peer := s.peer
	cancel := s.loopCancel
	s.session = nil
	s.peer = nil
	s.loopCancel = nil
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	var result error

	if err := s.client.Close(); err != nil {
		result = err
	}

	if notifyServer && session != nil {
		if err := s.client.LeaveSession(session.ID); err != nil && result == nil {
			result = err
		}
	}

	if peer != nil {
		if err := peer.Close(); err != nil && result == nil {
			result = err
		}
	}

	return result
}
