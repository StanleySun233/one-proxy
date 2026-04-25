import argparse
import asyncio
import contextlib
import ipaddress
import logging
import os
import ssl
import subprocess
from dataclasses import dataclass
from pathlib import Path
from typing import Optional
from urllib.parse import urlsplit, urlunsplit

from config_store import ConfigStore, Node


HOP_BY_HOP_HEADERS = {
    "proxy-connection",
    "proxy-authenticate",
    "proxy-authorization",
    "connection",
    "keep-alive",
    "te",
    "trailer",
    "transfer-encoding",
    "upgrade",
}


@dataclass
class Request:
    method: str
    target: str
    version: str
    headers: list[tuple[str, str]]
    raw_headers: dict[str, str]


def load_dotenv(path: Path) -> dict[str, str]:
    values: dict[str, str] = {}
    if not path.exists():
        return values
    for raw_line in path.read_text(encoding="utf-8").splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue
        if line.startswith("export "):
            line = line[7:].strip()
        if "=" not in line:
            continue
        key, value = line.split("=", 1)
        key = key.strip()
        value = value.strip()
        if not key:
            continue
        if len(value) >= 2 and value[0] == value[-1] and value[0] in {"'", '"'}:
            value = value[1:-1]
        values[key] = value
    return values


def env_value(name: str, dotenv: dict[str, str], default: str) -> str:
    return os.environ.get(name, dotenv.get(name, default))


def ensure_certificate(certfile: str, keyfile: str, days: int, common_name: str) -> None:
    cert_path = Path(certfile)
    key_path = Path(keyfile)
    cert_path.parent.mkdir(parents=True, exist_ok=True)
    key_path.parent.mkdir(parents=True, exist_ok=True)
    if cert_path.exists() and key_path.exists():
        return
    subprocess.run(
        [
            "openssl",
            "req",
            "-x509",
            "-nodes",
            "-newkey",
            "rsa:2048",
            "-keyout",
            str(key_path),
            "-out",
            str(cert_path),
            "-days",
            str(days),
            "-subj",
            f"/CN={common_name}",
        ],
        check=True,
    )
    os.chmod(key_path, 0o600)
    os.chmod(cert_path, 0o644)


def parse_host_port(value: str, default_port: int) -> tuple[str, int]:
    value = value.strip()
    if value.startswith("["):
        host, rest = value[1:].split("]", 1)
        port = int(rest[1:]) if rest.startswith(":") and rest[1:] else default_port
        return host, port
    if value.count(":") == 1:
        host, port_text = value.rsplit(":", 1)
        if port_text.isdigit():
            return host, int(port_text)
    return value, default_port


def format_host_port(host: str, port: int) -> str:
    try:
        ip = ipaddress.ip_address(host)
    except ValueError:
        ip = None
    if ip and ip.version == 6:
        return f"[{host}]:{port}"
    if ":" in host and not host.startswith("["):
        return f"[{host}]:{port}"
    return f"{host}:{port}"


async def read_request(reader: asyncio.StreamReader) -> Optional[Request]:
    try:
        raw = await reader.readuntil(b"\r\n\r\n")
    except asyncio.IncompleteReadError:
        return None
    except asyncio.LimitOverrunError:
        return None
    lines = raw.decode("iso-8859-1").split("\r\n")
    if not lines or not lines[0]:
        return None
    request_line = lines[0].split(" ", 2)
    if len(request_line) != 3:
        return None
    method, target, version = request_line
    headers = []
    raw_headers = {}
    for line in lines[1:]:
        if not line:
            continue
        if ":" not in line:
            continue
        name, value = line.split(":", 1)
        value = value.lstrip(" ")
        headers.append((name, value))
        raw_headers[name.lower()] = value
    return Request(method=method, target=target, version=version, headers=headers, raw_headers=raw_headers)


async def pipe_stream(reader: asyncio.StreamReader, writer: asyncio.StreamWriter) -> None:
    try:
        while True:
            chunk = await reader.read(65536)
            if not chunk:
                break
            writer.write(chunk)
            await writer.drain()
    except Exception:
        pass
    finally:
        with contextlib.suppress(Exception):
            writer.close()


