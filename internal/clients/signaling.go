package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

type Session struct {
	ID           string
	WebSocketURL string
}

var ErrSignalingAcknowledgement = errors.New("client: signaling acknowledgement")

type SignalingRoutes struct {
	WebSocketPath string
}

func DefaultSignalingRoutes() SignalingRoutes {
	return SignalingRoutes{
		WebSocketPath: "/api/ws/",
	}
}

type SignalingEventType string

const (
	SignalingEventOffer               SignalingEventType = "webrtc_offer"
	SignalingEventAnswer              SignalingEventType = "webrtc_answer"
	SignalingEventCandidate           SignalingEventType = "ice_candidate"
	SignalingEventRenegotiationNeeded SignalingEventType = "renegotiation_needed"
)

type SignalingMessage struct {
	Type      SignalingEventType         `json:"type"`
	SessionID string                     `json:"sessionId,omitempty"`
	Offer     *webrtc.SessionDescription `json:"offer,omitempty"`
	Answer    *webrtc.SessionDescription `json:"answer,omitempty"`
	Candidate *webrtc.ICECandidateInit   `json:"candidate,omitempty"`
}

type SignalingClient struct {
	api    *apiClient
	routes SignalingRoutes
	dialer *websocket.Dialer

	connMu sync.Mutex
	conn   *websocket.Conn
	readMu sync.Mutex
}

func NewSignalingClient(config ClientConfig, routes SignalingRoutes) (*SignalingClient, error) {
	api, err := newAPIClient(config)
	if err != nil {
		return nil, err
	}

	routes = normalizeSignalingRoutes(routes)

	return &SignalingClient{
		api:    api,
		routes: routes,
		dialer: websocket.DefaultDialer,
	}, nil
}

func (c *SignalingClient) CreateSession() (*Session, error) {
	if err := c.Connect(nil); err != nil {
		return nil, err
	}

	if err := c.writeEnvelope(signalingEnvelope{
		Type: signalingCommandCreateSession,
		Data: emptyCommandData{},
	}); err != nil {
		return nil, err
	}

	response, err := c.receiveEnvelope()
	if err != nil {
		return nil, err
	}

	if response.Status == signalingStatusError {
		return nil, response.asError()
	}
	if response.Status != signalingStatusSuccess {
		return nil, fmt.Errorf("client: create_session returned unexpected status %q", response.Status)
	}
	if response.SessionID == "" {
		return nil, fmt.Errorf("client: create_session response does not contain session_id")
	}

	return &Session{ID: response.SessionID}, nil
}

func (c *SignalingClient) JoinSession(sessionID string) (*Session, error) {
	if err := c.Connect(nil); err != nil {
		return nil, err
	}

	if err := c.writeEnvelope(signalingEnvelope{
		Type: signalingCommandJoinSession,
		Data: sessionCommandData{SessionID: sessionID},
	}); err != nil {
		return nil, err
	}

	response, err := c.receiveEnvelope()
	if err != nil {
		return nil, err
	}
	if response.Status == signalingStatusError {
		return nil, response.asError()
	}
	if response.Status != signalingStatusSuccess {
		return nil, fmt.Errorf("client: join_session returned unexpected status %q", response.Status)
	}

	return &Session{ID: sessionID}, nil
}

func (c *SignalingClient) LeaveSession(sessionID string) error {
	return c.Close()
}

func (c *SignalingClient) Connect(session *Session) error {
	websocketURL, err := c.resolveWebSocketURL(session)
	if err != nil {
		return err
	}

	headers := c.authorizationHeaders()

	connection, response, err := c.dialer.Dial(websocketURL, headers)
	if err != nil {
		if response != nil {
			defer response.Body.Close()

			message, readErr := io.ReadAll(io.LimitReader(response.Body, 4096))
			if readErr != nil {
				return &RequestError{StatusCode: response.StatusCode}
			}

			return &RequestError{
				StatusCode: response.StatusCode,
				Message:    strings.TrimSpace(string(message)),
			}
		}

		return err
	}

	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn != nil {
		c.conn.Close()
	}
	c.conn = connection

	return nil
}

