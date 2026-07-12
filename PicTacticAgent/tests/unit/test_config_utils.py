"""Tests for config utility functions: update_env_file, apply_settings_in_memory."""

from unittest.mock import MagicMock, patch

import pytest


class TestUpdateEnvFile:
    """Tests for update_env_file."""

    def test_update_existing_key(self, tmp_path):
        """Existing key in .env gets replaced."""
        env_file = tmp_path / ".env"
        env_file.write_text("FOO=bar\nBAZ=qux\n")

        mock_settings = MagicMock()
        mock_settings.model_config.get.return_value = str(env_file)

        with patch("pictactic.core.config.settings", mock_settings):
            from pictactic.core.config import update_env_file

            update_env_file({"FOO": "new_bar"})

        content = env_file.read_text()
        assert "FOO=new_bar" in content
        assert "BAZ=qux" in content

    def test_append_new_key(self, tmp_path):
        """New key not in .env gets appended."""
        env_file = tmp_path / ".env"
        env_file.write_text("FOO=bar\n")

        mock_settings = MagicMock()
        mock_settings.model_config.get.return_value = str(env_file)

        with patch("pictactic.core.config.settings", mock_settings):
            from pictactic.core.config import update_env_file

            update_env_file({"NEW_KEY": "new_value"})

        content = env_file.read_text()
        assert "FOO=bar" in content
        assert "NEW_KEY=new_value" in content

    def test_update_and_append_mixed(self, tmp_path):
        """Some existing keys get replaced, some new keys get appended."""
        env_file = tmp_path / ".env"
        env_file.write_text("EXISTING=old\nKEEP=same\n")

        mock_settings = MagicMock()
        mock_settings.model_config.get.return_value = str(env_file)

        with patch("pictactic.core.config.settings", mock_settings):
            from pictactic.core.config import update_env_file

            update_env_file({"EXISTING": "updated", "BRAND_NEW": "fresh"})

        content = env_file.read_text()
        assert "EXISTING=updated" in content
        assert "KEEP=same" in content
        assert "BRAND_NEW=fresh" in content
        # Original value should be gone
        assert "EXISTING=old" not in content

    def test_preserves_comments(self, tmp_path):
        """Lines starting with # are preserved."""
        env_file = tmp_path / ".env"
        env_file.write_text("# This is a comment\nFOO=bar\n# Another comment\n")

        mock_settings = MagicMock()
        mock_settings.model_config.get.return_value = str(env_file)

        with patch("pictactic.core.config.settings", mock_settings):
            from pictactic.core.config import update_env_file

            update_env_file({"FOO": "baz"})

        content = env_file.read_text()
        assert "# This is a comment" in content
        assert "# Another comment" in content
        assert "FOO=baz" in content

    def test_preserves_empty_lines(self, tmp_path):
        """Blank lines are preserved."""
        env_file = tmp_path / ".env"
        env_file.write_text("FOO=bar\n\nBAZ=qux\n")

        mock_settings = MagicMock()
        mock_settings.model_config.get.return_value = str(env_file)

        with patch("pictactic.core.config.settings", mock_settings):
            from pictactic.core.config import update_env_file

            update_env_file({"FOO": "updated"})

        lines = env_file.read_text().splitlines(keepends=True)
        # The blank line should still be there
        blank_lines = [line for line in lines if line.strip() == ""]
        assert len(blank_lines) >= 1
        assert "FOO=updated" in env_file.read_text()
        assert "BAZ=qux" in env_file.read_text()

    def test_creates_file_if_not_exists(self, tmp_path):
        """.env file doesn't exist, gets created."""
        env_file = tmp_path / ".env"
        assert not env_file.exists()

        mock_settings = MagicMock()
        mock_settings.model_config.get.return_value = str(env_file)

        with patch("pictactic.core.config.settings", mock_settings):
            from pictactic.core.config import update_env_file

            update_env_file({"NEW_KEY": "value1", "OTHER_KEY": "value2"})

        assert env_file.exists()
        content = env_file.read_text()
        assert "NEW_KEY=value1" in content
        assert "OTHER_KEY=value2" in content

    def test_handles_value_with_equals(self, tmp_path):
        """Values containing = sign (e.g. API keys with =) are handled correctly."""
        env_file = tmp_path / ".env"
        env_file.write_text("API_KEY=abc123==\n")

        mock_settings = MagicMock()
        mock_settings.model_config.get.return_value = str(env_file)

        with patch("pictactic.core.config.settings", mock_settings):
            from pictactic.core.config import update_env_file

            update_env_file({"API_KEY": "xyz789==end"})

        content = env_file.read_text()
        assert "API_KEY=xyz789==end" in content
        # Old value should be gone
        assert "abc123==" not in content


class TestApplySettingsInMemory:
    """Tests for apply_settings_in_memory."""

    def test_applies_known_field(self):
        """DEFAULT_PROVIDER -> default_provider gets set."""
        mock_settings = MagicMock()
        mock_settings.__dict__ = {"default_provider": "gemini"}

        with patch("pictactic.core.config.settings", mock_settings):
            from pictactic.core.config import apply_settings_in_memory

            apply_settings_in_memory({"DEFAULT_PROVIDER": "jiekouai"})

        assert mock_settings.__dict__["default_provider"] == "jiekouai"

    def test_ignores_unknown_field(self):
        """UNKNOWN_FIELD is ignored (not in Settings.model_fields)."""
        mock_settings = MagicMock()
        original_dict = dict(mock_settings.__dict__)

        with patch("pictactic.core.config.settings", mock_settings):
            from pictactic.core.config import apply_settings_in_memory

            # This should not raise and should not add the key
            apply_settings_in_memory({"TOTALLY_UNKNOWN_FIELD_XYZ": "value"})

        # The unknown field should not appear in settings
        assert "totally_unknown_field_xyz" not in mock_settings.__dict__

    def test_multiple_updates(self):
        """Multiple keys all get applied."""
        mock_settings = MagicMock()
        mock_settings.__dict__ = {
            "default_provider": "gemini",
            "gemini_api_key": "old_key",
        }

        with patch("pictactic.core.config.settings", mock_settings):
            from pictactic.core.config import apply_settings_in_memory

            apply_settings_in_memory({
                "DEFAULT_PROVIDER": "antigravity",
                "GEMINI_API_KEY": "new_key_123",
            })

        assert mock_settings.__dict__["default_provider"] == "antigravity"
        assert mock_settings.__dict__["gemini_api_key"] == "new_key_123"

    def test_case_conversion(self):
        """GEMINI_API_KEY -> gemini_api_key (uppercase ENV to lowercase field)."""
        mock_settings = MagicMock()
        mock_settings.__dict__ = {"gemini_api_key": ""}

        with patch("pictactic.core.config.settings", mock_settings):
            from pictactic.core.config import apply_settings_in_memory

            apply_settings_in_memory({"GEMINI_API_KEY": "test_key_abc"})

        assert mock_settings.__dict__["gemini_api_key"] == "test_key_abc"




def _make_settings():
    """Create a fresh Settings instance without reading .env file."""
    from pictactic.core.config import Settings

    return Settings(_env_file=None)
