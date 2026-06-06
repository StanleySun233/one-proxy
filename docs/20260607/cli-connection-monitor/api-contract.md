# API Contract: CLI Connection Monitor

## Command

```text
onep monitor <program> [args...]
```

## Log Event

Each event is written as one JSON object per line.

```json
{
  "timestamp": "2026-06-07T00:00:00.000Z",
  "source": "netstat",
  "process": "Game.exe",
  "pid": 1234,
  "protocol": "tcp",
  "localAddress": "127.0.0.1",
  "localPort": 50000,
  "remoteAddress": "203.0.113.10",
  "remotePort": 443,
  "remoteHost": "203.0.113.10",
  "domain": null,
  "domainSource": null,
  "state": "ESTABLISHED"
}
```

## Console Event

```text
onep monitor: Game.exe[1234] tcp 203.0.113.10:443 ESTABLISHED
```

