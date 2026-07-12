"""Tests for TemplateService."""

import pytest
from datetime import datetime
from unittest.mock import AsyncMock, MagicMock, patch

from pictactic.core.exceptions import TemplateNotFoundError
from pictactic.models.schemas import (
    TemplateConfigOverride,
    TemplateCreate,
    TemplateGenerateRequest,
    TemplateUpdate,
)
from pictactic.services.template_service import TemplateService, _to_response


def _make_orm_template(**overrides):
    """Create a mock PromptTemplate ORM object."""
    defaults = dict(
        id="tpl-001",
        user_id="user-001",
        title="Test Template",
        description="A test template",
        prompt_text="Generate a beautiful landscape",
        category="landscape",
        tags=["nature", "scenic"],
        preview_image_url=None,
        config=None,
        usage_count=0,
        is_public=False,
        is_system=False,
        source_url=None,
        source_author=None,
        created_at=datetime(2025, 1, 1, 0, 0, 0),
        updated_at=datetime(2025, 1, 2, 0, 0, 0),
    )
    defaults.update(overrides)
    obj = MagicMock()
    for k, v in defaults.items():
        setattr(obj, k, v)
    return obj


class TestToResponse:
    """Tests for the _to_response helper."""

    def test_converts_orm_to_schema(self):
        orm = _make_orm_template()
        resp = _to_response(orm)
        assert resp.id == "tpl-001"
        assert resp.user_id == "user-001"
        assert resp.title == "Test Template"
        assert resp.tags == ["nature", "scenic"]
        assert resp.created_at == "2025-01-01T00:00:00"
        assert resp.updated_at == "2025-01-02T00:00:00"

    def test_none_tags_becomes_empty_list(self):
        orm = _make_orm_template(tags=None)
        resp = _to_response(orm)
        assert resp.tags == []


class TestCreateTemplate:
    async def test_create_template_success(self):
        service = TemplateService()
        orm = _make_orm_template()
        service.repo = MagicMock()
        service.repo.create = AsyncMock(return_value=orm)

        data = TemplateCreate(title="Test Template", prompt_text="Generate a beautiful landscape")
        result = await service.create_template("user-001", data)

        service.repo.create.assert_awaited_once_with("user-001", data)
        assert result.id == "tpl-001"
        assert result.title == "Test Template"


class TestGetTemplate:
    async def test_get_template_success(self):
        service = TemplateService()
        orm = _make_orm_template()
        service.repo = MagicMock()
        service.repo.get_by_id = AsyncMock(return_value=orm)

        result = await service.get_template("tpl-001", "user-001")
        service.repo.get_by_id.assert_awaited_once_with("tpl-001", "user-001")
        assert result.id == "tpl-001"

    async def test_get_template_not_found(self):
        service = TemplateService()
        service.repo = MagicMock()
        service.repo.get_by_id = AsyncMock(return_value=None)

        with pytest.raises(TemplateNotFoundError):
            await service.get_template("nonexistent", "user-001")


class TestListTemplates:
    async def test_list_templates_pagination(self):
        service = TemplateService()
        orm1 = _make_orm_template(id="tpl-001")
        orm2 = _make_orm_template(id="tpl-002")
        service.repo = MagicMock()
        service.repo.list_templates = AsyncMock(return_value=([orm1, orm2], 2))

        result = await service.list_templates(
            user_id="user-001", search=None, category=None, page=1, page_size=20
        )
        assert result.total == 2
        assert len(result.items) == 2
        assert result.page == 1
        assert result.page_size == 20

    async def test_list_templates_with_search(self):
        service = TemplateService()
        service.repo = MagicMock()
        service.repo.list_templates = AsyncMock(return_value=([], 0))

        result = await service.list_templates(
            user_id="user-001", search="landscape", category="landscape", page=1, page_size=10
        )
        service.repo.list_templates.assert_awaited_once_with(
            user_id="user-001", search="landscape", category="landscape", page=1, page_size=10
        )
        assert result.total == 0
        assert result.items == []


class TestUpdateTemplate:
    async def test_update_template_success(self):
        service = TemplateService()
        updated_orm = _make_orm_template(title="Updated Title")
        service.repo = MagicMock()
        service.repo.update = AsyncMock(return_value=updated_orm)

        data = TemplateUpdate(title="Updated Title")
        result = await service.update_template("tpl-001", "user-001", data)
        service.repo.update.assert_awaited_once_with("tpl-001", "user-001", data)
        assert result.title == "Updated Title"

    async def test_update_template_not_found(self):
        service = TemplateService()
        service.repo = MagicMock()
        service.repo.update = AsyncMock(return_value=None)

        data = TemplateUpdate(title="X")
        with pytest.raises(TemplateNotFoundError):
            await service.update_template("nonexistent", "user-001", data)


