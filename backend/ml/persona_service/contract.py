"""Versioned internal request/response contract for persona inference."""

from __future__ import annotations

import json
from dataclasses import dataclass, field
from typing import Any

from .errors import InvalidModelResponseError, ValidationError
from .settings import (
    MAX_EMPLOYEE_MESSAGE_CHARS,
    MAX_HISTORY_CHARS,
    MAX_ID_CHARS,
    MAX_LIST_ITEM_CHARS,
    MAX_PERSONA_DESCRIPTION_CHARS,
    MAX_SCENARIO_CONTEXT_CHARS,
    MAX_TURN_CONTENT_CHARS,
    MAX_TURN_COUNT,
)

EMOTIONS = frozenset({"neutral", "calm", "concerned", "frustrated", "angry", "relieved", "cooperative"})
CONVERSATION_STATES = frozenset({"active", "resolved", "needs_clarification", "escalating", "ended"})
HISTORY_ROLES = frozenset({"employee", "persona"})


def _object(value: Any, field_name: str) -> dict[str, Any]:
    if not isinstance(value, dict):
        raise ValidationError(f"{field_name} must be an object")
    return value


def _string(value: Any, field_name: str, *, required: bool = False, max_chars: int = MAX_ID_CHARS) -> str:
    if value is None and not required:
        return ""
    if not isinstance(value, str):
        raise ValidationError(f"{field_name} must be a string")
    value = value.strip()
    if required and not value:
        raise ValidationError(f"{field_name} must not be empty")
    if len(value) > max_chars:
        raise ValidationError(f"{field_name} exceeds the maximum length")
    return value


def _string_list(value: Any, field_name: str) -> list[str]:
    if value is None:
        return []
    if not isinstance(value, list):
        raise ValidationError(f"{field_name} must be an array")
    result = []
    for index, item in enumerate(value):
        result.append(_string(item, f"{field_name}[{index}]", required=True, max_chars=MAX_LIST_ITEM_CHARS))
    return result


@dataclass(frozen=True)
class ConversationTurn:
    role: str
    content: str

    @classmethod
    def from_dict(cls, value: Any, index: int) -> "ConversationTurn":
        item = _object(value, f"conversation_history[{index}]")
        role = _string(item.get("role"), f"conversation_history[{index}].role", required=True, max_chars=32)
        if role not in HISTORY_ROLES:
            raise ValidationError(f"conversation_history[{index}].role is invalid")
        content = _string(
            item.get("content"), f"conversation_history[{index}].content", required=True, max_chars=MAX_TURN_CONTENT_CHARS
        )
        return cls(role=role, content=content)


@dataclass(frozen=True)
class PersonaData:
    id: str
    name: str
    role: str
    description: str
    behavioral_traits: tuple[str, ...] = ()
    current_state: dict[str, Any] = field(default_factory=dict)


@dataclass(frozen=True)
class ScenarioData:
    id: str
    title: str
    context: str
    persona_goal: str
    allowed_knowledge: tuple[str, ...] = ()
    behavioral_rules: tuple[str, ...] = ()


