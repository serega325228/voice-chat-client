package service

import (
	"errors"
	"net/http"
	client "selfcord/internal/clients"
)

type TokenStorage interface {
	Save(accessToken, refreshToken string) error
	GetAccess() string
	GetRefresh() (string, error)
	Clear() error
}

type AuthClient interface {
	Register(username, email, password string) (string, string, error)
	Login(email, password string) (string, string, error)
	Refresh(refresh string) (string, string, error)
}

type AuthService struct {
	client  AuthClient
	storage TokenStorage
}

func NewAuthService(client AuthClient, storage TokenStorage) *AuthService {
	return &AuthService{
		client:  client,
		storage: storage,
	}
}

func (s *AuthService) Register(username, email, password string) error {
	access, refresh, err := s.client.Register(username, email, password)
	if err != nil {
		return err
	}

	if err = s.storage.Save(access, refresh); err != nil {
		return err
	}
	return nil
}

func (s *AuthService) Login(email, password string) error {
	access, refresh, err := s.client.Login(email, password)
	if err != nil {
		return err
	}

	if err = s.storage.Save(access, refresh); err != nil {
		return err
	}
	return nil
}

func (s *AuthService) Refresh() error {
	oldRefresh, err := s.storage.GetRefresh()
	if err != nil {
		return err
	}

	newAccess, newRefresh, err := s.client.Refresh(oldRefresh)
	if err != nil {
		var requestErr *client.RequestError
		if errors.As(err, &requestErr) && requestErr.StatusCode == http.StatusUnauthorized {
			_ = s.storage.Clear()
		}
		return err
	}

	if err = s.storage.Save(newAccess, newRefresh); err != nil {
		return err
	}

	return nil
}
