import sqlite3
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path


SCHEMA_STATEMENTS = [
    """
    CREATE TABLE IF NOT EXISTS instances (
      id TEXT PRIMARY KEY,
      node_id TEXT NOT NULL,
      name TEXT NOT NULL,
      enabled INTEGER NOT NULL DEFAULT 1,
      created_at TEXT NOT NULL,
      updated_at TEXT NOT NULL
    )
    """,
    """
    CREATE TABLE IF NOT EXISTS nodes (
      id TEXT PRIMARY KEY,
      name TEXT NOT NULL,
      protocol TEXT NOT NULL,
      host TEXT NOT NULL,
      port INTEGER NOT NULL,
      username TEXT,
      password TEXT,
      tls_enabled INTEGER NOT NULL DEFAULT 0,
      connect_timeout_ms INTEGER NOT NULL DEFAULT 5000,
      read_timeout_ms INTEGER NOT NULL DEFAULT 30000,
      enabled INTEGER NOT NULL DEFAULT 1,
      created_at TEXT NOT NULL,
      updated_at TEXT NOT NULL
    )
    """,
    """
    CREATE TABLE IF NOT EXISTS node_upstreams (
      node_id TEXT PRIMARY KEY,
      upstream_node_id TEXT NOT NULL,
      mode TEXT NOT NULL,
      created_at TEXT NOT NULL,
      updated_at TEXT NOT NULL
    )
    """,
    """
    CREATE TABLE IF NOT EXISTS chains (
      id TEXT PRIMARY KEY,
      name TEXT NOT NULL,
      dns_policy TEXT NOT NULL,
      failure_policy TEXT NOT NULL,
      enabled INTEGER NOT NULL DEFAULT 1,
      created_at TEXT NOT NULL,
      updated_at TEXT NOT NULL
    )
    """,
    """
    CREATE TABLE IF NOT EXISTS chain_hops (
      chain_id TEXT NOT NULL,
      hop_index INTEGER NOT NULL,
      node_id TEXT NOT NULL,
      PRIMARY KEY (chain_id, hop_index)
    )
    """,
    """
    CREATE TABLE IF NOT EXISTS route_rules (
      id TEXT PRIMARY KEY,
      priority INTEGER NOT NULL,
      match_type TEXT NOT NULL,
      match_value TEXT NOT NULL,
      action_type TEXT NOT NULL,
      action_value TEXT,
      enabled INTEGER NOT NULL DEFAULT 1,
      created_at TEXT NOT NULL,
      updated_at TEXT NOT NULL
    )
    """,
    """
    CREATE TABLE IF NOT EXISTS runtime_config (
      key TEXT PRIMARY KEY,
      value TEXT NOT NULL,
      updated_at TEXT NOT NULL
    )
    """,
    """
    CREATE TABLE IF NOT EXISTS request_logs (
      id TEXT PRIMARY KEY,
      request_id TEXT NOT NULL,
      instance_id TEXT NOT NULL,
      chain_id TEXT,
      target_host TEXT,
      target_port INTEGER,
      method TEXT,
      upstream_node_id TEXT,
      status TEXT NOT NULL,
      error_message TEXT,
      created_at TEXT NOT NULL
    )
    """,
    """
    CREATE INDEX IF NOT EXISTS idx_route_rules_priority
      ON route_rules (enabled, priority)
    """,
    """
    CREATE INDEX IF NOT EXISTS idx_chain_hops_node
      ON chain_hops (node_id)
    """,
    """
    CREATE INDEX IF NOT EXISTS idx_request_logs_request_id
      ON request_logs (request_id)
    """,
]


@dataclass(frozen=True)
class Node:
    id: str
    name: str
    protocol: str
    host: str
    port: int
    username: str | None
    password: str | None
    tls_enabled: bool
    connect_timeout_ms: int
    read_timeout_ms: int
    enabled: bool


def utc_now() -> str:
    return datetime.now(timezone.utc).isoformat()


def row_to_node(row: sqlite3.Row) -> Node:
    return Node(
        id=row["id"],
        name=row["name"],
        protocol=row["protocol"],
        host=row["host"],
        port=int(row["port"]),
        username=row["username"],
        password=row["password"],
        tls_enabled=bool(row["tls_enabled"]),
        connect_timeout_ms=int(row["connect_timeout_ms"]),
        read_timeout_ms=int(row["read_timeout_ms"]),
        enabled=bool(row["enabled"]),
    )


class ConfigStore:
    def __init__(self, db_path: str) -> None:
        self.db_path = Path(db_path)
        self.db_path.parent.mkdir(parents=True, exist_ok=True)

    def connect(self) -> sqlite3.Connection:
        conn = sqlite3.connect(self.db_path)
        conn.row_factory = sqlite3.Row
        return conn

    def init_schema(self) -> None:
        with self.connect() as conn:
            for statement in SCHEMA_STATEMENTS:
                conn.execute(statement)
            conn.commit()

    def bootstrap_local_instance(self, instance_id: str, host: str, port: int) -> None:
        now = utc_now()
        with self.connect() as conn:
            node = conn.execute(
                "SELECT id FROM nodes WHERE id = ?",
                (instance_id,),
            ).fetchone()
            if node is None:
                conn.execute(
                    """
                    INSERT INTO nodes (
                      id, name, protocol, host, port, username, password,
                      tls_enabled, connect_timeout_ms, read_timeout_ms,
                      enabled, created_at, updated_at
                    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                    """,
                    (
                        instance_id,
                        instance_id,
                        "http_proxy",
                        host,
                        port,
                        None,
                        None,
                        0,
                        5000,
                        30000,
                        1,
                        now,
                        now,
                    ),
                )
            instance = conn.execute(
                "SELECT id FROM instances WHERE id = ?",
                (instance_id,),
            ).fetchone()
            if instance is None:
                conn.execute(
                    """
                    INSERT INTO instances (
                      id, node_id, name, enabled, created_at, updated_at
                    ) VALUES (?, ?, ?, ?, ?, ?)
                    """,
                    (instance_id, instance_id, instance_id, 1, now, now),
                )
            conn.commit()

    def get_local_node(self, instance_id: str) -> Node:
        with self.connect() as conn:
            row = conn.execute(
                """
                SELECT n.*
                FROM instances i
                JOIN nodes n ON n.id = i.node_id
                WHERE i.id = ? AND i.enabled = 1 AND n.enabled = 1
                """,
                (instance_id,),
            ).fetchone()
        if row is None:
            raise ValueError(f"instance not found or disabled: {instance_id}")
        return row_to_node(row)

    def get_upstream_node(self, node_id: str) -> Node | None:
        with self.connect() as conn:
            row = conn.execute(
                """
                SELECT n.*
                FROM node_upstreams u
                JOIN nodes n ON n.id = u.upstream_node_id
                WHERE u.node_id = ? AND n.enabled = 1
                """,
                (node_id,),
            ).fetchone()
        if row is None:
            return None
        return row_to_node(row)