async def tunnel(client_reader: asyncio.StreamReader, client_writer: asyncio.StreamWriter,
                 upstream_reader: asyncio.StreamReader, upstream_writer: asyncio.StreamWriter) -> None:
    await asyncio.gather(
        pipe_stream(client_reader, upstream_writer),
        pipe_stream(upstream_reader, client_writer),
    )


async def forward_fixed_body(reader: asyncio.StreamReader, writer: asyncio.StreamWriter, size: int) -> None:
    remaining = size
    while remaining > 0:
        chunk = await reader.read(min(65536, remaining))
        if not chunk:
            break
        writer.write(chunk)
        await writer.drain()
        remaining -= len(chunk)


async def forward_chunked_body(reader: asyncio.StreamReader, writer: asyncio.StreamWriter) -> None:
    while True:
        line = await reader.readuntil(b"\r\n")
        writer.write(line)
        await writer.drain()
        chunk_size = int(line.split(b";", 1)[0].strip(), 16)
        if chunk_size == 0:
            trailer = await reader.readuntil(b"\r\n")
            writer.write(trailer)
            await writer.drain()
            while True:
                extra = await reader.readuntil(b"\r\n")
                writer.write(extra)
                await writer.drain()
                if extra == b"\r\n":
                    break
            break
        data = await reader.readexactly(chunk_size + 2)
        writer.write(data)
        await writer.drain()


async def relay_response(method: str, upstream_reader: asyncio.StreamReader, client_writer: asyncio.StreamWriter) -> tuple[bool, bool]:
    try:
        raw = await upstream_reader.readuntil(b"\r\n\r\n")
    except asyncio.IncompleteReadError as exc:
        if exc.partial:
            client_writer.write(exc.partial)
            await client_writer.drain()
        return False, True
    client_writer.write(raw)
    await client_writer.drain()

    lines = raw.decode("iso-8859-1").split("\r\n")
    status_line = lines[0]
    parts = status_line.split(" ", 2)
    status_code = int(parts[1]) if len(parts) > 1 and parts[1].isdigit() else 0
    headers = {}
    for line in lines[1:]:
        if not line or ":" not in line:
            continue
        name, value = line.split(":", 1)
        headers[name.lower()] = value.lstrip(" ")

    if status_code == 101:
        return True, True
    if method.upper() == "HEAD" or status_code in {204, 304} or 100 <= status_code < 200:
        return False, True
    if "content-length" in headers:
        await forward_fixed_body(upstream_reader, client_writer, int(headers["content-length"]))
        return False, True
    if headers.get("transfer-encoding", "").lower() == "chunked":
        await forward_chunked_body(upstream_reader, client_writer)
        return False, True
    while True:
        chunk = await upstream_reader.read(65536)
        if not chunk:
            break
        client_writer.write(chunk)
        await client_writer.drain()
    return False, True


