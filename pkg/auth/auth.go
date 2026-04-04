package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Endgame-Labs/endgame-cli/pkg/endgame"
	"golang.org/x/oauth2"
)

const (
	configFileName         = ".endgame-auth.json"
	authServerMetadataURL  = "https://app.endgame.io/.well-known/oauth-authorization-server"
	defaultAuthTimeout     = 5 * time.Minute
	defaultOAuthClientName = "endgame-cli"
)

var oauthScopes = []string{"openid", "profile", "email", "offline_access"}

var directDeviceProvider = deviceProviderConfig{
	ClientID:  "client_01K0CCK4E3M8E6WQ9ETH3DQP8D",
	DeviceURL: "https://api.workos.com/user_management/authorize/device",
	TokenURL:  "https://api.workos.com/user_management/authenticate",
}

type Config struct {
	OAuth *OAuthConfig `json:"oauth,omitempty"`
}

type OAuthConfig struct {
	ClientID    string     `json:"client_id,omitempty"`
	RedirectURI string     `json:"redirect_uri,omitempty"`
	TokenURL    string     `json:"token_url,omitempty"`
	Token       OAuthToken `json:"token"`
}

type OAuthToken struct {
	AccessToken  string    `json:"access_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
}

type Status struct {
	Email   string
	OrgID   string
	OrgName string
}

type LoginOptions struct {
	Mode         string
	OpenBrowser  bool
	CallbackHost string
	CallbackPort int
}

type authServerMetadata struct {
	AuthorizationEndpoint       string   `json:"authorization_endpoint"`
	TokenEndpoint               string   `json:"token_endpoint"`
	RegistrationEndpoint        string   `json:"registration_endpoint"`
	DeviceAuthorizationEndpoint string   `json:"device_authorization_endpoint"`
	GrantTypesSupported         []string `json:"grant_types_supported"`
	ScopesSupported             []string `json:"scopes_supported"`
}

type dynamicClientRegistrationRequest struct {
	ClientName              string   `json:"client_name"`
	ApplicationType         string   `json:"application_type"`
	GrantTypes              []string `json:"grant_types"`
	RedirectURIs            []string `json:"redirect_uris"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

type dynamicClientRegistrationResponse struct {
	ClientID string `json:"client_id"`
}

type callbackResult struct {
	Code string
	Err  string
}

type tokenClaims struct {
	Email string `json:"email"`
	Org   struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"urn:endgame:workos_org"`
}

type deviceTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	IDToken      string `json:"id_token"`
	Error        string `json:"error"`
	User         *struct {
		Email string `json:"email"`
	} `json:"user,omitempty"`
}

type deviceAuthorizationResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type deviceProviderConfig struct {
	ClientID  string
	DeviceURL string
	TokenURL  string
}

func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, configFileName), nil
}

func Save(config Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("not authenticated")
		}
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	if config.OAuth == nil || strings.TrimSpace(config.OAuth.ClientID) == "" || strings.TrimSpace(config.OAuth.Token.AccessToken) == "" {
		return nil, errors.New("not authenticated")
	}

	return &config, nil
}

func Login(options LoginOptions) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	fmt.Println("Endgame Authentication")
	fmt.Println()
	fmt.Printf("OAuth tokens will be stored in: %s\n", configPath)
	fmt.Println("Signing in grants this CLI read access to all Endgame Context Graph information")
	fmt.Println("you already have rights to inspect, including deals, CRM data, and related context.")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), defaultAuthTimeout)
	defer cancel()

	if options.Mode == "device" {
		return loginWithDeviceFlow(ctx, configPath)
	}

	metadata, err := discoverAuthServerMetadata(ctx)
	if err != nil {
		return err
	}

	callbackHost := strings.TrimSpace(options.CallbackHost)
	if callbackHost == "" {
		callbackHost = "127.0.0.1"
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(callbackHost, strconv.Itoa(options.CallbackPort)))
	if err != nil {
		return fmt.Errorf("start localhost callback listener: %w", err)
	}
	defer listener.Close()

	redirectURI := fmt.Sprintf("http://%s/callback", listener.Addr().String())
	clientID, err := registerOAuthClient(ctx, metadata, redirectURI)
	if err != nil {
		return err
	}

	oauthConfig := &oauth2.Config{
		ClientID:    clientID,
		RedirectURL: redirectURI,
		Scopes:      oauthScopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  metadata.AuthorizationEndpoint,
			TokenURL: metadata.TokenEndpoint,
		},
	}

	state, err := randomString(32)
	if err != nil {
		return err
	}
	verifier, err := randomString(48)
	if err != nil {
		return err
	}

	callbackCh := make(chan callbackResult, 1)
	server := startCallbackServer(listener, state, callbackCh)
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	authURL := oauthConfig.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
	)

	if options.OpenBrowser {
		fmt.Println("Opening browser for Endgame sign-in...")
		if err := openBrowser(authURL); err != nil {
			fmt.Printf("Open this URL in your browser:\n%s\n\n", authURL)
		}
	} else {
		fmt.Printf("Open this URL in your browser:\n%s\n\n", authURL)
		fmt.Printf("Waiting for OAuth callback on %s\n\n", redirectURI)
	}

	var result callbackResult
	select {
	case result = <-callbackCh:
	case <-ctx.Done():
		return errors.New("timed out waiting for OAuth callback")
	}

	if result.Err != "" {
		return fmt.Errorf("oauth callback failed: %s", result.Err)
	}
	if result.Code == "" {
		return errors.New("oauth callback did not provide an authorization code")
	}

	token, err := oauthConfig.Exchange(ctx, result.Code, oauth2.VerifierOption(verifier))
	if err != nil {
		return fmt.Errorf("exchange authorization code: %w", err)
	}

	config := Config{
		OAuth: &OAuthConfig{
			ClientID:    clientID,
			RedirectURI: redirectURI,
			TokenURL:    metadata.TokenEndpoint,
			Token:       tokenToConfig(token),
		},
	}
	if err := Save(config); err != nil {
		return err
	}

	status, err := Verify()
	if err != nil {
		return fmt.Errorf("authenticated but MCP verification failed: %w", err)
	}

	if status.OrgName != "" {
		fmt.Printf("Authenticated for %s (%s)\n", status.OrgName, status.OrgID)
	} else if status.OrgID != "" {
		fmt.Printf("Authenticated for org %s\n", status.OrgID)
	} else {
		fmt.Println("Authenticated")
	}
	return nil
}

func SupportsDeviceLogin() (bool, error) {
	return strings.TrimSpace(directDeviceProvider.ClientID) != "" &&
		strings.TrimSpace(directDeviceProvider.DeviceURL) != "" &&
		strings.TrimSpace(directDeviceProvider.TokenURL) != "", nil
}

func loginWithDeviceFlow(ctx context.Context, configPath string) error {
	clientID := directDeviceProvider.ClientID
	deviceAuth, err := startDirectDeviceAuthorization(ctx, directDeviceProvider)
	if err != nil {
		return err
	}

	fmt.Println("Device login requested.")
	if deviceAuth.VerificationURIComplete != "" {
		fmt.Printf("Open this URL in your browser:\n%s\n\n", deviceAuth.VerificationURIComplete)
	} else {
		fmt.Printf("Open this URL in your browser:\n%s\n\n", deviceAuth.VerificationURI)
		fmt.Printf("Then enter code: %s\n\n", deviceAuth.UserCode)
	}
	fmt.Printf("OAuth tokens will be stored in: %s\n\n", configPath)

	token, email, err := pollDirectDeviceAuthorization(ctx, directDeviceProvider, clientID, deviceAuth)
	if err != nil {
		return err
	}

	if strings.TrimSpace(email) != "" {
		fmt.Printf("Authenticated as %s\n", email)
	}

	config := Config{
		OAuth: &OAuthConfig{
			ClientID: clientID,
			TokenURL: directDeviceProvider.TokenURL,
			Token:    tokenToConfig(token),
		},
	}
	if err := Save(config); err != nil {
		return err
	}

	status, err := Verify()
	if err != nil {
		return fmt.Errorf("authenticated but MCP verification failed: %w", err)
	}

	if status.OrgName != "" {
		fmt.Printf("Authenticated for %s (%s)\n", status.OrgName, status.OrgID)
	} else if status.OrgID != "" {
		fmt.Printf("Authenticated for org %s\n", status.OrgID)
	} else {
		fmt.Println("Authenticated")
	}
	return nil
}

func Verify() (*Status, error) {
	client, err := NewClient()
	if err != nil {
		return nil, err
	}

	if err := client.Initialize(); err != nil {
		return nil, err
	}
	if _, err := client.ListTools(); err != nil {
		return nil, err
	}

	token, err := currentToken()
	if err != nil {
		return nil, err
	}

	status := &Status{}
	if claims, err := parseTokenClaims(token.AccessToken); err == nil {
		status.Email = claims.Email
		status.OrgID = claims.Org.ID
		status.OrgName = claims.Org.Name
	}

	return status, nil
}

func Logout() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	err = os.Remove(configPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func NewClient() (*endgame.Client, error) {
	httpClient, err := NewHTTPClient()
	if err != nil {
		return nil, err
	}
	return endgame.NewClient(httpClient)
}

func NewHTTPClient() (*http.Client, error) {
	timeout, err := getTimeout()
	if err != nil {
		return nil, err
	}

	config, err := Load()
	if err != nil {
		return nil, err
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{Timeout: timeout})
	source, err := newTokenSource(ctx, config)
	if err != nil {
		return nil, err
	}

	httpClient := oauth2.NewClient(ctx, source)
	httpClient.Timeout = timeout
	return httpClient, nil
}

func currentToken() (*oauth2.Token, error) {
	config, err := Load()
	if err != nil {
		return nil, err
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{Timeout: 30 * time.Second})
	source, err := newTokenSource(ctx, config)
	if err != nil {
		return nil, err
	}

	return source.Token()
}

func newTokenSource(ctx context.Context, config *Config) (oauth2.TokenSource, error) {
	metadata, err := discoverAuthServerMetadata(ctx)
	if err != nil {
		return nil, err
	}

	tokenURL := strings.TrimSpace(config.OAuth.TokenURL)
	if tokenURL == "" {
		tokenURL = metadata.TokenEndpoint
	}

	oauthConfig := &oauth2.Config{
		ClientID:    config.OAuth.ClientID,
		RedirectURL: config.OAuth.RedirectURI,
		Scopes:      oauthScopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  metadata.AuthorizationEndpoint,
			TokenURL: tokenURL,
		},
	}

	base := oauthConfig.TokenSource(ctx, configToToken(config.OAuth.Token))
	return &persistingTokenSource{
		base:   oauth2.ReuseTokenSource(configToToken(config.OAuth.Token), base),
		config: config,
	}, nil
}

type persistingTokenSource struct {
	base   oauth2.TokenSource
	config *Config
}

func (p *persistingTokenSource) Token() (*oauth2.Token, error) {
	token, err := p.base.Token()
	if err != nil {
		return nil, err
	}

	newToken := tokenToConfig(token)
	if tokensEqual(p.config.OAuth.Token, newToken) {
		return token, nil
	}

	p.config.OAuth.Token = newToken
	if err := Save(*p.config); err != nil {
		return nil, err
	}

	return token, nil
}

func discoverAuthServerMetadata(ctx context.Context) (*authServerMetadata, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, authServerMetadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build auth metadata request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch auth server metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("fetch auth server metadata: http %d", resp.StatusCode)
	}

	var metadata authServerMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("decode auth server metadata: %w", err)
	}

	switch {
	case metadata.AuthorizationEndpoint == "":
		return nil, errors.New("authorization endpoint missing from auth server metadata")
	case metadata.TokenEndpoint == "":
		return nil, errors.New("token endpoint missing from auth server metadata")
	case metadata.RegistrationEndpoint == "":
		return nil, errors.New("registration endpoint missing from auth server metadata")
	}

	return &metadata, nil
}

func supportsDeviceAuthorization(metadata *authServerMetadata) bool {
	if strings.TrimSpace(metadata.DeviceAuthorizationEndpoint) == "" {
		return false
	}
	for _, grantType := range metadata.GrantTypesSupported {
		if grantType == "urn:ietf:params:oauth:grant-type:device_code" {
			return true
		}
	}
	return false
}

func registerOAuthClient(ctx context.Context, metadata *authServerMetadata, redirectURI string) (string, error) {
	payload := dynamicClientRegistrationRequest{
		ClientName:              defaultOAuthClientName,
		ApplicationType:         "native",
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		RedirectURIs:            []string{redirectURI},
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: "none",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal client registration payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, metadata.RegistrationEndpoint, strings.NewReader(string(data)))
	if err != nil {
		return "", fmt.Errorf("build client registration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("register OAuth client: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("register OAuth client: http %d", resp.StatusCode)
	}

	var registration dynamicClientRegistrationResponse
	if err := json.NewDecoder(resp.Body).Decode(&registration); err != nil {
		return "", fmt.Errorf("decode client registration response: %w", err)
	}
	if strings.TrimSpace(registration.ClientID) == "" {
		return "", errors.New("client registration response did not include client_id")
	}

	return registration.ClientID, nil
}

func registerDeviceOAuthClient(ctx context.Context, metadata *authServerMetadata) (string, error) {
	payload := dynamicClientRegistrationRequest{
		ClientName:              defaultOAuthClientName,
		ApplicationType:         "native",
		GrantTypes:              []string{"urn:ietf:params:oauth:grant-type:device_code", "refresh_token"},
		ResponseTypes:           []string{},
		TokenEndpointAuthMethod: "none",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal device client registration payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, metadata.RegistrationEndpoint, strings.NewReader(string(data)))
	if err != nil {
		return "", fmt.Errorf("build device client registration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("register OAuth device client: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("register OAuth device client: http %d", resp.StatusCode)
	}

	var registration dynamicClientRegistrationResponse
	if err := json.NewDecoder(resp.Body).Decode(&registration); err != nil {
		return "", fmt.Errorf("decode device client registration response: %w", err)
	}
	if strings.TrimSpace(registration.ClientID) == "" {
		return "", errors.New("device client registration response did not include client_id")
	}

	return registration.ClientID, nil
}

func startDeviceAuthorization(ctx context.Context, metadata *authServerMetadata, clientID string) (*deviceAuthorizationResponse, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("scope", strings.Join(oauthScopes, " "))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, metadata.DeviceAuthorizationEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build device authorization request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("start device authorization: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("start device authorization: http %d", resp.StatusCode)
	}

	var deviceAuth deviceAuthorizationResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceAuth); err != nil {
		return nil, fmt.Errorf("decode device authorization response: %w", err)
	}
	if strings.TrimSpace(deviceAuth.DeviceCode) == "" {
		return nil, errors.New("device authorization response did not include device_code")
	}
	return &deviceAuth, nil
}

func startDirectDeviceAuthorization(ctx context.Context, provider deviceProviderConfig) (*deviceAuthorizationResponse, error) {
	form := url.Values{}
	form.Set("client_id", provider.ClientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.DeviceURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build device authorization request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("start device authorization: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("start device authorization: http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var deviceAuth deviceAuthorizationResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceAuth); err != nil {
		return nil, fmt.Errorf("decode device authorization response: %w", err)
	}
	if strings.TrimSpace(deviceAuth.DeviceCode) == "" {
		return nil, errors.New("device authorization response did not include device_code")
	}
	return &deviceAuth, nil
}

func pollDeviceAuthorization(ctx context.Context, metadata *authServerMetadata, clientID string, deviceAuth *deviceAuthorizationResponse) (*oauth2.Token, error) {
	interval := time.Duration(deviceAuth.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}

	expiry := time.Now().Add(time.Duration(deviceAuth.ExpiresIn) * time.Second)
	if deviceAuth.ExpiresIn <= 0 {
		expiry = time.Now().Add(defaultAuthTimeout)
	}

	for {
		if time.Now().After(expiry) {
			return nil, errors.New("device authorization expired before login completed")
		}

		form := url.Values{}
		form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
		form.Set("device_code", deviceAuth.DeviceCode)
		form.Set("client_id", clientID)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, metadata.TokenEndpoint, strings.NewReader(form.Encode()))
		if err != nil {
			return nil, fmt.Errorf("build device token request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("poll device token: %w", err)
		}

		var tokenResp struct {
			AccessToken  string `json:"access_token"`
			TokenType    string `json:"token_type"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
			Error        string `json:"error"`
		}
		decodeErr := json.NewDecoder(resp.Body).Decode(&tokenResp)
		resp.Body.Close()
		if decodeErr != nil {
			return nil, fmt.Errorf("decode device token response: %w", decodeErr)
		}

		if resp.StatusCode < http.StatusBadRequest && tokenResp.AccessToken != "" {
			token := &oauth2.Token{
				AccessToken:  tokenResp.AccessToken,
				TokenType:    tokenResp.TokenType,
				RefreshToken: tokenResp.RefreshToken,
			}
			if tokenResp.ExpiresIn > 0 {
				token.Expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
			}
			return token, nil
		}

		switch tokenResp.Error {
		case "authorization_pending", "":
			// keep polling
		case "slow_down":
			interval += 5 * time.Second
		case "access_denied":
			return nil, errors.New("device authorization was denied")
		case "expired_token":
			return nil, errors.New("device authorization expired")
		default:
			if tokenResp.Error != "" {
				return nil, fmt.Errorf("device authorization failed: %s", tokenResp.Error)
			}
			return nil, fmt.Errorf("device authorization failed: http %d", resp.StatusCode)
		}

		select {
		case <-ctx.Done():
			return nil, errors.New("timed out waiting for device authorization")
		case <-time.After(interval):
		}
	}
}

