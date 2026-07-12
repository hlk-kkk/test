"""Tests for image editing functionality in GenerationService."""

from datetime import datetime
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from pictactic.core.exceptions import PicTacticError, ProviderError, TaskNotFoundError
from pictactic.models.enums import GenerationStatus, ImageProvider
from pictactic.models.schemas import (
    GeneratedImage,
    GenerationRequest,
    ImageEditRequest,
    ImageEditResult,
)
from pictactic.providers.base import GenerationResult as ProviderResult
from pictactic.services.generation_service import GenerationService
from pydantic import ValidationError


def _build_service_with_task(
    task_id: str = "task-1",
    user_id: str | None = "user-123",
    images: list[dict] | None = None,
) -> GenerationService:
    """Create a GenerationService with a pre-populated task."""
    service = GenerationService()
    request = GenerationRequest(
        prompt="test prompt",
        max_rounds=3,
        images_per_round=5,
        provider=ImageProvider.GEMINI,
    )
    now = datetime.utcnow()

    default_images = images or [
        {
            "id": "img-1",
            "url": "https://example.com/img1.png",
            "local_path": "/tmp/img1.png",
            "score": 0.8,
            "review": "good",
            "round_number": 1,
            "provider": "gemini",
            "metadata": {},
        },
        {
            "id": "img-2",
            "url": "https://example.com/img2.png",
            "local_path": None,
            "score": 0.6,
            "review": "ok",
            "round_number": 1,
            "provider": "gemini",
            "metadata": {},
        },
    ]

    service._tasks[task_id] = {
        "id": task_id,
        "user_id": user_id,
        "status": GenerationStatus.COMPLETED,
        "request": request,
        "state": {
            "messages": [],
            "user_prompt": "test prompt",
            "enhanced_prompt": "enhanced test prompt",
            "generated_images": default_images,
            "best_images": [default_images[0]] if default_images else [],
            "current_round": 1,
        },
        "created_at": now,
        "updated_at": now,
        "result": None,
        "error": None,
    }
    return service


# Patch targets for lazy imports inside edit_image / _enhance_edit_prompt
_WS_PATCH = "pictactic.api.websocket.get_connection_manager"
_NORMALIZE_PATCH = "pictactic.agents.tools._image_utils.normalize_reference_images"
_PROVIDER_PATCH = "pictactic.providers.get_provider"
_LLM_PATCH = "pictactic.core.config.create_vision_llm"


class TestImageEditRequest:
    """Tests for ImageEditRequest model."""

    def test_valid_request(self):
        """Test creating a valid edit request."""
        req = ImageEditRequest(edit_prompt="Make the background blue")
        assert req.edit_prompt == "Make the background blue"
        assert req.enhance_prompt is True

    def test_request_with_enhance_disabled(self):
        """Test edit request with prompt enhancement disabled."""
        req = ImageEditRequest(edit_prompt="Change text", enhance_prompt=False)
        assert req.enhance_prompt is False

    def test_request_requires_prompt(self):
        """Test that edit_prompt is required and non-empty."""
        with pytest.raises(ValidationError):
            ImageEditRequest(edit_prompt="")

    def test_request_missing_prompt(self):
        """Test that edit_prompt field is required."""
        with pytest.raises(ValidationError):
            ImageEditRequest()


class TestImageEditResult:
    """Tests for ImageEditResult model."""

    def test_create_result(self):
        """Test creating an edit result."""
        image = GeneratedImage(
            id="new-img-1",
            url="https://example.com/edited.png",
            prompt="edited prompt",
            provider=ImageProvider.GEMINI,
            round_number=1,
            score=0.0,
        )
        result = ImageEditResult(
            original_image_id="img-1",
            new_image=image,
            edit_prompt="Make it blue",
            enhanced_edit_prompt="Make the background a vibrant blue color",
        )
        assert result.original_image_id == "img-1"
        assert result.new_image.id == "new-img-1"
        assert result.enhanced_edit_prompt is not None

    def test_result_without_enhanced_prompt(self):
        """Test result when prompt enhancement was disabled."""
        image = GeneratedImage(
            id="new-img-1",
            url="https://example.com/edited.png",
            prompt="edited prompt",
            provider=ImageProvider.GEMINI,
            round_number=1,
        )
        result = ImageEditResult(
            original_image_id="img-1",
            new_image=image,
            edit_prompt="Make it blue",
        )
        assert result.enhanced_edit_prompt is None


