"""Custom exceptions for PicTacticAgent."""


class PicTacticError(Exception):
    """Base exception for all PicTacticAgent errors."""

    def __init__(self, message: str, details: dict | None = None):
        super().__init__(message)
        self.message = message
        self.details = details or {}


class GenerationError(PicTacticError):
    """Error during image generation."""

    pass


class EvaluationError(PicTacticError):
    """Error during image evaluation."""

    pass


class ProviderError(PicTacticError):
    """Error from an external provider (Gemini, OpenAI, etc.)."""

    def __init__(self, message: str, provider: str, details: dict | None = None):
        super().__init__(message, details)
        self.provider = provider


class TemplateAnalysisError(PicTacticError):
    """Error during template image analysis."""

    pass


class StorageError(PicTacticError):
    """Error during storage operations."""

    pass


class TaskNotFoundError(PicTacticError):
    """Task not found in the system."""

    def __init__(self, task_id: str):
        super().__init__(f"Task not found: {task_id}")
        self.task_id = task_id


class ImageEditConflictError(PicTacticError):
    """Raised when an image is already being edited."""

    def __init__(self, image_id: str):
        super().__init__(f"Image is already being edited: {image_id}")
        self.image_id = image_id


class TemplateNotFoundError(PicTacticError):
    """Template not found."""

    def __init__(self, template_id: str):
        super().__init__(f"Template not found: {template_id}")
        self.template_id = template_id
