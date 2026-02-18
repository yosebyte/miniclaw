// MIT License - Copyright (c) 2026 yosebyte
package provider

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	oauthClientID    = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	oauthAuthURL     = "https://claude.ai/oauth/authorize"
	oauthTokenURL    = "https://claude.ai/oauth/token"
	oauthRedirectURL = "http://localhost:54321/callback"
	oauthScopes      = "openid profile email offline_access"
)

// OAuthTokenResponse is the response from the token endpoint.
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// Login performs the OAuth PKCE flow, returning access and refresh tokens.
func Login(ctx context.Context) (accessToken, refreshToken string, err error) {
	verifier, err := generateCodeVerifier()
	if err != nil {
		return "", "", fmt.Errorf("generating code verifier: %w", err)
	}
	challenge := generateCodeChallenge(verifier)

	state, err := randomString(16)
	if err != nil {
		return "", "", fmt.Errorf("generating state: %w", err)
	}

	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {oauthClientID},
		"redirect_uri":          {oauthRedirectURL},
		"scope":                 {oauthScopes},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	authURL := oauthAuthURL + "?" + params.Encode()

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	ln, err := net.Listen("tcp", "localhost:54321")
	if err != nil {
		return "", "", fmt.Errorf("starting callback server: %w", err)
	}

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			if q.Get("state") != state {
				http.Error(w, "invalid state", http.StatusBadRequest)
				errCh <- fmt.Errorf("state mismatch")
				return
			}
			if e := q.Get("error"); e != "" {
				http.Error(w, e, http.StatusBadRequest)
				errCh <- fmt.Errorf("oauth error: %s", e)
				return
			}
			code := q.Get("code")
			if code == "" {
				http.Error(w, "missing code", http.StatusBadRequest)
				errCh <- fmt.Errorf("missing code in callback")
				return
			}
			fmt.Fprint(w, "<html><body><h2>✅ Authenticated! You can close this tab.</h2></body></html>")
			codeCh <- code
		}),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		if e := srv.Serve(ln); e != nil && e != http.ErrServerClosed {
			errCh <- e
		}
	}()
	defer func() { _ = srv.Shutdown(context.Background()) }()

	fmt.Println("\n🔐 Opening claude.ai for authentication...")
	fmt.Println("If the browser doesn't open, visit this URL manually:")
	fmt.Println(authURL)
	openBrowser(authURL)

	select {
	case code := <-codeCh:
		tokens, err := exchangeCode(ctx, code, verifier)
		if err != nil {
			return "", "", err
		}
		return tokens.AccessToken, tokens.RefreshToken, nil
	case err := <-errCh:
		return "", "", err
	case <-time.After(5 * time.Minute):
		return "", "", fmt.Errorf("authentication timed out after 5 minutes")
	case <-ctx.Done():
		return "", "", ctx.Err()
	}
}

// RefreshAccessToken exchanges a refresh token for a new access token.
func RefreshAccessToken(ctx context.Context, refreshToken string) (string, string, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {oauthClientID},
		"refresh_token": {refreshToken},
	}
	resp, err := doTokenRequest(ctx, form)
	if err != nil {
		return "", "", err
	}
	return resp.AccessToken, resp.RefreshToken, nil
}

func exchangeCode(ctx context.Context, code, verifier string) (*OAuthTokenResponse, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {oauthClientID},
		"redirect_uri":  {oauthRedirectURL},
		"code":          {code},
		"code_verifier": {verifier},
	}
	return doTokenRequest(ctx, form)
}

func doTokenRequest(ctx context.Context, form url.Values) (*OAuthTokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, oauthTokenURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer res.Body.Close()

	var tok OAuthTokenResponse
	if err := json.NewDecoder(res.Body).Decode(&tok); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}
	if tok.Error != "" {
		return nil, fmt.Errorf("token error: %s - %s", tok.Error, tok.ErrorDesc)
	}
	slog.Info("OAuth token obtained", "token_type", tok.TokenType, "expires_in", tok.ExpiresIn)
	return &tok, nil
}

func generateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b)[:n], nil
}

func openBrowser(urlStr string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{urlStr}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", urlStr}
	default:
		cmd = "xdg-open"
		args = []string{urlStr}
	}
	if err := exec.Command(cmd, args...).Start(); err != nil {
		slog.Debug("could not open browser automatically", "err", err)
	}
}