class ProxyServer:
    def __init__(self, config_store: ConfigStore, instance_id: str) -> None:
        self.log = logging.getLogger("rootless_proxy")
        self.config_store = config_store
        self.instance_id = instance_id

    def get_local_node(self) -> Node:
        return self.config_store.get_local_node(self.instance_id)

    def get_upstream_node(self) -> Node | None:
        return self.config_store.get_upstream_node(self.get_local_node().id)

    async def open_upstream_connection(self, target_host: str, target_port: int) -> tuple[asyncio.StreamReader, asyncio.StreamWriter]:
        upstream_node = self.get_upstream_node()
        if upstream_node is None:
            return await asyncio.open_connection(target_host, target_port)
        if upstream_node.protocol != "http_proxy":
            raise ValueError(f"unsupported upstream protocol: {upstream_node.protocol}")
        return await asyncio.open_connection(upstream_node.host, upstream_node.port)

    async def negotiate_upstream_connect(
        self,
        upstream_reader: asyncio.StreamReader,
        upstream_writer: asyncio.StreamWriter,
        target_host: str,
        target_port: int,
    ) -> None:
        address = format_host_port(target_host, target_port)
        upstream_writer.write(
            f"CONNECT {address} HTTP/1.1\r\n"
            f"Host: {address}\r\n"
            "Connection: close\r\n\r\n".encode("iso-8859-1")
        )
        await upstream_writer.drain()
        raw = await upstream_reader.readuntil(b"\r\n\r\n")
        lines = raw.decode("iso-8859-1").split("\r\n")
        status_line = lines[0] if lines else ""
        parts = status_line.split(" ", 2)
        status_code = int(parts[1]) if len(parts) > 1 and parts[1].isdigit() else 0
        if status_code != 200:
            raise ValueError(f"upstream CONNECT failed: {status_line}")

    async def handle(self, reader: asyncio.StreamReader, writer: asyncio.StreamWriter) -> None:
        peer = writer.get_extra_info("peername")
        try:
            request = await read_request(reader)
            if request is None:
                return
            if request.method.upper() == "CONNECT":
                await self.handle_connect(request, reader, writer)
                return
            await self.handle_forward(request, reader, writer)
        except Exception as exc:
            self.log.warning("request failed from %s: %s", peer, exc)
            with contextlib.suppress(Exception):
                writer.write(b"HTTP/1.1 502 Bad Gateway\r\nConnection: close\r\nContent-Length: 0\r\n\r\n")
                await writer.drain()
        finally:
            with contextlib.suppress(Exception):
                writer.close()
                await writer.wait_closed()

    async def handle_connect(self, request: Request, client_reader: asyncio.StreamReader, client_writer: asyncio.StreamWriter) -> None:
        host, port = parse_host_port(request.target, 443)
        upstream_reader, upstream_writer = await self.open_upstream_connection(host, port)
        if self.get_upstream_node() is not None:
            await self.negotiate_upstream_connect(upstream_reader, upstream_writer, host, port)
        client_writer.write(f"{request.version} 200 Connection Established\r\n\r\n".encode("iso-8859-1"))
        await client_writer.drain()
        await tunnel(client_reader, client_writer, upstream_reader, upstream_writer)

    async def handle_forward(self, request: Request, client_reader: asyncio.StreamReader, client_writer: asyncio.StreamWriter) -> None:
        parsed = urlsplit(request.target)
        host = ""
        port = 80
        path = request.target
        if parsed.scheme and parsed.hostname:
            host = parsed.hostname
            if parsed.scheme in {"https", "wss"}:
                port = 443
            elif parsed.scheme in {"http", "ws"}:
                port = 80
            if parsed.port:
                port = parsed.port
            path = urlunsplit(("", "", parsed.path or "/", parsed.query, ""))
        else:
            host_header = request.raw_headers.get("host", "")
            if not host_header:
                raise ValueError("missing host header")
            host, port = parse_host_port(host_header, 80)
            path = request.target or "/"

        upstream_node = self.get_upstream_node()
        if upstream_node is None:
            upstream_reader, upstream_writer = await asyncio.open_connection(host, port)
        else:
            upstream_reader, upstream_writer = await asyncio.open_connection(upstream_node.host, upstream_node.port)

        connection_tokens = {
            token.strip().lower()
            for token in request.raw_headers.get("connection", "").split(",")
            if token.strip()
        }
        keep_upgrade = request.raw_headers.get("upgrade")
        headers_out = []
        saw_host = False
        for name, value in request.headers:
            lower = name.lower()
            if lower == "host":
                saw_host = True
                headers_out.append((name, format_host_port(host, port)))
                continue
            if lower in HOP_BY_HOP_HEADERS or lower in connection_tokens:
                if lower == "upgrade" and keep_upgrade:
                    headers_out.append((name, value))
                continue
            headers_out.append((name, value))
        if not saw_host:
            headers_out.append(("Host", format_host_port(host, port)))
        if keep_upgrade:
            headers_out.append(("Connection", "Upgrade"))
            headers_out.append(("Upgrade", keep_upgrade))
        else:
            headers_out.append(("Connection", "close"))

        header_blob = "".join(f"{name}: {value}\r\n" for name, value in headers_out).encode("iso-8859-1")
        request_target = path or "/"
        if upstream_node is not None:
            if parsed.scheme and parsed.hostname:
                request_target = request.target
            else:
                request_target = urlunsplit(("http", format_host_port(host, port), path or "/", "", ""))
        upstream_writer.write(f"{request.method} {request_target} {request.version}\r\n".encode("iso-8859-1"))
        upstream_writer.write(header_blob)
        upstream_writer.write(b"\r\n")
        await upstream_writer.drain()

        if "content-length" in request.raw_headers:
            await forward_fixed_body(client_reader, upstream_writer, int(request.raw_headers["content-length"]))
        elif request.raw_headers.get("transfer-encoding", "").lower() == "chunked":
            await forward_chunked_body(client_reader, upstream_writer)

        upgraded, _ = await relay_response(request.method, upstream_reader, client_writer)
        if upgraded:
            await tunnel(client_reader, client_writer, upstream_reader, upstream_writer)
            return
        with contextlib.suppress(Exception):
            upstream_writer.close()
            await upstream_writer.wait_closed()


