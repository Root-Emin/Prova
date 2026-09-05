"""Safe, client-facing errors for the persona service."""


class PersonaServiceError(Exception):
    code = "internal_error"
    status = 500

    def __init__(self, message: str):
        super().__init__(message)
        self.message = message


class ValidationError(PersonaServiceError):
    code = "validation_error"
    status = 422


class ProviderUnavailableError(PersonaServiceError):
    code = "provider_unavailable"
    status = 503


class ProviderTimeoutError(PersonaServiceError):
    code = "provider_timeout"
    status = 504


class InvalidModelResponseError(PersonaServiceError):
    code = "invalid_model_response"
    status = 502


class InternalError(PersonaServiceError):
    code = "internal_error"
    status = 500
