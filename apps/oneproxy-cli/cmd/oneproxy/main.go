package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/StanleySun233/python-proxy/apps/oneproxy-cli/internal/proxycommand"
)

const version = "0.4.0-dev"

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

func runTCP(name string, args []string) error {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg proxycommand.Config
	fs.StringVar(&cfg.EntryHost, "entry-host", "", "OneProxy node host")
	fs.IntVar(&cfg.EntryPort, "entry-port", 2333, "OneProxy node proxy port")
	fs.StringVar(&cfg.TargetHost, "target-host", "", "target host behind OneProxy")
	fs.IntVar(&cfg.TargetPort, "target-port", 0, "target port behind OneProxy")
	fs.StringVar(&cfg.TokenEnv, "token-env", "ONEPROXY_PROXY_TOKEN", "environment variable containing the proxy token")
	fs.StringVar(&cfg.TokenFile, "token-file", "", "file containing the proxy token")
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
	fs.IntVar(&cfg.EntryPort, "entry-port", 2333, "OneProxy node proxy port")
	fs.StringVar(&cfg.TokenEnv, "token-env", "ONEPROXY_PROXY_TOKEN", "environment variable containing the proxy token")
	fs.StringVar(&cfg.TokenFile, "token-file", "", "file containing the proxy token")
	fs.DurationVar(&cfg.ConnectTimeout, "connect-timeout", 10*time.Second, "TCP connect timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return proxycommand.RunSocks5(cfg)
}

func usageError() error {
	return fmt.Errorf("usage: oneproxy <tcp|proxy-command|tcp-frame|socks5|ss5> --entry-host <host> --entry-port <port> [--target-host <host> --target-port <port>] [--token-env ONEPROXY_PROXY_TOKEN|--token-file path]")
}