func pollDirectDeviceAuthorization(ctx context.Context, provider deviceProviderConfig, clientID string, deviceAuth *deviceAuthorizationResponse) (*oauth2.Token, string, error) {
	interval := time.Duration(deviceAuth.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}

	expiry := time.Now().Add(time.Duration(deviceAuth.ExpiresIn) * time.Second)
	if deviceAuth.ExpiresIn <= 0 {
		expiry = time.Now().Add(defaultAuthTimeout)
	}

	for {
		if time.Now().After(expiry) {
			return nil, "", errors.New("device authorization expired before login completed")
		}

		form := url.Values{}
		form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
		form.Set("device_code", deviceAuth.DeviceCode)
		form.Set("client_id", clientID)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.TokenURL, strings.NewReader(form.Encode()))
		if err != nil {
			return nil, "", fmt.Errorf("build device token request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, "", fmt.Errorf("poll device token: %w", err)
		}

		var tokenResp deviceTokenResponse
		decodeErr := json.NewDecoder(resp.Body).Decode(&tokenResp)
		resp.Body.Close()
		if decodeErr != nil {
			return nil, "", fmt.Errorf("decode device token response: %w", decodeErr)
		}

		if resp.StatusCode < http.StatusBadRequest && tokenResp.AccessToken != "" {
			token := &oauth2.Token{
				AccessToken:  tokenResp.AccessToken,
				TokenType:    tokenResp.TokenType,
				RefreshToken: tokenResp.RefreshToken,
			}
			if tokenResp.ExpiresIn > 0 {
				token.Expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
			}
			return token, emailFromDeviceTokenResponse(tokenResp), nil
		}

		switch tokenResp.Error {
		case "authorization_pending", "":
		case "slow_down":
			interval += 5 * time.Second
		case "access_denied":
			return nil, "", errors.New("device authorization was denied")
		case "expired_token":
			return nil, "", errors.New("device authorization expired")
		default:
			if tokenResp.Error != "" {
				return nil, "", fmt.Errorf("device authorization failed: %s", tokenResp.Error)
			}
			return nil, "", fmt.Errorf("device authorization failed: http %d", resp.StatusCode)
		}

		select {
		case <-ctx.Done():
			return nil, "", errors.New("timed out waiting for device authorization")
		case <-time.After(interval):
		}
	}
}

