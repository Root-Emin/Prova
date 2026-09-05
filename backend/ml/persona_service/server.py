"""Small dependency-free HTTP server for AI-01 local development and tests."""

from __future__ import annotations

import argparse
import json
import logging
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

try:
    from .errors import PersonaServiceError
    from .providers import create_provider
    from .service import PersonaService
    from .settings import MAX_BODY_BYTES, Settings
except ImportError:  # Direct `python backend/ml/persona_service/server.py` support.
    sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
    from persona_service.errors import PersonaServiceError
    from persona_service.providers import create_provider
    from persona_service.service import PersonaService
    from persona_service.settings import MAX_BODY_BYTES, Settings

LOGGER = logging.getLogger("prova.persona_service")


def create_server(settings: Settings) -> ThreadingHTTPServer:
    service = PersonaService(create_provider(settings))

    class Handler(BaseHTTPRequestHandler):
        server_version = "ProvaPersona/1"

        def _write(self, status: int, payload: dict) -> None:
            encoded = json.dumps(payload, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
            self.send_response(status)
            self.send_header("Content-Type", "application/json; charset=utf-8")
            self.send_header("Content-Length", str(len(encoded)))
            self.end_headers()
            self.wfile.write(encoded)

        def do_GET(self) -> None:  # noqa: N802
            if self.path == "/health":
                self._write(200, {"status": "ok"})
                return
            if self.path == "/ready":
                ready = service.ready()
                self._write(200 if ready else 503, {"status": "ready" if ready else "not_ready", "provider": service.provider.name})
                return
            self._write(404, {"error": {"code": "not_found", "message": "not found"}})

        def do_POST(self) -> None:  # noqa: N802
            if self.path != "/v1/persona/respond":
                self._write(404, {"error": {"code": "not_found", "message": "not found"}})
                return
            try:
                content_length = int(self.headers.get("Content-Length", "-1"))
            except ValueError:
                content_length = -1
            if content_length < 0 or content_length > MAX_BODY_BYTES:
                self._write(413, {"error": {"code": "validation_error", "message": "request body is too large"}})
                return
            try:
                payload = json.loads(self.rfile.read(content_length).decode("utf-8"))
                response = service.respond(payload)
                self._write(200, response.to_dict())
            except PersonaServiceError as exc:
                self._write(exc.status, {"error": {"code": exc.code, "message": exc.message}})
            except (UnicodeDecodeError, json.JSONDecodeError):
                self._write(422, {"error": {"code": "validation_error", "message": "request body must be valid JSON"}})
            except Exception:
                LOGGER.exception("persona request failed")
                self._write(500, {"error": {"code": "internal_error", "message": "internal server error"}})

        def log_message(self, format: str, *args: object) -> None:
            # Deliberately log only method/path/status; request and response bodies are never logged.
            LOGGER.info("http_request", extra={"request": format % args})

    return ThreadingHTTPServer((settings.host, settings.port), Handler)


def main() -> None:
    parser = argparse.ArgumentParser(description="Prova AI-01 persona inference service")
    parser.add_argument("--host", default=None)
    parser.add_argument("--port", type=int, default=None)
    args = parser.parse_args()
    settings = Settings.from_env()
    if args.host is not None or args.port is not None:
        settings = Settings(**{**settings.__dict__, "host": args.host or settings.host, "port": args.port or settings.port})
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    server = create_server(settings)
    LOGGER.info("persona service listening on %s:%s provider=%s", settings.host, settings.port, settings.provider)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()


if __name__ == "__main__":
    main()
