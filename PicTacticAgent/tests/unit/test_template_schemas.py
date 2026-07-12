"""Tests for prompt template Pydantic schemas."""

import pytest
from pydantic import ValidationError

from pictactic.models.schemas import (
    TemplateConfigOverride,
    TemplateCreate,
    TemplateGenerateRequest,
    TemplateListResponse,
    TemplateResponse,
    TemplateUpdate,
)


class TestTemplateCreate:
    """Tests for TemplateCreate schema."""

    def test_valid_minimal(self):
        data = TemplateCreate(title="My Template", prompt_text="A beautiful landscape")
        assert data.title == "My Template"
        assert data.prompt_text == "A beautiful landscape"
        assert data.category == "other"
        assert data.tags == []
        assert data.description is None
        assert data.preview_image_url is None
        assert data.config is None

    def test_valid_full(self):
        config = TemplateConfigOverride(max_rounds=5, quality_threshold=0.9)
        data = TemplateCreate(
            title="Full Template",
            description="A detailed description",
            prompt_text="Generate a product photo",
            category="product",
            tags=["product", "photo"],
            preview_image_url="https://example.com/img.png",
            config=config,
        )
        assert data.title == "Full Template"
        assert data.description == "A detailed description"
        assert data.category == "product"
        assert len(data.tags) == 2
        assert data.config.max_rounds == 5

    def test_empty_title_rejected(self):
        with pytest.raises(ValidationError) as exc_info:
            TemplateCreate(title="", prompt_text="some prompt")
        assert "title" in str(exc_info.value).lower()

    def test_missing_title_rejected(self):
        with pytest.raises(ValidationError):
            TemplateCreate(prompt_text="some prompt")

    def test_empty_prompt_text_rejected(self):
        with pytest.raises(ValidationError) as exc_info:
            TemplateCreate(title="Valid Title", prompt_text="")
        assert "prompt_text" in str(exc_info.value).lower()

    def test_missing_prompt_text_rejected(self):
        with pytest.raises(ValidationError):
            TemplateCreate(title="Valid Title")

    def test_title_max_length(self):
        long_title = "A" * 201
        with pytest.raises(ValidationError):
            TemplateCreate(title=long_title, prompt_text="prompt")

    def test_title_at_max_length(self):
        title = "A" * 200
        data = TemplateCreate(title=title, prompt_text="prompt")
        assert len(data.title) == 200


class TestTemplateUpdate:
    """Tests for TemplateUpdate schema."""

    def test_all_none_is_valid(self):
        data = TemplateUpdate()
        assert data.title is None
        assert data.prompt_text is None
        assert data.category is None
        assert data.tags is None

    def test_partial_update_title(self):
        data = TemplateUpdate(title="New Title")
        assert data.title == "New Title"
        assert data.prompt_text is None

    def test_partial_update_category(self):
        data = TemplateUpdate(category="portrait")
        assert data.category == "portrait"

    def test_partial_update_config(self):
        data = TemplateUpdate(config=TemplateConfigOverride(images_per_round=3))
        assert data.config.images_per_round == 3
        assert data.config.max_rounds is None

    def test_empty_title_rejected(self):
        with pytest.raises(ValidationError):
            TemplateUpdate(title="")

    def test_empty_prompt_text_rejected(self):
        with pytest.raises(ValidationError):
            TemplateUpdate(prompt_text="")


class TestTemplateResponse:
    """Tests for TemplateResponse schema."""

    def test_serialization(self):
        resp = TemplateResponse(
            id="tpl-001",
            user_id="user-001",
            title="Cover Template",
            prompt_text="A modern book cover",
            category="cover",
            tags=["book", "modern"],
            usage_count=5,
            is_public=False,
            created_at="2025-01-01T00:00:00",
            updated_at="2025-01-02T00:00:00",
        )
        assert resp.id == "tpl-001"
        assert resp.user_id == "user-001"
        assert resp.title == "Cover Template"
        assert resp.category == "cover"
        assert resp.tags == ["book", "modern"]
        assert resp.usage_count == 5
        assert resp.is_public is False
        assert resp.description is None
        assert resp.config is None
        assert resp.preview_image_url is None

    def test_serialization_with_optional_fields(self):
        resp = TemplateResponse(
            id="tpl-002",
            user_id="user-001",
            title="Template",
            description="Desc",
            prompt_text="prompt",
            category="other",
            config={"max_rounds": 3},
            preview_image_url="https://img.url/pic.png",
            created_at="2025-01-01T00:00:00",
            updated_at="2025-01-01T00:00:00",
        )
        assert resp.description == "Desc"
        assert resp.config == {"max_rounds": 3}
        assert resp.preview_image_url == "https://img.url/pic.png"

    def test_to_dict(self):
        resp = TemplateResponse(
            id="tpl-003",
            user_id="user-001",
            title="T",
            prompt_text="p",
            category="other",
            created_at="2025-01-01T00:00:00",
            updated_at="2025-01-01T00:00:00",
        )
        d = resp.model_dump()
        assert isinstance(d, dict)
        assert d["id"] == "tpl-003"
        assert d["tags"] == []
        assert d["usage_count"] == 0


