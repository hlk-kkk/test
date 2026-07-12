"""Tests for CLI providers command."""

import json
from unittest.mock import patch, MagicMock

from typer.testing import CliRunner

from pictactic.cli.providers import providers_app


runner = CliRunner()


def _mock_capabilities():
    """Create mock provider capabilities."""
    return [
        {
            "name": "gemini",
            "supported_modes": ["text_to_image", "image_to_image"],
            "supports_batch": True,
        },
        {
            "name": "jiekouai",
            "supported_modes": ["text_to_image", "image_to_image"],
            "supports_batch": True,
        },
        {
            "name": "antigravity",
            "supported_modes": ["text_to_image"],
            "supports_batch": False,
        },
    ]


class TestListProviders:
    """Tests for providers list command."""

    def test_json_output(self):
        """JSON format outputs parseable JSON array."""
        caps = _mock_capabilities()
        mock_settings = MagicMock()
        mock_settings.default_provider = "gemini"

        with patch(
            "pictactic.cli.providers._PROVIDERS",
            {"gemini": MagicMock, "jiekouai": MagicMock, "antigravity": MagicMock},
        ), patch(
            "pictactic.cli.providers.get_provider_capabilities",
            side_effect=lambda n: next(c for c in caps if c["name"] == n),
        ), patch(
            "pictactic.cli.providers.settings", mock_settings,
        ):
            result = runner.invoke(providers_app, ["--format", "json"])

        assert result.exit_code == 0
        data = json.loads(result.output)
        assert len(data) == 3
        names = [c["name"] for c in data]
        assert "gemini" in names
        assert "jiekouai" in names
        assert "antigravity" in names

    def test_json_marks_default(self):
        """Default provider is flagged with is_default."""
        caps = _mock_capabilities()
        mock_settings = MagicMock()
        mock_settings.default_provider = "jiekouai"

        with patch(
            "pictactic.cli.providers._PROVIDERS",
            {"gemini": MagicMock, "jiekouai": MagicMock, "antigravity": MagicMock},
        ), patch(
            "pictactic.cli.providers.get_provider_capabilities",
            side_effect=lambda n: next(c for c in caps if c["name"] == n),
        ), patch(
            "pictactic.cli.providers.settings", mock_settings,
        ):
            result = runner.invoke(providers_app, ["--format", "json"])

        data = json.loads(result.output)
        default_entries = [c for c in data if c.get("is_default")]
        assert len(default_entries) == 1
        assert default_entries[0]["name"] == "jiekouai"

    def test_text_output(self):
        """Text format renders a rich table."""
        caps = _mock_capabilities()
        mock_settings = MagicMock()
        mock_settings.default_provider = "gemini"

        with patch(
            "pictactic.cli.providers._PROVIDERS",
            {"gemini": MagicMock, "jiekouai": MagicMock, "antigravity": MagicMock},
        ), patch(
            "pictactic.cli.providers.get_provider_capabilities",
            side_effect=lambda n: next(c for c in caps if c["name"] == n),
        ), patch(
            "pictactic.cli.providers.settings", mock_settings,
        ):
            result = runner.invoke(providers_app, ["--format", "text"])

        assert result.exit_code == 0
        # Table should contain provider names
        assert "gemini" in result.output
        assert "jiekouai" in result.output

    def test_default_format_is_text(self):
        """Default format should be text (no --format specified)."""
        caps = _mock_capabilities()
        mock_settings = MagicMock()
        mock_settings.default_provider = "gemini"

        with patch(
            "pictactic.cli.providers._PROVIDERS",
            {"gemini": MagicMock},
        ), patch(
            "pictactic.cli.providers.get_provider_capabilities",
            return_value=caps[0],
        ), patch(
            "pictactic.cli.providers.settings", mock_settings,
        ):
            result = runner.invoke(providers_app, [])

        assert result.exit_code == 0
        assert "gemini" in result.output

    def test_single_provider(self):
        """Works with a single registered provider."""
        cap = {
            "name": "gemini",
            "supported_modes": ["text_to_image"],
            "supports_batch": True,
        }
        mock_settings = MagicMock()
        mock_settings.default_provider = "gemini"

        with patch(
            "pictactic.cli.providers._PROVIDERS",
            {"gemini": MagicMock},
        ), patch(
            "pictactic.cli.providers.get_provider_capabilities",
            return_value=cap,
        ), patch(
            "pictactic.cli.providers.settings", mock_settings,
        ):
            result = runner.invoke(providers_app, ["--format", "json"])

        data = json.loads(result.output)
        assert len(data) == 1
        assert data[0]["is_default"] is True
