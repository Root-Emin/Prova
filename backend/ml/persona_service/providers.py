"""Persona provider abstraction and mock/vLLM implementations."""

from __future__ import annotations

import json
import urllib.error
import urllib.request
from typing import Protocol

from .contract import PersonaRequest, PersonaResponse
from .errors import InvalidModelResponseError, ProviderTimeoutError, ProviderUnavailableError
from .settings import Settings


class PersonaProvider(Protocol):
    name: str
    model: str

    def ready(self) -> bool: ...

    def respond(self, request: PersonaRequest, messages: list[dict[str, str]]) -> PersonaResponse: ...


class MockPersonaProvider:
    name = "mock"
    model = "mock-persona-v1"

    def ready(self) -> bool:
        return True

    def respond(self, request: PersonaRequest, messages: list[dict[str, str]]) -> PersonaResponse:
        message = request.employee_message.casefold()
        if any(word in message for word in ("kimlik", "doğrula", "dogrula")):
            return PersonaResponse(
                text="Bu bilgiyi paylaşmadan önce kimliğinizi doğrulamamız gerekiyor.",
                emotion="concerned",
                conversation_state="needs_clarification",
            )
        if any(word in message for word in ("teşekkür", "tesekkur", "anlıyorum", "anliyorum")):
            return PersonaResponse(
                text="Teşekkür ederim, yaklaşımınızı anlıyorum. Konuyu birlikte netleştirebiliriz.",
                emotion="cooperative",
                conversation_state="active",
            )
        return PersonaResponse(
            text="Talebinizi anladım. Konuyu biraz daha açıklar mısınız?",
            emotion="neutral",
            conversation_state="needs_clarification",
        )


class VLLMPersonaProvider:
    name = "vllm"

    def __init__(self, settings: Settings):
        self.base_url = settings.base_url
        self.model = settings.model
        self.api_key = settings.api_key
        self.timeout_seconds = settings.timeout_seconds
        self.temperature = settings.temperature
        self.top_p = settings.top_p
        self.max_tokens = settings.max_tokens

    def _request(self, method: str, path: str, payload: dict | None = None, timeout_seconds: float | None = None) -> bytes:
        body = None
        headers = {"Accept": "application/json"}
        if payload is not None:
            body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
            headers["Content-Type"] = "application/json"
        if self.api_key:
            headers["Authorization"] = f"Bearer {self.api_key}"
        request = urllib.request.Request(f"{self.base_url}{path}", data=body, headers=headers, method=method)
        try:
            with urllib.request.urlopen(request, timeout=timeout_seconds or self.timeout_seconds) as response:
                return response.read(256 * 1024)
        except urllib.error.HTTPError as exc:
            if exc.code in (408, 504):
                raise ProviderTimeoutError("persona model timed out") from exc
            raise ProviderUnavailableError("persona model is unavailable") from exc
        except (TimeoutError, urllib.error.URLError) as exc:
            reason = getattr(exc, "reason", exc)
            if isinstance(reason, TimeoutError) or "timed out" in str(reason).lower():
                raise ProviderTimeoutError("persona model timed out") from exc
            raise ProviderUnavailableError("persona model is unavailable") from exc

    def ready(self) -> bool:
        try:
            self._request("GET", "/models", timeout_seconds=min(self.timeout_seconds, 1.0))
            return True
        except (ProviderUnavailableError, ProviderTimeoutError):
            return False

    def respond(self, request: PersonaRequest, messages: list[dict[str, str]]) -> PersonaResponse:
        payload = {
            "model": self.model,
            "messages": messages,
            "temperature": self.temperature,
            "top_p": self.top_p,
            "max_tokens": self.max_tokens,
            "response_format": {"type": "json_object"},
            "chat_template_kwargs": {"enable_thinking": False},
        }
        raw = self._request("POST", "/chat/completions", payload)
        try:
            envelope = json.loads(raw.decode("utf-8"))
            content = envelope["choices"][0]["message"]["content"]
        except (UnicodeDecodeError, json.JSONDecodeError, KeyError, IndexError, TypeError) as exc:
            raise InvalidModelResponseError("persona model returned an invalid envelope") from exc
        from .contract import parse_provider_response

        return parse_provider_response(content)


def create_provider(settings: Settings) -> PersonaProvider:
    if settings.provider == "mock":
        return MockPersonaProvider()
    return VLLMPersonaProvider(settings)