class TestEditImageService:
    """Tests for GenerationService.edit_image method."""

    @pytest.mark.asyncio
    async def test_edit_image_success(self):
        """Test successful image edit."""
        service = _build_service_with_task()

        mock_provider_result = ProviderResult(
            url="https://example.com/edited.png",
            local_path="/tmp/edited.png",
            metadata={"provider": "gemini-3-pro-image"},
        )

        with patch(_WS_PATCH) as mock_ws:
            mock_ws.return_value.send_progress = AsyncMock()

            with patch(_NORMALIZE_PATCH, new_callable=AsyncMock) as mock_normalize:
                mock_normalize.return_value = ["data:image/png;base64,abc123"]

                with patch(_PROVIDER_PATCH) as mock_get_provider:
                    mock_provider = MagicMock()
                    mock_provider.generate = AsyncMock(return_value=mock_provider_result)
                    mock_get_provider.return_value = mock_provider

                    edit_request = ImageEditRequest(
                        edit_prompt="Make the background blue",
                        enhance_prompt=False,
                    )
                    result = await service.edit_image(
                        task_id="task-1",
                        image_id="img-1",
                        edit_request=edit_request,
                        user_id="user-123",
                    )

                    assert isinstance(result, ImageEditResult)
                    assert result.original_image_id == "img-1"
                    assert result.new_image.url == "https://example.com/edited.png"
                    assert result.edit_prompt == "Make the background blue"
                    assert result.enhanced_edit_prompt is None
                    assert result.new_image.metadata["edited_from"] == "img-1"

    @pytest.mark.asyncio
    async def test_edit_image_with_prompt_enhancement(self):
        """Test edit with prompt enhancement enabled."""
        service = _build_service_with_task()

        mock_provider_result = ProviderResult(
            url="https://example.com/edited.png",
            metadata={"provider": "gemini-3-pro-image"},
        )

        with patch(_WS_PATCH) as mock_ws:
            mock_ws.return_value.send_progress = AsyncMock()

            with patch.object(
                service,
                "_enhance_edit_prompt",
                new_callable=AsyncMock,
                return_value="Enhanced: make the background a vibrant blue sky",
            ), patch(_NORMALIZE_PATCH, new_callable=AsyncMock) as mock_normalize:
                mock_normalize.return_value = ["/tmp/img1.png"]

                with patch(_PROVIDER_PATCH) as mock_get_provider:
                    mock_provider = MagicMock()
                    mock_provider.generate = AsyncMock(return_value=mock_provider_result)
                    mock_get_provider.return_value = mock_provider

                    edit_request = ImageEditRequest(
                        edit_prompt="Make the background blue",
                        enhance_prompt=True,
                    )
                    result = await service.edit_image(
                        task_id="task-1",
                        image_id="img-1",
                        edit_request=edit_request,
                        user_id="user-123",
                    )

                    assert result.enhanced_edit_prompt == (
                        "Enhanced: make the background a vibrant blue sky"
                    )
                    # Provider should receive the enhanced prompt
                    call_args = mock_provider.generate.call_args
                    params = call_args[0][0]
                    assert params.prompt == ("Enhanced: make the background a vibrant blue sky")

    @pytest.mark.asyncio
    async def test_edit_image_task_not_found(self):
        """Test edit with non-existent task ID."""
        service = _build_service_with_task()

        edit_request = ImageEditRequest(edit_prompt="Edit this")
        with pytest.raises(TaskNotFoundError):
            await service.edit_image(
                task_id="nonexistent-task",
                image_id="img-1",
                edit_request=edit_request,
            )

    @pytest.mark.asyncio
    async def test_edit_image_wrong_user(self):
        """Test edit with wrong user ID raises TaskNotFoundError."""
        service = _build_service_with_task(user_id="user-123")

        edit_request = ImageEditRequest(edit_prompt="Edit this")
        with pytest.raises(TaskNotFoundError):
            await service.edit_image(
                task_id="task-1",
                image_id="img-1",
                edit_request=edit_request,
                user_id="wrong-user",
            )

    @pytest.mark.asyncio
    async def test_edit_image_not_found(self):
        """Test edit with non-existent image ID."""
        service = _build_service_with_task()

        with patch(_WS_PATCH) as mock_ws:
            mock_ws.return_value.send_progress = AsyncMock()

            edit_request = ImageEditRequest(edit_prompt="Edit this")
            with pytest.raises(PicTacticError, match="Image not found"):
                await service.edit_image(
                    task_id="task-1",
                    image_id="nonexistent-img",
                    edit_request=edit_request,
                    user_id="user-123",
                )

    @pytest.mark.asyncio
    async def test_edit_image_no_source(self):
        """Test edit when image has no URL or local_path."""
        service = _build_service_with_task(
            images=[
                {
                    "id": "img-no-source",
                    "url": "",
                    "local_path": None,
                    "score": 0.5,
                    "review": "",
                    "round_number": 1,
                    "provider": "gemini",
                    "metadata": {},
                }
            ]
        )

        with patch(_WS_PATCH) as mock_ws:
            mock_ws.return_value.send_progress = AsyncMock()

            edit_request = ImageEditRequest(edit_prompt="Edit this")
            with pytest.raises(PicTacticError, match="Image source not available"):
                await service.edit_image(
                    task_id="task-1",
                    image_id="img-no-source",
                    edit_request=edit_request,
                    user_id="user-123",
                )

    @pytest.mark.asyncio
    async def test_edit_image_websocket_events(self):
        """Test that WebSocket events are sent during edit."""
        service = _build_service_with_task()

        mock_provider_result = ProviderResult(
            url="https://example.com/edited.png",
            metadata={"provider": "gemini-3-pro-image"},
        )

        with patch(_WS_PATCH) as mock_ws:
            mock_send = AsyncMock()
            mock_ws.return_value.send_progress = mock_send

            with patch(_NORMALIZE_PATCH, new_callable=AsyncMock) as mock_normalize:
                mock_normalize.return_value = ["/tmp/img1.png"]

                with patch(_PROVIDER_PATCH) as mock_get_provider:
                    mock_provider = MagicMock()
                    mock_provider.generate = AsyncMock(return_value=mock_provider_result)
                    mock_get_provider.return_value = mock_provider

                    edit_request = ImageEditRequest(edit_prompt="Edit", enhance_prompt=False)
                    await service.edit_image(
                        task_id="task-1",
                        image_id="img-1",
                        edit_request=edit_request,
                        user_id="user-123",
                    )

                    # Should have sent 2 WebSocket events: start and complete
                    assert mock_send.call_count == 2

                    start_call = mock_send.call_args_list[0]
                    assert start_call[0][0] == "task-1"
                    assert start_call[0][1]["type"] == "image_edit_start"
                    assert start_call[0][1]["image_id"] == "img-1"

                    complete_call = mock_send.call_args_list[1]
                    assert complete_call[0][0] == "task-1"
                    assert complete_call[0][1]["type"] == "image_edit_complete"
                    assert complete_call[0][1]["image_id"] == "img-1"
                    assert "new_image" in complete_call[0][1]

    @pytest.mark.asyncio
    async def test_edit_image_adds_to_generated_images(self):
        """Test that edited image is added to task's generated_images."""
        service = _build_service_with_task()

        mock_provider_result = ProviderResult(
            url="https://example.com/edited.png",
            metadata={"provider": "gemini-3-pro-image"},
        )

        with patch(_WS_PATCH) as mock_ws:
            mock_ws.return_value.send_progress = AsyncMock()

            with patch(_NORMALIZE_PATCH, new_callable=AsyncMock) as mock_normalize:
                mock_normalize.return_value = ["/tmp/img1.png"]

                with patch(_PROVIDER_PATCH) as mock_get_provider:
                    mock_provider = MagicMock()
                    mock_provider.generate = AsyncMock(return_value=mock_provider_result)
                    mock_get_provider.return_value = mock_provider

                    initial_count = len(service._tasks["task-1"]["state"]["generated_images"])

                    edit_request = ImageEditRequest(edit_prompt="Edit", enhance_prompt=False)
                    await service.edit_image(
                        task_id="task-1",
                        image_id="img-1",
                        edit_request=edit_request,
                        user_id="user-123",
                    )

                    new_count = len(service._tasks["task-1"]["state"]["generated_images"])
                    assert new_count == initial_count + 1

                    # Verify the new image has edit metadata
                    new_img = service._tasks["task-1"]["state"]["generated_images"][-1]
                    assert new_img["metadata"]["edited_from"] == "img-1"
                    assert new_img["metadata"]["edit_prompt"] == "Edit"

    @pytest.mark.asyncio
    async def test_edit_image_uses_local_path_as_reference(self):
        """Test that local_path is preferred over URL as reference."""
        service = _build_service_with_task()

        mock_provider_result = ProviderResult(
            url="https://example.com/edited.png",
            metadata={},
        )

        with patch(_WS_PATCH) as mock_ws:
            mock_ws.return_value.send_progress = AsyncMock()

            with patch(_NORMALIZE_PATCH, new_callable=AsyncMock) as mock_normalize:
                mock_normalize.return_value = ["/tmp/img1.png"]

                with patch(_PROVIDER_PATCH) as mock_get_provider:
                    mock_provider = MagicMock()
                    mock_provider.generate = AsyncMock(return_value=mock_provider_result)
                    mock_get_provider.return_value = mock_provider

                    edit_request = ImageEditRequest(edit_prompt="Edit", enhance_prompt=False)
                    await service.edit_image(
                        task_id="task-1",
                        image_id="img-1",
                        edit_request=edit_request,
                        user_id="user-123",
                    )

                    # normalize_reference_images should be called with local_path
                    # and edit-specific compression config.
                    mock_normalize.assert_called_once_with(
                        ["/tmp/img1.png"],
                        max_long_edge=768,
                        keep_http_urls=False,
                    )

    @pytest.mark.asyncio
    async def test_edit_image_falls_back_to_url(self):
        """Test that URL is used when local_path is not available."""
        service = _build_service_with_task()

        mock_provider_result = ProviderResult(
            url="https://example.com/edited.png",
            metadata={},
        )

        with patch(_WS_PATCH) as mock_ws:
            mock_ws.return_value.send_progress = AsyncMock()

            with patch(_NORMALIZE_PATCH, new_callable=AsyncMock) as mock_normalize:
                mock_normalize.return_value = ["https://example.com/img2.png"]

                with patch(_PROVIDER_PATCH) as mock_get_provider:
                    mock_provider = MagicMock()
                    mock_provider.generate = AsyncMock(return_value=mock_provider_result)
                    mock_get_provider.return_value = mock_provider

                    edit_request = ImageEditRequest(edit_prompt="Edit", enhance_prompt=False)
                    # img-2 has no local_path, only URL
                    await service.edit_image(
                        task_id="task-1",
                        image_id="img-2",
                        edit_request=edit_request,
                        user_id="user-123",
                    )

                    mock_normalize.assert_called_once_with(
                        ["https://example.com/img2.png"],
                        max_long_edge=768,
                        keep_http_urls=False,
                    )

    @pytest.mark.asyncio
    async def test_edit_image_normalize_failure_raises(self):
        """Test that normalization failure raises ProviderError instead of silent fallback."""
        service = _build_service_with_task()

        with patch(_WS_PATCH) as mock_ws:
            mock_ws.return_value.send_progress = AsyncMock()

            with patch(
                _NORMALIZE_PATCH,
                new_callable=AsyncMock,
                side_effect=Exception("normalization failed"),
            ):
                edit_request = ImageEditRequest(edit_prompt="Edit", enhance_prompt=False)
                with pytest.raises(ProviderError, match="Failed to prepare reference image"):
                    await service.edit_image(
                        task_id="task-1",
                        image_id="img-1",
                        edit_request=edit_request,
                        user_id="user-123",
                    )

                # Edit lock should be released after failure
                assert "img-1" not in service._editing_images

    @pytest.mark.asyncio
    async def test_edit_image_finds_in_best_images(self):
        """Test that edit can find image in best_images list."""
        service = _build_service_with_task(
            images=[
                {
                    "id": "img-only-in-gen",
                    "url": "https://example.com/gen.png",
                    "local_path": None,
                    "score": 0.5,
                    "review": "",
                    "round_number": 1,
                    "provider": "gemini",
                    "metadata": {},
                }
            ]
        )
        # Add a separate image only in best_images
        service._tasks["task-1"]["state"]["best_images"] = [
            {
                "id": "img-best-only",
                "url": "https://example.com/bestonly.png",
                "local_path": None,
                "score": 0.99,
                "review": "",
                "round_number": 1,
                "provider": "gemini",
                "metadata": {},
            }
        ]

        mock_provider_result = ProviderResult(
            url="https://example.com/edited.png",
            metadata={},
        )

        with patch(_WS_PATCH) as mock_ws:
            mock_ws.return_value.send_progress = AsyncMock()

            with patch(_NORMALIZE_PATCH, new_callable=AsyncMock) as mock_normalize:
                mock_normalize.return_value = ["https://example.com/bestonly.png"]

                with patch(_PROVIDER_PATCH) as mock_get_provider:
                    mock_provider = MagicMock()
                    mock_provider.generate = AsyncMock(return_value=mock_provider_result)
                    mock_get_provider.return_value = mock_provider

                    edit_request = ImageEditRequest(edit_prompt="Edit best", enhance_prompt=False)
                    result = await service.edit_image(
                        task_id="task-1",
                        image_id="img-best-only",
                        edit_request=edit_request,
                        user_id="user-123",
                    )

                    assert result.original_image_id == "img-best-only"

    @pytest.mark.asyncio
    async def test_edit_image_no_user_id_check(self):
        """Test edit without user_id (no ownership check)."""
        service = _build_service_with_task(user_id="user-123")

        mock_provider_result = ProviderResult(
            url="https://example.com/edited.png",
            metadata={},
        )

        with patch(_WS_PATCH) as mock_ws:
            mock_ws.return_value.send_progress = AsyncMock()

            with patch(_NORMALIZE_PATCH, new_callable=AsyncMock) as mock_normalize:
                mock_normalize.return_value = ["/tmp/img1.png"]

                with patch(_PROVIDER_PATCH) as mock_get_provider:
                    mock_provider = MagicMock()
                    mock_provider.generate = AsyncMock(return_value=mock_provider_result)
                    mock_get_provider.return_value = mock_provider

                    edit_request = ImageEditRequest(edit_prompt="Edit", enhance_prompt=False)
                    # user_id=None should bypass ownership check
                    result = await service.edit_image(
                        task_id="task-1",
                        image_id="img-1",
                        edit_request=edit_request,
                        user_id=None,
                    )

                    assert result.new_image.url == "https://example.com/edited.png"

    @pytest.mark.asyncio
    async def test_edit_image_preserves_round_number(self):
        """Test that edited image inherits round_number from original."""
        service = _build_service_with_task(
            images=[
                {
                    "id": "round2-img",
                    "url": "https://example.com/r2.png",
                    "local_path": "/tmp/r2.png",
                    "score": 0.7,
                    "review": "",
                    "round_number": 2,
                    "provider": "gemini",
                    "metadata": {},
                }
            ]
        )

        mock_provider_result = ProviderResult(
            url="https://example.com/edited.png",
            metadata={},
        )

        with patch(_WS_PATCH) as mock_ws:
            mock_ws.return_value.send_progress = AsyncMock()

            with patch(_NORMALIZE_PATCH, new_callable=AsyncMock) as mock_normalize:
                mock_normalize.return_value = ["/tmp/r2.png"]

                with patch(_PROVIDER_PATCH) as mock_get_provider:
                    mock_provider = MagicMock()
                    mock_provider.generate = AsyncMock(return_value=mock_provider_result)
                    mock_get_provider.return_value = mock_provider

                    edit_request = ImageEditRequest(edit_prompt="Edit", enhance_prompt=False)
                    result = await service.edit_image(
                        task_id="task-1",
                        image_id="round2-img",
                        edit_request=edit_request,
                        user_id="user-123",
                    )

                    assert result.new_image.round_number == 2


