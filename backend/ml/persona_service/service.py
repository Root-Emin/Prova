"""Application service for validated persona inference."""

from __future__ import annotations

import time

from .contract import PersonaRequest, PersonaResponse
from .prompt import PromptBuilder
from .providers import PersonaProvider


class PersonaService:
    def __init__(self, provider: PersonaProvider, prompt_builder: PromptBuilder | None = None):
        self.provider = provider
        self.prompt_builder = prompt_builder or PromptBuilder()

    def ready(self) -> bool:
        return self.provider.ready()

    def respond(self, payload: object) -> PersonaResponse:
        request = PersonaRequest.from_dict(payload)
        messages = self.prompt_builder.build(request)
        started = time.monotonic()
        response = self.provider.respond(request, messages)
        validated = PersonaResponse.from_dict(response.to_dict() if isinstance(response, PersonaResponse) else response)
        metadata = dict(validated.metadata)
        metadata.update(
            {
                "provider": self.provider.name,
                "model": self.provider.model,
                "latency_ms": max(0, round((time.monotonic() - started) * 1000)),
            }
        )
        return PersonaResponse(validated.text, validated.emotion, validated.conversation_state, metadata)
