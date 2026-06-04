# OneProxy CLI

`oneproxy` is the local helper for non-browser clients. It exposes HTTP CONNECT, TCP access frame, and local SOCKS5 commands through a OneProxy node.

## Commands

```bash
ONEPROXY_PROXY_TOKEN=... oneproxy tcp \
  --entry-host 172.20.116.58 \
  --entry-port 2333 \
  --target-host 172.20.116.91 \
  --target-port 22
```

`proxy-command` is an alias intended for OpenSSH and VS Code Remote-SSH:

```sshconfig
Host oneproxy-node-ssh
  HostName 172.20.116.91
  Port 22
  User ubuntu
  ProxyCommand oneproxy proxy-command --entry-host 172.20.116.58 --entry-port 2333 --target-host %h --target-port %p
```

Tokens are read from `ONEPROXY_PROXY_TOKEN` by default, or from `--token-file`. The CLI does not accept proxy tokens as command-line values.

Raw TCP services can use the TCP access listener:

```bash
ONEPROXY_PROXY_TOKEN=... oneproxy tcp-frame \
  --entry-host 172.20.116.58 \
  --entry-port 2990 \
  --target-host 172.20.116.91 \
  --target-port 3389
```

SOCKS5 and SS5-compatible clients can use the local adapter:

```bash
ONEPROXY_PROXY_TOKEN=... oneproxy socks5 \
  --listen 127.0.0.1:1080 \
  --entry-host 172.20.116.58 \
  --entry-port 2333
```