class TestDeleteTemplate:
    async def test_delete_template_success(self):
        service = TemplateService()
        service.repo = MagicMock()
        service.repo.delete = AsyncMock(return_value=True)

        await service.delete_template("tpl-001", "user-001")
        service.repo.delete.assert_awaited_once_with("tpl-001", "user-001")

    async def test_delete_template_not_found(self):
        service = TemplateService()
        service.repo = MagicMock()
        service.repo.delete = AsyncMock(return_value=False)

        with pytest.raises(TemplateNotFoundError):
            await service.delete_template("nonexistent", "user-001")


class TestGenerateFromTemplate:
    async def test_generate_from_template_success(self):
        service = TemplateService()
        orm = _make_orm_template(
            prompt_text="A landscape prompt",
            config={"max_rounds": 5},
        )
        service.repo = MagicMock()
        service.repo.get_by_id = AsyncMock(return_value=orm)
        service.repo.increment_usage = AsyncMock()

        mock_gen_service = MagicMock()
        mock_gen_service.create_task = AsyncMock(return_value="task-001")
        mock_gen_service.start_task = AsyncMock()

        background_tasks = MagicMock()

        request = TemplateGenerateRequest()

        with patch(
            "pictactic.services.generation_service.get_generation_service",
            return_value=mock_gen_service,
        ):
            with patch("pictactic.services.template_service.settings") as mock_settings:
                mock_settings.default_max_rounds = 3
                mock_settings.default_images_per_round = 5
                mock_settings.default_quality_threshold = 0.7
                mock_settings.default_aspect_ratio = "16:9"
                mock_settings.default_image_size = "2K"
                mock_settings.default_provider = "gemini"

                result = await service.generate_from_template(
                    "tpl-001", "user-001", request, background_tasks
                )

        assert result.task_id == "task-001"
        assert result.status == "pending"
        assert result.message == "Task created"
        service.repo.get_by_id.assert_awaited_once_with("tpl-001", "user-001")
        service.repo.increment_usage.assert_awaited_once_with("tpl-001")
        mock_gen_service.create_task.assert_awaited_once()
        background_tasks.add_task.assert_called_once()

    async def test_generate_from_template_not_found(self):
        service = TemplateService()
        service.repo = MagicMock()
        service.repo.get_by_id = AsyncMock(return_value=None)

        background_tasks = MagicMock()
        request = TemplateGenerateRequest()

        with pytest.raises(TemplateNotFoundError):
            await service.generate_from_template(
                "nonexistent", "user-001", request, background_tasks
            )

    async def test_generate_with_config_overrides(self):
        service = TemplateService()
        orm = _make_orm_template(prompt_text="prompt", config={"max_rounds": 5})
        service.repo = MagicMock()
        service.repo.get_by_id = AsyncMock(return_value=orm)
        service.repo.increment_usage = AsyncMock()

        mock_gen_service = MagicMock()
        mock_gen_service.create_task = AsyncMock(return_value="task-002")
        mock_gen_service.start_task = AsyncMock()

        background_tasks = MagicMock()
        request = TemplateGenerateRequest(
            config_overrides=TemplateConfigOverride(max_rounds=2, quality_threshold=0.5),
        )

        with patch(
            "pictactic.services.generation_service.get_generation_service",
            return_value=mock_gen_service,
        ):
            with patch("pictactic.services.template_service.settings") as mock_settings:
                mock_settings.default_max_rounds = 3
                mock_settings.default_images_per_round = 5
                mock_settings.default_quality_threshold = 0.7
                mock_settings.default_aspect_ratio = "16:9"
                mock_settings.default_image_size = "2K"
                mock_settings.default_provider = "gemini"

                result = await service.generate_from_template(
                    "tpl-001", "user-001", request, background_tasks
                )

        assert result.task_id == "task-002"
        # Verify that create_task was called with merged config
        call_args = mock_gen_service.create_task.call_args
        gen_request = call_args[0][0]
        assert gen_request.max_rounds == 2
        assert gen_request.quality_threshold == 0.5