@dataclass(frozen=True)
class PersonaRequest:
    request_id: str
    session_id: str
    scenario_id: str
    persona: PersonaData
    scenario: ScenarioData
    conversation_history: tuple[ConversationTurn, ...]
    employee_message: str

    @classmethod
    def from_dict(cls, value: Any) -> "PersonaRequest":
        root = _object(value, "request")
        persona_raw = _object(root.get("persona"), "persona")
        scenario_raw = _object(root.get("scenario"), "scenario")
        history_raw = root.get("conversation_history", [])
        if not isinstance(history_raw, list):
            raise ValidationError("conversation_history must be an array")
        if len(history_raw) > MAX_TURN_COUNT:
            raise ValidationError("conversation_history exceeds the maximum turn count")
        history = tuple(ConversationTurn.from_dict(item, index) for index, item in enumerate(history_raw))
        history_chars = sum(len(turn.content) for turn in history)
        if history_chars > MAX_HISTORY_CHARS:
            raise ValidationError("conversation_history exceeds the maximum size")

        current_state = persona_raw.get("current_state", {})
        if not isinstance(current_state, dict):
            raise ValidationError("persona.current_state must be an object")
        persona = PersonaData(
            id=_string(persona_raw.get("id"), "persona.id"),
            name=_string(persona_raw.get("name"), "persona.name", required=True),
            role=_string(persona_raw.get("role"), "persona.role", required=True),
            description=_string(
                persona_raw.get("description"), "persona.description", required=True, max_chars=MAX_PERSONA_DESCRIPTION_CHARS
            ),
            behavioral_traits=tuple(_string_list(persona_raw.get("behavioral_traits"), "persona.behavioral_traits")),
            current_state=current_state,
        )
        scenario = ScenarioData(
            id=_string(scenario_raw.get("id"), "scenario.id"),
            title=_string(scenario_raw.get("title"), "scenario.title", required=True),
            context=_string(scenario_raw.get("context"), "scenario.context", required=True, max_chars=MAX_SCENARIO_CONTEXT_CHARS),
            persona_goal=_string(scenario_raw.get("persona_goal"), "scenario.persona_goal"),
            allowed_knowledge=tuple(_string_list(scenario_raw.get("allowed_knowledge"), "scenario.allowed_knowledge")),
            behavioral_rules=tuple(_string_list(scenario_raw.get("behavioral_rules"), "scenario.behavioral_rules")),
        )
        employee_message = _string(
            root.get("employee_message"), "employee_message", required=True, max_chars=MAX_EMPLOYEE_MESSAGE_CHARS
        )
        return cls(
            request_id=_string(root.get("request_id"), "request_id"),
            session_id=_string(root.get("session_id"), "session_id"),
            scenario_id=_string(root.get("scenario_id"), "scenario_id"),
            persona=persona,
            scenario=scenario,
            conversation_history=history,
            employee_message=employee_message,
        )


@dataclass(frozen=True)
class PersonaResponse:
    text: str
    emotion: str
    conversation_state: str
    metadata: dict[str, Any] = field(default_factory=dict)

    @classmethod
    def from_dict(cls, value: Any) -> "PersonaResponse":
        if not isinstance(value, dict):
            raise InvalidModelResponseError("model response must be an object")
        text = value.get("text")
        if not isinstance(text, str) or not text.strip():
            raise InvalidModelResponseError("model response text is empty")
        emotion = value.get("emotion")
        if emotion not in EMOTIONS:
            raise InvalidModelResponseError("model response emotion is invalid")
        state = value.get("conversation_state")
        if state not in CONVERSATION_STATES:
            raise InvalidModelResponseError("model response conversation_state is invalid")
        metadata = value.get("metadata", {})
        if not isinstance(metadata, dict):
            raise InvalidModelResponseError("model response metadata is invalid")
        return cls(text=text.strip(), emotion=emotion, conversation_state=state, metadata=metadata)

    def to_dict(self) -> dict[str, Any]:
        result = {"text": self.text, "emotion": self.emotion, "conversation_state": self.conversation_state}
        if self.metadata:
            result["metadata"] = self.metadata
        return result


def parse_provider_response(raw: str) -> PersonaResponse:
    if not isinstance(raw, str) or not raw.strip():
        raise InvalidModelResponseError("model response is empty")
    try:
        decoder = json.JSONDecoder()
        payload, end = decoder.raw_decode(raw.lstrip())
        if raw.lstrip()[end:].strip():
            raise InvalidModelResponseError("model response contains extra text")
    except (json.JSONDecodeError, TypeError) as exc:
        raise InvalidModelResponseError("model response is not valid JSON") from exc
    return PersonaResponse.from_dict(payload)
