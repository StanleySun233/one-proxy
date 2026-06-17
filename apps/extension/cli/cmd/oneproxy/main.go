package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/StanleySun233/python-proxy/apps/extension/cli/internal/proxycommand"
)

const version = "0.4.0-dev"
const defaultNodeHTTPPort = 2988

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usageError()
	}
	switch args[0] {
	case "login":
		return runLogin(args[1:])
	case "tcp", "proxy-command":
		return runTCP(args[0], args[1:])
	case "tcp-frame":
		return runTCPFrame(args[0], args[1:])
	case "socks5", "ss5":
		return runSocks5(args[0], args[1:])
	case "version":
		fmt.Println(version)
		return nil
	case "help", "-h", "--help":
		return usageError()
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

type apiEnvelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type tenantMembership struct {
	TenantID   string `json:"tenantId"`
	TenantName string `json:"tenantName"`
}

type loginResponse struct {
	Account struct {
		Account string `json:"account"`
	} `json:"account"`
	AccessToken       string             `json:"accessToken"`
	TenantMemberships []tenantMembership `json:"tenantMemberships"`
	ActiveTenantID    *string            `json:"activeTenantId"`
}

type bootstrapResponse struct {
	ProxyToken          string `json:"proxyToken"`
	ProxyTokenExpiresAt string `json:"proxyTokenExpiresAt"`
}

func runLogin(args []string) error {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var panelURL string
	var account string
	var tenantID string
	var tokenFile string
	var accessTokenFile string
	fs.StringVar(&panelURL, "panel-url", "", "OneProxy panel URL")
	fs.StringVar(&account, "account", "", "account")
	fs.StringVar(&tenantID, "tenant-id", "", "tenant ID")
	fs.StringVar(&tokenFile, "token-file", defaultTokenFile(), "file to write the proxy token")
	fs.StringVar(&accessTokenFile, "access-token-file", defaultAccessTokenFile(), "file to write the account access token")
	if err := fs.Parse(args); err != nil {
		return err
	}
	password := os.Getenv("ONEPROXY_PASSWORD")
	if panelURL == "" || account == "" || password == "" {
		return fmt.Errorf("usage: ONEPROXY_PASSWORD=<password> oneproxy login --panel-url <url> --account <account> [--tenant-id <id>] [--token-file path]")
	}
	session, err := loginAccount(panelURL, account, password)
	if err != nil {
		return err
	}
	selectedTenantID, err := selectTenantID(session, tenantID)
	if err != nil {
		return err
	}
	bootstrap, err := extensionBootstrap(panelURL, session.AccessToken, selectedTenantID)
	if err != nil {
		return err
	}
	if bootstrap.ProxyToken == "" {
		return fmt.Errorf("missing proxy token in bootstrap response")
	}
	if err := writeProxyTokenFile(tokenFile, bootstrap.ProxyToken); err != nil {
		return err
	}
	if err := writeProxyTokenFile(accessTokenFile, session.AccessToken); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "logged in as %s tenant=%s token_expires=%s token_file=%s access_token_file=%s\n", session.Account.Account, selectedTenantID, bootstrap.ProxyTokenExpiresAt, tokenFile, accessTokenFile)
	return nil
}

func loginAccount(panelURL string, account string, password string) (loginResponse, error) {
	var result loginResponse
	body, _ := json.Marshal(map[string]string{"account": account, "password": password})
	req, err := http.NewRequest(http.MethodPost, trimURL(panelURL)+"/api/auth/login", bytes.NewReader(body))
	if err != nil {
		return result, err
	}
	req.Header.Set("Content-Type", "application/json")
	return decodeAPIResponse[loginResponse](req)
}

func extensionBootstrap(panelURL string, accessToken string, tenantID string) (bootstrapResponse, error) {
	req, err := http.NewRequest(http.MethodGet, trimURL(panelURL)+"/api/proxy/extension/bootstrap", nil)
	if err != nil {
		return bootstrapResponse{}, err
	}
	req.Header.Set("X-One-Proxy-Access-Token", accessToken)
	req.Header.Set("X-One-Proxy-Tenant-ID", tenantID)
	return decodeAPIResponse[bootstrapResponse](req)
}

func decodeAPIResponse[T any](req *http.Request) (T, error) {
	var result T
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	var envelope apiEnvelope[T]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return result, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || envelope.Code != 0 {
		if envelope.Message != "" {
			return result, errors.New(envelope.Message)
		}
		return result, fmt.Errorf("http_%d", resp.StatusCode)
	}
	return envelope.Data, nil
}

func selectTenantID(session loginResponse, requested string) (string, error) {
	if requested != "" {
		for _, membership := range session.TenantMemberships {
			if membership.TenantID == requested {
				return requested, nil
			}
		}
		return "", errors.New("tenant_forbidden")
	}
	if session.ActiveTenantID != nil && *session.ActiveTenantID != "" {
		return *session.ActiveTenantID, nil
	}
	if len(session.TenantMemberships) == 1 {
		return session.TenantMemberships[0].TenantID, nil
	}
	for _, membership := range session.TenantMemberships {
		fmt.Fprintf(os.Stderr, "tenant %s %s\n", membership.TenantID, membership.TenantName)
	}
	return "", errors.New("tenant_required")
}