async def watch_certificates(
    ssl_ctx: ssl.SSLContext,
    certfile: str,
    keyfile: str,
    interval: float,
    log: logging.Logger,
) -> None:
    last_state: tuple[int, int] | None = None
    while True:
        try:
            cert_mtime = os.path.getmtime(certfile)
            key_mtime = os.path.getmtime(keyfile)
            state = (int(cert_mtime), int(key_mtime))
            if state != last_state:
                ssl_ctx.load_cert_chain(certfile=certfile, keyfile=keyfile)
                if last_state is None:
                    log.info("certificate loaded from %s and %s", certfile, keyfile)
                else:
                    log.info("certificate reloaded from %s and %s", certfile, keyfile)
                last_state = state
        except FileNotFoundError:
            log.warning("certificate files not found: %s %s", certfile, keyfile)
        except Exception as exc:
            log.warning("certificate reload failed: %s", exc)
        await asyncio.sleep(interval)


async def main() -> None:
    root = Path(__file__).resolve().parent.parent
    dotenv = load_dotenv(root / ".env")
    parser = argparse.ArgumentParser()
    parser.add_argument("--instance-id", default=env_value("INSTANCE_ID", dotenv, "node-local"))
    parser.add_argument("--db-path", default=str(root / env_value("DB_PATH", dotenv, "runtime/proxy.db")))
    parser.add_argument("--http-host", default=env_value("HTTP_HOST", dotenv, "0.0.0.0"))
    parser.add_argument("--http-port", type=int, default=int(env_value("HTTP_PORT", dotenv, "18888")))
    parser.add_argument("--https-host", default=env_value("HTTPS_HOST", dotenv, "0.0.0.0"))
    parser.add_argument("--https-port", type=int, default=int(env_value("HTTPS_PORT", dotenv, "18889")))
    parser.add_argument("--certfile", default=str(root / env_value("CERT_FILE", dotenv, "certs/proxy.crt")))
    parser.add_argument("--keyfile", default=str(root / env_value("KEY_FILE", dotenv, "certs/proxy.key")))
    parser.add_argument("--cert-days", type=int, default=int(env_value("CERT_DAYS", dotenv, "365")))
    parser.add_argument("--cert-cn", default=env_value("CERT_CN", dotenv, "localhost"))
    parser.add_argument("--cert-reload-interval", type=float, default=float(env_value("CERT_RELOAD_INTERVAL", dotenv, "60")))
    args = parser.parse_args()

    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    config_store = ConfigStore(args.db_path)
    config_store.init_schema()
    config_store.bootstrap_local_instance(args.instance_id, args.http_host, args.http_port)
    server = ProxyServer(config_store=config_store, instance_id=args.instance_id)
    log = logging.getLogger("rootless_proxy")
    ensure_certificate(args.certfile, args.keyfile, args.cert_days, args.cert_cn)
    ssl_ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    ssl_ctx.load_cert_chain(certfile=args.certfile, keyfile=args.keyfile)

    http_server = await asyncio.start_server(server.handle, host=args.http_host, port=args.http_port)
    https_server = await asyncio.start_server(server.handle, host=args.https_host, port=args.https_port, ssl=ssl_ctx)

    for sock in http_server.sockets or []:
        logging.info("http proxy listening on %s", sock.getsockname())
    for sock in https_server.sockets or []:
        logging.info("https proxy listening on %s", sock.getsockname())

    async with http_server, https_server:
        await asyncio.gather(
            http_server.serve_forever(),
            https_server.serve_forever(),
            watch_certificates(
                ssl_ctx=ssl_ctx,
                certfile=args.certfile,
                keyfile=args.keyfile,
                interval=args.cert_reload_interval,
                log=log,
            ),
        )


if __name__ == "__main__":
    asyncio.run(main())