func emailFromDeviceTokenResponse(tokenResp deviceTokenResponse) string {
	if tokenResp.User != nil && strings.TrimSpace(tokenResp.User.Email) != "" {
		return strings.TrimSpace(tokenResp.User.Email)
	}
	return emailFromIDToken(tokenResp.IDToken)
}

func emailFromIDToken(idToken string) string {
	parts := strings.Split(strings.TrimSpace(idToken), ".")
	if len(parts) != 3 {
		return ""
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}

	var claims struct {
		Email string `json:"email"`
		Sub   string `json:"sub"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	if strings.TrimSpace(claims.Email) != "" {
		return strings.TrimSpace(claims.Email)
	}
	return strings.TrimSpace(claims.Sub)
}

func startCallbackServer(listener net.Listener, expectedState string, callbackCh chan<- callbackResult) *http.Server {
	mux := http.NewServeMux()
	server := &http.Server{Handler: mux}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if errValue := strings.TrimSpace(r.URL.Query().Get("error")); errValue != "" {
			select {
			case callbackCh <- callbackResult{Err: errValue}:
			default:
			}
			http.Error(w, "Authentication failed. Return to the terminal for details.", http.StatusBadRequest)
			return
		}

		if state := r.URL.Query().Get("state"); state != expectedState {
			select {
			case callbackCh <- callbackResult{Err: "state mismatch"}:
			default:
			}
			http.Error(w, "Authentication failed. Return to the terminal for details.", http.StatusBadRequest)
			return
		}

		code := strings.TrimSpace(r.URL.Query().Get("code"))
		if code == "" {
			select {
			case callbackCh <- callbackResult{Err: "missing code"}:
			default:
			}
			http.Error(w, "Authentication failed. Return to the terminal for details.", http.StatusBadRequest)
			return
		}

		select {
		case callbackCh <- callbackResult{Code: code}:
		default:
		}

		_, _ = w.Write([]byte("Authentication complete. You can close this window."))
	})

	go func() {
		_ = server.Serve(listener)
	}()

	return server
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func parseTokenClaims(accessToken string) (*tokenClaims, error) {
	parts := strings.Split(accessToken, ".")
	if len(parts) != 3 {
		return nil, errors.New("access token is not a JWT")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode token claims: %w", err)
	}

	var claims tokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal token claims: %w", err)
	}
	return &claims, nil
}

func randomString(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func tokenToConfig(token *oauth2.Token) OAuthToken {
	return OAuthToken{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}
}

func configToToken(token OAuthToken) *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}
}

func tokensEqual(left, right OAuthToken) bool {
	return left.AccessToken == right.AccessToken &&
		left.TokenType == right.TokenType &&
		left.RefreshToken == right.RefreshToken &&
		left.Expiry.Equal(right.Expiry)
}

func getTimeout() (time.Duration, error) {
	timeout := 120 * time.Second
	if timeoutValue := strings.TrimSpace(os.Getenv("ENDGAME_TIMEOUT_SECONDS")); timeoutValue != "" {
		timeoutSeconds, err := strconv.Atoi(timeoutValue)
		if err != nil || timeoutSeconds <= 0 {
			return 0, fmt.Errorf("invalid ENDGAME_TIMEOUT_SECONDS: %q", timeoutValue)
		}
		timeout = time.Duration(timeoutSeconds) * time.Second
	}
	return timeout, nil
}