func writeProxyTokenFile(path string, token string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func defaultTokenFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "proxy-token"
	}
	return filepath.Join(home, ".config", "oneproxy", "proxy-token")
}

func defaultAccessTokenFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "access-token"
	}
	return filepath.Join(home, ".config", "oneproxy", "access-token")
}

func trimURL(value string) string {
	for len(value) > 0 && value[len(value)-1] == '/' {
		value = value[:len(value)-1]
	}
	return value
}

func runTCP(name string, args []string) error {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg proxycommand.Config
	fs.StringVar(&cfg.EntryHost, "entry-host", "", "OneProxy node host")
	fs.IntVar(&cfg.EntryPort, "entry-port", defaultNodeHTTPPort, "OneProxy node proxy port")
	fs.StringVar(&cfg.TargetHost, "target-host", "", "target host behind OneProxy")
	fs.IntVar(&cfg.TargetPort, "target-port", 0, "target port behind OneProxy")
	fs.StringVar(&cfg.TokenEnv, "token-env", "ONEPROXY_PROXY_TOKEN", "environment variable containing the proxy token")
	fs.StringVar(&cfg.TokenFile, "token-file", "", "file containing the proxy token")
	addDirectFlags(fs, &cfg)
	fs.DurationVar(&cfg.ConnectTimeout, "connect-timeout", 10*time.Second, "TCP connect timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return proxycommand.RunTCP(cfg, os.Stdin, os.Stdout)
}

func runTCPFrame(name string, args []string) error {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg proxycommand.Config
	fs.StringVar(&cfg.EntryHost, "entry-host", "", "OneProxy TCP access host")
	fs.IntVar(&cfg.EntryPort, "entry-port", 2990, "OneProxy TCP access port")
	fs.StringVar(&cfg.TargetHost, "target-host", "", "target host behind OneProxy")
	fs.IntVar(&cfg.TargetPort, "target-port", 0, "target port behind OneProxy")
	fs.StringVar(&cfg.TokenEnv, "token-env", "ONEPROXY_PROXY_TOKEN", "environment variable containing the proxy token")
	fs.StringVar(&cfg.TokenFile, "token-file", "", "file containing the proxy token")
	addDirectFlags(fs, &cfg)
	fs.DurationVar(&cfg.ConnectTimeout, "connect-timeout", 10*time.Second, "TCP connect timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return proxycommand.RunTCPFrame(cfg, os.Stdin, os.Stdout)
}

func runSocks5(name string, args []string) error {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg proxycommand.Socks5Config
	fs.StringVar(&cfg.ListenAddr, "listen", "127.0.0.1:1080", "local SOCKS5 listen address")
	fs.StringVar(&cfg.EntryHost, "entry-host", "", "OneProxy node host")
	fs.IntVar(&cfg.EntryPort, "entry-port", defaultNodeHTTPPort, "OneProxy node proxy port")
	fs.StringVar(&cfg.TokenEnv, "token-env", "ONEPROXY_PROXY_TOKEN", "environment variable containing the proxy token")
	fs.StringVar(&cfg.TokenFile, "token-file", "", "file containing the proxy token")
	fs.StringVar(&cfg.PanelURL, "direct-panel-url", "", "OneProxy panel URL for direct sessions")
	fs.StringVar(&cfg.AccessTokenEnv, "access-token-env", "ONEPROXY_ACCESS_TOKEN", "environment variable containing the account access token")
	fs.StringVar(&cfg.AccessTokenFile, "access-token-file", "", "file containing the account access token")
	fs.StringVar(&cfg.TenantID, "tenant-id", "", "tenant ID for direct sessions")
	fs.StringVar(&cfg.AccessPathID, "direct-access-path-id", "", "access path ID for direct sessions")
	fs.DurationVar(&cfg.ConnectTimeout, "connect-timeout", 10*time.Second, "TCP connect timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return proxycommand.RunSocks5(cfg)
}

func usageError() error {
	return fmt.Errorf("usage: oneproxy <login|tcp|proxy-command|tcp-frame|socks5|ss5> ...")
}

func addDirectFlags(fs *flag.FlagSet, cfg *proxycommand.Config) {
	fs.StringVar(&cfg.PanelURL, "direct-panel-url", "", "OneProxy panel URL for direct sessions")
	fs.StringVar(&cfg.AccessTokenEnv, "access-token-env", "ONEPROXY_ACCESS_TOKEN", "environment variable containing the account access token")
	fs.StringVar(&cfg.AccessTokenFile, "access-token-file", "", "file containing the account access token")
	fs.StringVar(&cfg.TenantID, "tenant-id", "", "tenant ID for direct sessions")
	fs.StringVar(&cfg.AccessPathID, "direct-access-path-id", "", "access path ID for direct sessions")
}