func (c *SignalingClient) Send(message SignalingMessage) error {
	envelope, err := encodeSignalingMessage(message)
	if err != nil {
		return err
	}

	return c.writeEnvelope(envelope)
}

func (c *SignalingClient) Receive() (SignalingMessage, error) {
	envelope, err := c.receiveEnvelope()
	if err != nil {
		return SignalingMessage{}, err
	}

	if envelope.Status == signalingStatusError {
		return SignalingMessage{}, envelope.asError()
	}
	if envelope.Status == signalingStatusSuccess {
		return SignalingMessage{}, ErrSignalingAcknowledgement
	}

	return decodeSignalingMessage(envelope)
}

func (c *SignalingClient) Close() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn == nil {
		return nil
	}

	err := c.conn.Close()
	c.conn = nil

	return err
}

func (c *SignalingClient) authorizationHeaders() http.Header {
	headers := http.Header{}

	if token := c.authorizationToken(); token != "" {
		headers.Set("Authorization", token)
	}

	return headers
}

func (c *SignalingClient) authorizationToken() string {
	if c.api.accessTokenProvider == nil {
		return ""
	}

	token := strings.TrimSpace(c.api.accessTokenProvider())
	if token == "" {
		return ""
	}

	return "Bearer " + token
}

func (c *SignalingClient) resolveWebSocketURL(session *Session) (string, error) {
	if session != nil {
		if websocketURL := strings.TrimSpace(session.WebSocketURL); websocketURL != "" {
			return websocketURL, nil
		}
	}

	return c.api.joinWebSocketURL(c.routes.WebSocketPath)
}

func (c *SignalingClient) writeEnvelope(envelope signalingEnvelope) error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("client: signaling websocket is not connected")
	}

	return c.conn.WriteJSON(envelope)
}

func (c *SignalingClient) receiveEnvelope() (signalingEnvelope, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	c.connMu.Lock()
	connection := c.conn
	c.connMu.Unlock()

	if connection == nil {
		return signalingEnvelope{}, fmt.Errorf("client: signaling websocket is not connected")
	}

	_, payload, err := connection.ReadMessage()
	if err != nil {
		return signalingEnvelope{}, err
	}

	var envelope signalingEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return signalingEnvelope{}, err
	}

	return envelope, nil
}

func normalizeSignalingRoutes(routes SignalingRoutes) SignalingRoutes {
	defaults := DefaultSignalingRoutes()

	if routes.WebSocketPath == "" {
		routes.WebSocketPath = defaults.WebSocketPath
	}

	return routes
}

const (
	signalingCommandCreateSession = "create_session"
	signalingCommandJoinSession   = "join_session"
	signalingStatusSuccess        = "success"
	signalingStatusError          = "error"
)

