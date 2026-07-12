"""Tests for in-flight progress updates in GenerationService."""

from datetime import datetime
from unittest.mock import AsyncMock, patch

import pytest
from pictactic.models.enums import GenerationStatus, ImageProvider
from pictactic.models.schemas import GenerationRequest
from pictactic.services.generation_service import GenerationService


def _build_task(request: GenerationRequest) -> dict:
    now = datetime.utcnow()
    return {
        "id": "task-1",
        "status": GenerationStatus.PENDING,
        "request": request,
        "state": {
            "messages": [],
            "user_prompt": request.prompt,
            "enhanced_prompt": None,
            "generated_images": [],
            "best_images": [],
            "current_round": 0,
        },
        "created_at": now,
        "updated_at": now,
        "result": None,
        "error": None,
    }


@pytest.mark.asyncio
async def test_progress_updates_during_generation_and_reviewing():
    service = GenerationService()
    request = GenerationRequest(
        prompt="test prompt",
        max_rounds=3,
        images_per_round=5,
        provider=ImageProvider.GEMINI,
    )

    task_id = "task-1"
    service._tasks[task_id] = _build_task(request)

    service._apply_status_change(task_id, {"status": "generating", "current_round": 1})
    for idx in range(4):
        service._apply_image_ready(task_id, {
            "id": f"img-{idx}",
            "url": f"https://example.com/{idx}.png",
            "round_number": 1,
            "score": 0.0,
            "provider": "gemini",
        })

    progress = await service.get_task_status(task_id, lang="zh")
    assert progress.status == GenerationStatus.GENERATING
    assert progress.current_round == 1
    assert progress.images_generated == 4
    assert progress.images_in_current_round == 4
    assert "第 1 轮" in progress.message

    service._apply_status_change(task_id, {"status": "reviewing", "current_round": 1})
    progress = await service.get_task_status(task_id, lang="zh")
    assert progress.status == GenerationStatus.REVIEWING
    assert "正在评估图片" in progress.message


@pytest.mark.asyncio
async def test_current_best_score_reads_generated_images():
    service = GenerationService()
    request = GenerationRequest(
        prompt="test prompt",
        max_rounds=2,
        images_per_round=2,
        provider=ImageProvider.GEMINI,
    )

    task_id = "task-1"
    service._tasks[task_id] = _build_task(request)
    service._apply_status_change(task_id, {"status": "reviewing", "current_round": 1})
    service._apply_image_ready(task_id, {
        "id": "img-1",
        "url": "https://example.com/1.png",
        "round_number": 1,
        "score": 0.83,
        "provider": "gemini",
    })

    progress = await service.get_task_status(task_id)
    assert progress.current_best_score == 0.83


@pytest.mark.asyncio
async def test_status_persistence_is_coalesced():
    with (
        patch("pictactic.services.generation_service.create_generation_workflow"),
        patch("pictactic.services.generation_service.TaskRepository") as mock_repo_cls,
    ):
        mock_repo = mock_repo_cls.return_value
        mock_repo.update_status = AsyncMock()
        service = GenerationService()

    request = GenerationRequest(
        prompt="test prompt",
        max_rounds=2,
        images_per_round=2,
        provider=ImageProvider.GEMINI,
    )
    task_id = "task-1"
    service._tasks[task_id] = _build_task(request)

    service._update_task_status(task_id, GenerationStatus.ENHANCING)
    service._update_task_status(task_id, GenerationStatus.GENERATING)

    await service._wait_for_status_persist(task_id)

    mock_repo.update_status.assert_awaited_once_with(
        task_id,
        GenerationStatus.GENERATING.value,
        None,
    )
