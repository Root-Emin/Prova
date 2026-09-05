"""Validated runtime configuration for the local persona service."""

from __future__ import annotations

import os
from dataclasses import dataclass


DEFAULT_MODEL = "Qwen/Qwen3-4B-Instruct-2507"
SUPPORTED_PROVIDERS = frozenset({"mock", "vllm"})

MAX_BODY_BYTES = 64 * 1024
MAX_ID_CHARS = 128
MAX_EMPLOYEE_MESSAGE_CHARS = 4_000
MAX_PERSONA_DESCRIPTION_CHARS = 2_000
MAX_SCENARIO_CONTEXT_CHARS = 4_000
MAX_LIST_ITEM_CHARS = 1_000
MAX_TURN_COUNT = 20
MAX_TURN_CONTENT_CHARS = 2_000
MAX_HISTORY_CHARS = 16_000


def _env(name: str, default: str) -> str:
    value = os.getenv(name)
    return value if value else default


def _float_env(name: str, default: float, minimum: float, maximum: float) -> float:
    raw = _env(name, str(default))
    try:
        value = float(raw)
    except ValueError as exc:
        raise ValueError(f"{name} must be a number") from exc
    if not minimum <= value <= maximum:
        raise ValueError(f"{name} must be between {minimum} and {maximum}")
    return value


def _int_env(name: str, default: int, minimum: int, maximum: int) -> int:
    raw = _env(name, str(default))
    try:
        value = int(raw)
    except ValueError as exc:
        raise ValueError(f"{name} must be an integer") from exc
    if not minimum <= value <= maximum:
        raise ValueError(f"{name} must be between {minimum} and {maximum}")
    return value


@dataclass(frozen=True)
class Settings:
    host: str
    port: int
    provider: str
    base_url: str
    model: str
    api_key: str
    timeout_seconds: float
    temperature: float
    top_p: float
    max_tokens: int

    @classmethod
    def from_env(cls) -> "Settings":
        provider = _env("AI_PERSONA_PROVIDER", "mock").strip().lower()
        if provider not in SUPPORTED_PROVIDERS:
            raise ValueError("AI_PERSONA_PROVIDER must be mock or vllm")
        return cls(
            host=_env("PERSONA_SERVICE_HOST", "127.0.0.1"),
            port=_int_env("PERSONA_SERVICE_PORT", 8090, 1, 65535),
            provider=provider,
            base_url=_env("PERSONA_LLM_BASE_URL", "http://localhost:8000/v1").rstrip("/"),
            model=_env("PERSONA_LLM_MODEL", DEFAULT_MODEL),
            api_key=os.getenv("PERSONA_LLM_API_KEY", ""),
            timeout_seconds=_float_env("PERSONA_LLM_TIMEOUT", 10.0, 0.1, 120.0),
            temperature=_float_env("PERSONA_TEMPERATURE", 0.2, 0.0, 2.0),
            top_p=_float_env("PERSONA_TOP_P", 0.9, 0.0, 1.0),
            max_tokens=_int_env("PERSONA_MAX_TOKENS", 256, 1, 4096),
        )
