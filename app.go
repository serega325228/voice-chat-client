package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	clientpkg "selfcord/internal/clients"
	containerpkg "selfcord/internal/container"
	servicepkg "selfcord/internal/services"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/zalando/go-keyring"
)

type CallStateResult struct {
	SessionID string `json:"sessionId,omitempty"`
	IsMuted   bool   `json:"isMuted"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
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
		if isUnauthorizedRequest(err) {
			return BootstrapResult{
				AuthInfo: "Сессия истекла, войдите снова",
			}
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

func (a *App) CreateSession() (CallStateResult, error) {
	return a.withSessionRetry(func(signalingService *servicepkg.SignalingService) (CallStateResult, error) {
		session, err := signalingService.CreateSession()
		if err != nil {
			return CallStateResult{}, err
		}

		return CallStateResult{
			SessionID: session.ID,
			IsMuted:   signalingService.IsMuted(),
			Status:    "active",
			Message:   "Сессия создана и аудиоканал запущен",
		}, nil
	})
}

func (a *App) JoinSession(sessionID string) (CallStateResult, error) {
	normalizedSessionID := strings.TrimSpace(sessionID)

	return a.withSessionRetry(func(signalingService *servicepkg.SignalingService) (CallStateResult, error) {
		session, err := signalingService.JoinSession(normalizedSessionID)
		if err != nil {
			return CallStateResult{}, err
		}

		return CallStateResult{
			SessionID: session.ID,
			IsMuted:   signalingService.IsMuted(),
			Status:    "active",
			Message:   "Подключение к сессии выполнено",
		}, nil
	})
}

func (a *App) LeaveSession() (CallStateResult, error) {
	signalingService, err := a.signalingService()
	if err != nil {
		return CallStateResult{}, err
	}

	if err := signalingService.LeaveSession(); err != nil {
		return CallStateResult{}, err
	}

	return CallStateResult{
		Status:  "idle",
		IsMuted: false,
		Message: "Сессия завершена",
	}, nil
}

func (a *App) SetMicrophoneMuted(muted bool) (CallStateResult, error) {
	signalingService, err := a.signalingService()
	if err != nil {
		return CallStateResult{}, err
	}

	if err := signalingService.SetMuted(muted); err != nil {
		return CallStateResult{}, err
	}

	return CallStateResult{
		Status:  "active",
		IsMuted: signalingService.IsMuted(),
		Message: microphoneMessage(signalingService.IsMuted()),
	}, nil
}

func (a *App) signalingService() (*servicepkg.SignalingService, error) {
	signalingService, err := a.container.SignalingService()
	if err != nil {
		return nil, err
	}

	signalingService.SetEventHandler(a.emitCallEvent)

	return signalingService, nil
}

func (a *App) emitCallEvent(event servicepkg.SessionEvent) {
	if a.ctx == nil {
		return
	}

	wailsruntime.EventsEmit(a.ctx, "call:session", event)
}

func (a *App) withSessionRetry(action func(*servicepkg.SignalingService) (CallStateResult, error)) (CallStateResult, error) {
	signalingService, err := a.signalingService()
	if err != nil {
		return CallStateResult{}, err
	}

	result, err := action(signalingService)
	if err == nil || !isUnauthorizedRequest(err) {
		return result, err
	}

	if refreshErr := a.refreshAccessToken(); refreshErr != nil {
		return CallStateResult{}, refreshErr
	}

	signalingService, err = a.signalingService()
	if err != nil {
		return CallStateResult{}, err
	}

	return action(signalingService)
}

func (a *App) refreshAccessToken() error {
	authService, err := a.container.AuthService()
	if err != nil {
		return err
	}

	if err := authService.Refresh(); err != nil {
		if isUnauthorizedRequest(err) || errors.Is(err, keyring.ErrNotFound) {
			return fmt.Errorf("сессия истекла, войдите снова")
		}

		return err
	}

	return nil
}

func isUnauthorizedRequest(err error) bool {
	var requestErr *clientpkg.RequestError
	return errors.As(err, &requestErr) && requestErr.StatusCode == http.StatusUnauthorized
}

func microphoneMessage(muted bool) string {
	if muted {
		return "Микрофон выключен"
	}

	return "Микрофон включен"
}