class TestTemplateConfigOverride:
    """Tests for TemplateConfigOverride schema."""

    def test_all_defaults_none(self):
        cfg = TemplateConfigOverride()
        assert cfg.max_rounds is None
        assert cfg.images_per_round is None
        assert cfg.quality_threshold is None
        assert cfg.aspect_ratio is None
        assert cfg.size is None
        assert cfg.enhance_prompt is None
        assert cfg.enable_ai_scoring is None
        assert cfg.provider is None

    def test_valid_values(self):
        cfg = TemplateConfigOverride(
            max_rounds=5,
            images_per_round=8,
            quality_threshold=0.85,
            aspect_ratio="1:1",
            size="4K",
            enhance_prompt=False,
            enable_ai_scoring=True,
            provider="gemini",
        )
        assert cfg.max_rounds == 5
        assert cfg.images_per_round == 8
        assert cfg.quality_threshold == 0.85
        assert cfg.provider == "gemini"

    def test_max_rounds_too_low(self):
        with pytest.raises(ValidationError):
            TemplateConfigOverride(max_rounds=0)

    def test_max_rounds_too_high(self):
        with pytest.raises(ValidationError):
            TemplateConfigOverride(max_rounds=11)

    def test_images_per_round_too_low(self):
        with pytest.raises(ValidationError):
            TemplateConfigOverride(images_per_round=0)

    def test_images_per_round_too_high(self):
        with pytest.raises(ValidationError):
            TemplateConfigOverride(images_per_round=11)

    def test_quality_threshold_too_low(self):
        with pytest.raises(ValidationError):
            TemplateConfigOverride(quality_threshold=-0.1)

    def test_quality_threshold_too_high(self):
        with pytest.raises(ValidationError):
            TemplateConfigOverride(quality_threshold=1.1)

    def test_exclude_none_dump(self):
        cfg = TemplateConfigOverride(max_rounds=3)
        d = cfg.model_dump(exclude_none=True)
        assert d == {"max_rounds": 3}


class TestTemplateListResponse:
    """Tests for TemplateListResponse schema."""

    def test_structure(self):
        item = TemplateResponse(
            id="t1",
            user_id="u1",
            title="T1",
            prompt_text="p1",
            category="other",
            created_at="2025-01-01T00:00:00",
            updated_at="2025-01-01T00:00:00",
        )
        resp = TemplateListResponse(items=[item], total=1, page=1, page_size=20)
        assert len(resp.items) == 1
        assert resp.total == 1
        assert resp.page == 1
        assert resp.page_size == 20

    def test_empty_list(self):
        resp = TemplateListResponse(items=[], total=0, page=1, page_size=20)
        assert resp.items == []
        assert resp.total == 0


class TestTemplateGenerateRequest:
    """Tests for TemplateGenerateRequest schema."""

    def test_defaults(self):
        req = TemplateGenerateRequest()
        assert req.config_overrides is None
        assert req.template_images == []

    def test_with_overrides(self):
        req = TemplateGenerateRequest(
            config_overrides=TemplateConfigOverride(max_rounds=2),
            template_images=["img1.jpg"],
        )
        assert req.config_overrides.max_rounds == 2
        assert req.template_images == ["img1.jpg"]

    def test_with_empty_overrides(self):
        req = TemplateGenerateRequest(config_overrides=TemplateConfigOverride())
        assert req.config_overrides is not None
        assert req.config_overrides.max_rounds is None
