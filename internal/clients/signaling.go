package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

type Session struct {
	ID           string
	WebSocketURL string
}

type SignalingRoutes struct {
	CreateSessionMethod string
	CreateSessionPath   string
	JoinSessionMethod   string
	JoinSessionPath     string
	LeaveSessionMethod  string
	LeaveSessionPath    string
	WebSocketPath       string
}

func DefaultSignalingRoutes() SignalingRoutes {
	return SignalingRoutes{
		CreateSessionMethod: http.MethodPost,
		CreateSessionPath:   "/signaling/sessions",
		JoinSessionMethod:   http.MethodPost,
		JoinSessionPath:     "/signaling/sessions/%s/join",
		LeaveSessionMethod:  http.MethodPost,
		LeaveSessionPath:    "/signaling/sessions/%s/leave",
		WebSocketPath:       "/signaling/ws",
	}
}

type SignalingEventType string

const (
	SignalingEventOffer               SignalingEventType = "offer"
	SignalingEventAnswer              SignalingEventType = "answer"
	SignalingEventCandidate           SignalingEventType = "candidate"
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
	requestURL, err := c.api.joinURL(c.routes.CreateSessionPath)
	if err != nil {
		return nil, err
	}

	request, err := c.newAuthorizedJSONRequest(c.routes.CreateSessionMethod, requestURL, nil)
	if err != nil {
		return nil, err
	}

	var response sessionResponse
	if err := c.api.doJSON(request, &response); err != nil {
		return nil, err
	}

	return resolveSession(response)
}

func (c *SignalingClient) JoinSession(sessionID string) (*Session, error) {
	requestURL, err := c.api.joinURL(fmt.Sprintf(c.routes.JoinSessionPath, url.PathEscape(sessionID)))
	if err != nil {
		return nil, err
	}

	request, err := c.newAuthorizedJSONRequest(c.routes.JoinSessionMethod, requestURL, nil)
	if err != nil {
		return nil, err
	}

	var response sessionResponse
	if err := c.api.doJSON(request, &response); err != nil {
		return nil, err
	}

	session, err := resolveSession(response)
	if err != nil {
		session = &Session{ID: sessionID}
	}
	if session.ID == "" {
		session.ID = sessionID
	}

	return session, nil
}

func (c *SignalingClient) LeaveSession(sessionID string) error {
	requestURL, err := c.api.joinURL(fmt.Sprintf(c.routes.LeaveSessionPath, url.PathEscape(sessionID)))
	if err != nil {
		return err
	}

	request, err := c.newAuthorizedJSONRequest(c.routes.LeaveSessionMethod, requestURL, nil)
	if err != nil {
		return err
	}

	return c.api.doJSON(request, nil)
}

func (c *SignalingClient) Connect(session *Session) error {
	if session == nil {
		return fmt.Errorf("client: session is required")
	}

	websocketURL := strings.TrimSpace(session.WebSocketURL)
	if websocketURL == "" {
		derived, err := c.api.joinWebSocketURL(c.routes.WebSocketPath)
		if err != nil {
			return err
		}

		parsed, err := url.Parse(derived)
		if err != nil {
			return err
		}

		query := parsed.Query()
		query.Set("sessionId", session.ID)
		parsed.RawQuery = query.Encode()
		websocketURL = parsed.String()
	}

	headers := c.authorizationHeaders()

	connection, _, err := c.dialer.Dial(websocketURL, headers)
	if err != nil {
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
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("client: signaling websocket is not connected")
	}

	return c.conn.WriteJSON(message)
}

func (c *SignalingClient) Receive() (SignalingMessage, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	c.connMu.Lock()
	connection := c.conn
	c.connMu.Unlock()

	if connection == nil {
		return SignalingMessage{}, fmt.Errorf("client: signaling websocket is not connected")
	}

	_, payload, err := connection.ReadMessage()
	if err != nil {
		return SignalingMessage{}, err
	}

	var message SignalingMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		return SignalingMessage{}, err
	}

	return message, nil
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

func (c *SignalingClient) newAuthorizedJSONRequest(method, requestURL string, body any) (*http.Request, error) {
	return c.api.newJSONRequest(method, requestURL, body, true)
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

type sessionResponse struct {
	SessionID    string `json:"sessionId"`
	ID           string `json:"id"`
	WebSocketURL string `json:"websocketUrl"`
	WSURL        string `json:"wsUrl"`
}

func resolveSession(response sessionResponse) (*Session, error) {
	sessionID := response.SessionID
	if sessionID == "" {
		sessionID = response.ID
	}

	if sessionID == "" {
		return nil, fmt.Errorf("client: session response does not contain a session ID")
	}

	websocketURL := response.WebSocketURL
	if websocketURL == "" {
		websocketURL = response.WSURL
	}

	return &Session{
		ID:           sessionID,
		WebSocketURL: websocketURL,
	}, nil
}

func normalizeSignalingRoutes(routes SignalingRoutes) SignalingRoutes {
	defaults := DefaultSignalingRoutes()

	if routes.CreateSessionMethod == "" {
		routes.CreateSessionMethod = defaults.CreateSessionMethod
	}
	if routes.CreateSessionPath == "" {
		routes.CreateSessionPath = defaults.CreateSessionPath
	}
	if routes.JoinSessionMethod == "" {
		routes.JoinSessionMethod = defaults.JoinSessionMethod
	}
	if routes.JoinSessionPath == "" {
		routes.JoinSessionPath = defaults.JoinSessionPath
	}
	if routes.LeaveSessionMethod == "" {
		routes.LeaveSessionMethod = defaults.LeaveSessionMethod
	}
	if routes.LeaveSessionPath == "" {
		routes.LeaveSessionPath = defaults.LeaveSessionPath
	}
	if routes.WebSocketPath == "" {
		routes.WebSocketPath = defaults.WebSocketPath
	}

	return routes
}
