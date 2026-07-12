"""Tests for data models."""

import pytest
from pictactic.models.schemas import (
    GenerationRequest,
    GeneratedImage,
    TemplateAnalysis,
    ScoreDetails,
)
from pictactic.models.enums import ImageProvider, GenerationStatus


class TestGenerationRequest:
    """Tests for GenerationRequest model."""

    def test_minimal_request(self):
        """Test creating request with minimal fields."""
        request = GenerationRequest(prompt="test prompt", provider=ImageProvider.GEMINI)
        assert request.prompt == "test prompt"
        assert request.max_rounds == 3
        assert request.images_per_round == 5
        assert request.quality_threshold == 0.7
        assert request.aspect_ratio == "16:9"
        assert request.size == "2K"
        assert request.provider == ImageProvider.GEMINI

    def test_full_request(self):
        """Test creating request with all fields."""
        request = GenerationRequest(
            prompt="detailed prompt",
            template_images=["img1.jpg", "img2.jpg"],
            max_rounds=5,
            images_per_round=8,
            quality_threshold=0.85,
            aspect_ratio="1:1",
            size="4K",
            provider=ImageProvider.GEMINI,
        )
        assert request.prompt == "detailed prompt"
        assert len(request.template_images) == 2
        assert request.max_rounds == 5

    def test_validation_prompt_required(self):
        """Test that prompt is required."""
        with pytest.raises(Exception):
            GenerationRequest()

    def test_validation_max_rounds_range(self):
        """Test max_rounds validation."""
        with pytest.raises(Exception):
            GenerationRequest(prompt="test", max_rounds=0)
        with pytest.raises(Exception):
            GenerationRequest(prompt="test", max_rounds=11)


class TestTemplateAnalysis:
    """Tests for TemplateAnalysis model."""

    def test_create_analysis(self):
        """Test creating template analysis."""
        analysis = TemplateAnalysis(
            style="minimalist",
            colors=["blue", "white"],
            layout="centered",
            mood="calm",
            elements=["gradients", "icons"],
            suggested_keywords=["modern", "clean"],
        )
        assert analysis.style == "minimalist"
        assert len(analysis.colors) == 2
        assert analysis.mood == "calm"


class TestScoreDetails:
    """Tests for ScoreDetails model."""

    def test_score_validation(self):
        """Test score values are within range."""
        scores = ScoreDetails(
            prompt_match=0.8,
            style_consistency=0.7,
            visual_appeal=0.9,
            composition=0.75,
            technical_quality=0.85,
        )
        assert scores.prompt_match == 0.8
        assert all(0 <= v <= 1 for v in [
            scores.prompt_match,
            scores.style_consistency,
            scores.visual_appeal,
            scores.composition,
            scores.technical_quality,
        ])


class TestEnums:
    """Tests for enum types."""

    def test_image_provider_values(self):
        """Test ImageProvider enum values."""
        assert ImageProvider.GEMINI.value == "gemini"
        assert ImageProvider.DALLE.value == "dalle"
        assert ImageProvider.STABILITY.value == "stability"

    def test_generation_status_values(self):
        """Test GenerationStatus enum values."""
        assert GenerationStatus.PENDING.value == "pending"
        assert GenerationStatus.COMPLETED.value == "completed"
        assert GenerationStatus.FAILED.value == "failed"
