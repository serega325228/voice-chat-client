package main

import (
	"context"
	"errors"
	"strings"
	containerpkg "voice-chat-client/internal/container"

	"github.com/zalando/go-keyring"
)

type SessionResult struct {
	ID string `json:"id"`
}

type App struct {
	ctx       context.Context
	container *containerpkg.Container
}

func NewApp(
	container *containerpkg.Container,
) *App {
	return &App{
		container: container,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

type BootstrapResult struct {
	IsAuthenticated bool   `json:"isAuthenticated"`
	AuthError       string `json:"authError,omitempty"`
	AuthInfo        string `json:"authInfo,omitempty"`
}

func (a *App) Bootstrap() BootstrapResult {
	authService, err := a.container.AuthService()
	if err != nil {
		return BootstrapResult{
			AuthError: err.Error(),
		}
	}

	if err := authService.Refresh(); err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return BootstrapResult{}
		}

		return BootstrapResult{
			AuthError: "Не удалось восстановить сессию",
		}
	}

	return BootstrapResult{
		IsAuthenticated: true,
		AuthInfo:        "Сессия восстановлена",
	}
}

func (a *App) Login(email, password string) error {
	authService, err := a.container.AuthService()
	if err != nil {
		return err
	}

	return authService.Login(strings.TrimSpace(email), password)
}

func (a *App) Register(username, email, password string) error {
	authService, err := a.container.AuthService()
	if err != nil {
		return err
	}

	return authService.Register(strings.TrimSpace(username), strings.TrimSpace(email), password)
}

func (a *App) CreateSession() (SessionResult, error) {
	signalingService, err := a.container.SignalingService()
	if err != nil {
		return SessionResult{}, err
	}

	session, err := signalingService.CreateSession()
	if err != nil {
		return SessionResult{}, err
	}

	return SessionResult{
		ID: session.ID,
	}, nil
}

func (a *App) JoinSession(sessionID string) (SessionResult, error) {
	signalingService, err := a.container.SignalingService()
	if err != nil {
		return SessionResult{}, err
	}

	session, err := signalingService.JoinSession(strings.TrimSpace(sessionID))
	if err != nil {
		return SessionResult{}, err
	}

	return SessionResult{
		ID: session.ID,
	}, nil
}

func (a *App) LeaveSession() error {
	signalingService, err := a.container.SignalingService()
	if err != nil {
		return err
	}

	return signalingService.LeaveSession()
}
