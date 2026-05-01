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
		RegisterPath: "/api/user/register",
		LoginPath:    "/api/user/login",
		RefreshPath:  "/api/token/refresh",
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

	request, err := c.api.newJSONRequest(http.MethodPost, requestURL, registerRequest{
		Username: username,
		Email:    email,
		Password: password,
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

	request, err := c.api.newJSONRequest(http.MethodPost, requestURL, loginRequest{
		Email:    email,
		Password: password,
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

	request, err := c.api.newJSONRequest(http.MethodPost, requestURL, refreshRequest{
		Refresh: refresh,
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
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
}

func (r tokenResponse) Resolve() (string, string, error) {
	if r.Access == "" || r.Refresh == "" {
		return "", "", fmt.Errorf("client: auth response does not contain both access and refresh tokens")
	}

	return r.Access, r.Refresh, nil
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	Refresh string `json:"refresh"`
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
