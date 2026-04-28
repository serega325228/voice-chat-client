package client

import (
	"fmt"
	"net/http"
)

type AuthRoutes struct {
	RegisterPath string
	LoginPath    string
	RefreshPath  string
}

func DefaultAuthRoutes() AuthRoutes {
	return AuthRoutes{
		RegisterPath: "/auth/register",
		LoginPath:    "/auth/login",
		RefreshPath:  "/auth/refresh",
	}
}

type AuthClient struct {
	api    *apiClient
	routes AuthRoutes
}

func NewAuthClient(config ClientConfig, routes AuthRoutes) (*AuthClient, error) {
	api, err := newAPIClient(config)
	if err != nil {
		return nil, err
	}

	routes = normalizeAuthRoutes(routes)

	return &AuthClient{
		api:    api,
		routes: routes,
	}, nil
}

func (c *AuthClient) Register(username, email, password string) (string, string, error) {
	requestURL, err := c.api.joinURL(c.routes.RegisterPath)
	if err != nil {
		return "", "", err
	}

	request, err := c.api.newJSONRequest(http.MethodPost, requestURL, map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	}, false)
	if err != nil {
		return "", "", err
	}

	var response tokenResponse
	if err := c.api.doJSON(request, &response); err != nil {
		return "", "", err
	}

	return response.Resolve()
}

func (c *AuthClient) Login(email, password string) (string, string, error) {
	requestURL, err := c.api.joinURL(c.routes.LoginPath)
	if err != nil {
		return "", "", err
	}

	request, err := c.api.newJSONRequest(http.MethodPost, requestURL, map[string]string{
		"email":    email,
		"password": password,
	}, false)
	if err != nil {
		return "", "", err
	}

	var response tokenResponse
	if err := c.api.doJSON(request, &response); err != nil {
		return "", "", err
	}

	return response.Resolve()
}

func (c *AuthClient) Refresh(refresh string) (string, string, error) {
	requestURL, err := c.api.joinURL(c.routes.RefreshPath)
	if err != nil {
		return "", "", err
	}

	request, err := c.api.newJSONRequest(http.MethodPost, requestURL, map[string]string{
		"refreshToken": refresh,
	}, false)
	if err != nil {
		return "", "", err
	}

	var response tokenResponse
	if err := c.api.doJSON(request, &response); err != nil {
		return "", "", err
	}

	return response.Resolve()
}

type tokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	Access       string `json:"access"`
	Refresh      string `json:"refresh"`
}

func (r tokenResponse) Resolve() (string, string, error) {
	accessToken := r.AccessToken
	if accessToken == "" {
		accessToken = r.Access
	}

	refreshToken := r.RefreshToken
	if refreshToken == "" {
		refreshToken = r.Refresh
	}

	if accessToken == "" || refreshToken == "" {
		return "", "", fmt.Errorf("client: auth response does not contain both access and refresh tokens")
	}

	return accessToken, refreshToken, nil
}

func normalizeAuthRoutes(routes AuthRoutes) AuthRoutes {
	defaults := DefaultAuthRoutes()

	if routes.RegisterPath == "" {
		routes.RegisterPath = defaults.RegisterPath
	}
	if routes.LoginPath == "" {
		routes.LoginPath = defaults.LoginPath
	}
	if routes.RefreshPath == "" {
		routes.RefreshPath = defaults.RefreshPath
	}

	return routes
}