type signalingEnvelope struct {
	Type      string `json:"type,omitempty"`
	Data      any    `json:"data,omitempty"`
	Status    string `json:"status,omitempty"`
	Error     string `json:"error,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

func (e signalingEnvelope) asError() error {
	if trimmed := strings.TrimSpace(e.Error); trimmed != "" {
		return fmt.Errorf(trimmed)
	}

	return fmt.Errorf("client: signaling request failed")
}

type emptyCommandData struct{}

type sessionCommandData struct {
	SessionID string `json:"session_id"`
}

type sessionDescriptionData struct {
	SessionID string `json:"session_id"`
	SDP       string `json:"sdp"`
}

type wireICECandidate struct {
	SessionID        string  `json:"session_id"`
	Candidate        string  `json:"candidate"`
	SDPMid           *string `json:"sdp_mid,omitempty"`
	SDPMLineIndex    *uint16 `json:"sdp_mline_index,omitempty"`
	UsernameFragment *string `json:"username_fragment,omitempty"`
}

func encodeSignalingMessage(message SignalingMessage) (signalingEnvelope, error) {
	switch message.Type {
	case SignalingEventOffer:
		if message.Offer == nil {
			return signalingEnvelope{}, fmt.Errorf("client: offer payload is required")
		}

		return signalingEnvelope{
			Type: string(SignalingEventOffer),
			Data: sessionDescriptionData{
				SessionID: message.SessionID,
				SDP:       message.Offer.SDP,
			},
		}, nil
	case SignalingEventAnswer:
		if message.Answer == nil {
			return signalingEnvelope{}, fmt.Errorf("client: answer payload is required")
		}

		return signalingEnvelope{
			Type: string(SignalingEventAnswer),
			Data: sessionDescriptionData{
				SessionID: message.SessionID,
				SDP:       message.Answer.SDP,
			},
		}, nil
	case SignalingEventCandidate:
		if message.Candidate == nil {
			return signalingEnvelope{}, fmt.Errorf("client: candidate payload is required")
		}

		return signalingEnvelope{
			Type: string(SignalingEventCandidate),
			Data: wireICECandidateFromPion(message.SessionID, *message.Candidate),
		}, nil
	default:
		return signalingEnvelope{}, fmt.Errorf("client: unsupported signaling event type %q", message.Type)
	}
}

func decodeSignalingMessage(envelope signalingEnvelope) (SignalingMessage, error) {
	switch SignalingEventType(envelope.Type) {
	case SignalingEventAnswer:
		payload, err := decodeSessionDescriptionData(envelope.Data)
		if err != nil {
			return SignalingMessage{}, err
		}

		return SignalingMessage{
			Type:      SignalingEventAnswer,
			SessionID: payload.SessionID,
			Answer: &webrtc.SessionDescription{
				Type: webrtc.SDPTypeAnswer,
				SDP:  payload.SDP,
			},
		}, nil
	case SignalingEventOffer, SignalingEventRenegotiationNeeded:
		payload, err := decodeSessionDescriptionData(envelope.Data)
		if err != nil {
			return SignalingMessage{}, err
		}

		return SignalingMessage{
			Type:      SignalingEventType(envelope.Type),
			SessionID: payload.SessionID,
			Offer: &webrtc.SessionDescription{
				Type: webrtc.SDPTypeOffer,
				SDP:  payload.SDP,
			},
		}, nil
	case SignalingEventCandidate:
		payload, err := decodeWireICECandidate(envelope.Data)
		if err != nil {
			return SignalingMessage{}, err
		}

		candidate := payload.toPion()
		return SignalingMessage{
			Type:      SignalingEventCandidate,
			SessionID: payload.SessionID,
			Candidate: &candidate,
		}, nil
	default:
		return SignalingMessage{}, fmt.Errorf("client: unsupported signaling event type %q", envelope.Type)
	}
}

func decodeSessionDescriptionData(raw any) (sessionDescriptionData, error) {
	payload, err := json.Marshal(raw)
	if err != nil {
		return sessionDescriptionData{}, err
	}

	var description sessionDescriptionData
	if err := json.Unmarshal(payload, &description); err != nil {
		return sessionDescriptionData{}, err
	}

	if description.SessionID == "" || description.SDP == "" {
		return sessionDescriptionData{}, fmt.Errorf("client: signaling message is missing session_id or sdp")
	}

	return description, nil
}

func decodeWireICECandidate(raw any) (wireICECandidate, error) {
	payload, err := json.Marshal(raw)
	if err != nil {
		return wireICECandidate{}, err
	}

	var candidate wireICECandidate
	if err := json.Unmarshal(payload, &candidate); err != nil {
		return wireICECandidate{}, err
	}

	if candidate.SessionID == "" || candidate.Candidate == "" {
		return wireICECandidate{}, fmt.Errorf("client: signaling candidate is missing session_id or candidate")
	}

	return candidate, nil
}

func wireICECandidateFromPion(sessionID string, candidate webrtc.ICECandidateInit) wireICECandidate {
	return wireICECandidate{
		SessionID:        sessionID,
		Candidate:        candidate.Candidate,
		SDPMid:           candidate.SDPMid,
		SDPMLineIndex:    candidate.SDPMLineIndex,
		UsernameFragment: candidate.UsernameFragment,
	}
}

func (c wireICECandidate) toPion() webrtc.ICECandidateInit {
	return webrtc.ICECandidateInit{
		Candidate:        c.Candidate,
		SDPMid:           c.SDPMid,
		SDPMLineIndex:    c.SDPMLineIndex,
		UsernameFragment: c.UsernameFragment,
	}
}