class TestEnhanceEditPrompt:
    """Tests for the _enhance_edit_prompt helper method."""

    @pytest.mark.asyncio
    async def test_enhance_edit_prompt_success(self):
        """Test successful prompt enhancement."""
        service = GenerationService()

        mock_response = MagicMock()
        mock_response.content = "Enhanced: vibrant blue sky background"

        with patch(_LLM_PATCH) as mock_create_llm:
            mock_llm = MagicMock()
            mock_llm.ainvoke = AsyncMock(return_value=mock_response)
            mock_create_llm.return_value = mock_llm

            result = await service._enhance_edit_prompt("Make background blue")

            assert result == "Enhanced: vibrant blue sky background"
            mock_llm.ainvoke.assert_awaited_once()

    @pytest.mark.asyncio
    async def test_enhance_edit_prompt_strips_quotes(self):
        """Test that enhanced prompt strips surrounding quotes."""
        service = GenerationService()

        mock_response = MagicMock()
        mock_response.content = '"Enhanced prompt with quotes"'

        with patch(_LLM_PATCH) as mock_create_llm:
            mock_llm = MagicMock()
            mock_llm.ainvoke = AsyncMock(return_value=mock_response)
            mock_create_llm.return_value = mock_llm

            result = await service._enhance_edit_prompt("test")

            assert result == "Enhanced prompt with quotes"

    @pytest.mark.asyncio
    async def test_enhance_edit_prompt_handles_list_content(self):
        """Test handling of list content in LLM response."""
        service = GenerationService()

        mock_response = MagicMock()
        mock_response.content = [
            {"text": "Part 1 "},
            {"text": "Part 2"},
        ]

        with patch(_LLM_PATCH) as mock_create_llm:
            mock_llm = MagicMock()
            mock_llm.ainvoke = AsyncMock(return_value=mock_response)
            mock_create_llm.return_value = mock_llm

            result = await service._enhance_edit_prompt("test")

            assert result == "Part 1  Part 2"

    @pytest.mark.asyncio
    async def test_enhance_edit_prompt_fallback_on_error(self):
        """Test that original prompt is returned on LLM error."""
        service = GenerationService()

        with patch(_LLM_PATCH) as mock_create_llm:
            mock_llm = MagicMock()
            mock_llm.ainvoke = AsyncMock(side_effect=Exception("LLM unavailable"))
            mock_create_llm.return_value = mock_llm

            result = await service._enhance_edit_prompt("Original prompt")

            assert result == "Original prompt"
