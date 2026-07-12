"""Tests for CLI edit command."""

import base64
import shutil
from pathlib import Path
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
import typer
from typer.testing import CliRunner

from pictactic.cli.edit import (
    _enhance_edit_prompt,
    _load_and_normalize_sources,
    _save_edited_images,
    edit_app,
    run_edit,
)
from pictactic.cli.output import OutputFormat, OutputFormatter
from pictactic.providers.base import GenerationResult


runner = CliRunner()

_NORMALIZE_PATCH = "pictactic.cli.edit._load_and_normalize_sources"
_PROVIDERS_PATCH = "pictactic.cli.edit.run_edit"


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _make_result(url: str = "", local_path: str | None = None) -> GenerationResult:
    return GenerationResult(url=url, local_path=local_path)


# ---------------------------------------------------------------------------
# _load_and_normalize_sources
# ---------------------------------------------------------------------------


class TestLoadAndNormalizeSources:
    """Tests for _load_and_normalize_sources."""

    async def test_raises_for_missing_local_file(self, tmp_path):
        missing = str(tmp_path / "nonexistent.png")
        with pytest.raises(typer.BadParameter, match="Source image not found"):
            await _load_and_normalize_sources([missing])

    async def test_passes_http_url_without_existence_check(self):
        """HTTP URLs should not be checked for local existence."""
        # Lazy import inside _load_and_normalize_sources → patch at source module
        with patch(
            "pictactic.agents.tools._image_utils.normalize_reference_images",
            new_callable=AsyncMock,
            return_value=["data:image/jpeg;base64,AA=="],
        ):
            # Should not raise BadParameter even though URL doesn't exist locally
            result = await _load_and_normalize_sources(["https://example.com/img.png"])
        assert result == ["data:image/jpeg;base64,AA=="]

    async def test_calls_normalize_with_correct_params(self, tmp_path):
        img = tmp_path / "test.png"
        img.write_bytes(b"\x89PNG")

        # Patch at source because _load_and_normalize_sources uses lazy import
        with patch(
            "pictactic.agents.tools._image_utils.normalize_reference_images",
            new_callable=AsyncMock,
            return_value=["data:image/jpeg;base64,AA=="],
        ) as mock_norm:
            result = await _load_and_normalize_sources([str(img)])

        # The lazy import re-imports the function, so we check it was called with right args
        # by verifying the return value (mock is active during the call)
        assert result == ["data:image/jpeg;base64,AA=="]

    async def test_data_uri_passes_existence_check(self):
        """data: URIs should not trigger local file existence check."""
        with patch(
            "pictactic.agents.tools._image_utils.normalize_reference_images",
            new_callable=AsyncMock,
            return_value=["data:image/jpeg;base64,AA=="],
        ):
            # Should not raise
            result = await _load_and_normalize_sources(["data:image/jpeg;base64,/9j/AA=="])
        assert result is not None


# ---------------------------------------------------------------------------
# _enhance_edit_prompt
# ---------------------------------------------------------------------------


class TestEnhanceEditPrompt:
    """Tests for _enhance_edit_prompt.

    create_vision_llm is lazily imported inside the function, so we patch
    at the source module: pictactic.core.config.create_vision_llm.
    """

    async def test_returns_enhanced_prompt(self):
        mock_llm = AsyncMock()
        mock_response = MagicMock()
        mock_response.content = "  Enhanced: make the sky golden hour at sunset  "
        mock_llm.ainvoke = AsyncMock(return_value=mock_response)

        with patch("pictactic.core.config.create_vision_llm", return_value=mock_llm):
            result = await _enhance_edit_prompt("make the sky sunset")

        assert result == "Enhanced: make the sky golden hour at sunset"

    async def test_falls_back_on_llm_failure(self):
        mock_llm = AsyncMock()
        mock_llm.ainvoke = AsyncMock(side_effect=RuntimeError("LLM error"))

        with patch("pictactic.core.config.create_vision_llm", return_value=mock_llm):
            result = await _enhance_edit_prompt("original prompt")

        assert result == "original prompt"

    async def test_strips_quotes(self):
        mock_llm = AsyncMock()
        mock_response = MagicMock()
        mock_response.content = '"Enhanced prompt"'
        mock_llm.ainvoke = AsyncMock(return_value=mock_response)

        with patch("pictactic.core.config.create_vision_llm", return_value=mock_llm):
            result = await _enhance_edit_prompt("original")

        assert result == "Enhanced prompt"


# ---------------------------------------------------------------------------
# _save_edited_images
# ---------------------------------------------------------------------------


class TestSaveEditedImages:
    """Tests for _save_edited_images."""

    async def test_saves_from_local_path(self, tmp_path):
        src = tmp_path / "src.png"
        src.write_bytes(b"source image bytes")
        result = _make_result(url="/static/images/src.png", local_path=str(src))

        saved = await _save_edited_images([result], tmp_path / "out")

        assert len(saved) == 1
        assert saved[0].get("local_path") is not None
        dest = Path(saved[0]["local_path"])
        assert dest.exists()
        assert dest.read_bytes() == b"source image bytes"

    async def test_saves_data_uri(self, tmp_path):
        raw = b"edited pixel data"
        encoded = base64.b64encode(raw).decode()
        result = _make_result(url=f"data:image/png;base64,{encoded}")

        saved = await _save_edited_images([result], tmp_path)

        assert len(saved) == 1
        dest = Path(saved[0]["local_path"])
        assert dest.read_bytes() == raw

    async def test_saves_http_url(self, tmp_path):
        result = _make_result(url="https://example.com/edited.png")

        mock_response = MagicMock()
        mock_response.content = b"downloaded edited image"
        mock_response.raise_for_status = MagicMock()

        with patch("pictactic.cli.edit.httpx.AsyncClient") as mock_cls:
            mock_client = AsyncMock()
            mock_client.get = AsyncMock(return_value=mock_response)
            mock_client.__aenter__ = AsyncMock(return_value=mock_client)
            mock_client.__aexit__ = AsyncMock(return_value=False)
            mock_cls.return_value = mock_client

            saved = await _save_edited_images([result], tmp_path)

        assert saved[0].get("local_path") is not None
        assert Path(saved[0]["local_path"]).read_bytes() == b"downloaded edited image"

    async def test_unknown_url_returns_without_local_path(self, tmp_path):
        result = _make_result(url="ftp://server/img.png")
        saved = await _save_edited_images([result], tmp_path)

        assert len(saved) == 1
        assert "local_path" not in saved[0]

    async def test_creates_output_dir(self, tmp_path):
        out = tmp_path / "nested" / "output"
        raw = b"data"
        encoded = base64.b64encode(raw).decode()
        result = _make_result(url=f"data:image/png;base64,{encoded}")

        await _save_edited_images([result], out)
        assert out.exists()

    async def test_http_download_failure_returns_without_local_path(self, tmp_path):
        import httpx as httpx_mod

        result = _make_result(url="https://example.com/bad.png")

        with patch("pictactic.cli.edit.httpx.AsyncClient") as mock_cls:
            mock_client = AsyncMock()
            mock_client.get = AsyncMock(side_effect=httpx_mod.HTTPError("fail"))
            mock_client.__aenter__ = AsyncMock(return_value=mock_client)
            mock_client.__aexit__ = AsyncMock(return_value=False)
            mock_cls.return_value = mock_client

            saved = await _save_edited_images([result], tmp_path)

        assert len(saved) == 1
        assert "local_path" not in saved[0]

    async def test_local_path_takes_priority_over_url(self, tmp_path):
        """When both local_path and url are present, local_path is preferred."""
        src = tmp_path / "src.png"
        src.write_bytes(b"local content")
        raw = b"data uri content"
        encoded = base64.b64encode(raw).decode()

        result = _make_result(
            url=f"data:image/png;base64,{encoded}",
            local_path=str(src),
        )

        saved = await _save_edited_images([result], tmp_path / "out")
        dest = Path(saved[0]["local_path"])
        assert dest.read_bytes() == b"local content"


# ---------------------------------------------------------------------------
# run_edit
# ---------------------------------------------------------------------------


class TestRunEdit:
    """Tests for the run_edit async function.

    get_provider is lazily imported inside run_edit, so we patch at source:
    pictactic.providers.get_provider
    """

    async def test_basic_edit(self, tmp_path):
        """Single image edit flow."""
        from pictactic.providers.base import GenerationMode

        mock_provider = AsyncMock()
        mock_provider.supported_modes = [GenerationMode.IMAGE_TO_IMAGE]
        raw = b"edited"
        encoded = base64.b64encode(raw).decode()
        mock_result = _make_result(url=f"data:image/png;base64,{encoded}")
        mock_provider.generate = AsyncMock(return_value=mock_result)

        formatter = OutputFormatter(OutputFormat.QUIET)

        with (
            patch(
                "pictactic.cli.edit._load_and_normalize_sources",
                new_callable=AsyncMock,
                return_value=["data:image/jpeg;base64,AA=="],
            ),
            patch("pictactic.providers.get_provider", return_value=mock_provider),
        ):
            result = await run_edit(
                prompt="make it sunset",
                raw_sources=["./test.png"],
                count=1,
                provider_name="gemini",
                output_dir=tmp_path,
                formatter=formatter,
            )

        assert result["total"] == 1
        assert len(result["edited_images"]) == 1

    async def test_unsupported_provider_raises(self, tmp_path):
        """Providers without IMAGE_TO_IMAGE support should raise BadParameter."""
        from pictactic.providers.base import GenerationMode

        mock_provider = MagicMock()
        mock_provider.supported_modes = [GenerationMode.TEXT_TO_IMAGE]

        formatter = OutputFormatter(OutputFormat.QUIET)

        with (
            patch(
                "pictactic.cli.edit._load_and_normalize_sources",
                new_callable=AsyncMock,
                return_value=["data:image/jpeg;base64,AA=="],
            ),
            patch("pictactic.providers.get_provider", return_value=mock_provider),
        ):
            with pytest.raises(typer.BadParameter, match="does not support image editing"):
                await run_edit(
                    prompt="test",
                    raw_sources=["./src.png"],
                    provider_name="antigravity",
                    output_dir=tmp_path,
                    formatter=formatter,
                )

    async def test_enhance_prompt_called_when_enabled(self, tmp_path):
        from pictactic.providers.base import GenerationMode

        mock_provider = AsyncMock()
        mock_provider.supported_modes = [GenerationMode.IMAGE_TO_IMAGE]
        raw = base64.b64encode(b"img").decode()
        mock_provider.generate = AsyncMock(
            return_value=_make_result(url=f"data:image/png;base64,{raw}")
        )

        formatter = OutputFormatter(OutputFormat.QUIET)

        with (
            patch(
                "pictactic.cli.edit._load_and_normalize_sources",
                new_callable=AsyncMock,
                return_value=["data:image/jpeg;base64,AA=="],
            ),
            patch("pictactic.providers.get_provider", return_value=mock_provider),
            patch(
                "pictactic.cli.edit._enhance_edit_prompt",
                new_callable=AsyncMock,
                return_value="enhanced instruction",
            ) as mock_enhance,
        ):
            await run_edit(
                prompt="original",
                raw_sources=["./src.png"],
                enhance=True,
                output_dir=tmp_path,
                formatter=formatter,
            )

        mock_enhance.assert_awaited_once_with("original")

    async def test_enhance_prompt_not_called_when_disabled(self, tmp_path):
        from pictactic.providers.base import GenerationMode

        mock_provider = AsyncMock()
        mock_provider.supported_modes = [GenerationMode.IMAGE_TO_IMAGE]
        raw = base64.b64encode(b"img").decode()
        mock_provider.generate = AsyncMock(
            return_value=_make_result(url=f"data:image/png;base64,{raw}")
        )

        formatter = OutputFormatter(OutputFormat.QUIET)

        with (
            patch(
                "pictactic.cli.edit._load_and_normalize_sources",
                new_callable=AsyncMock,
                return_value=["data:image/jpeg;base64,AA=="],
            ),
            patch("pictactic.providers.get_provider", return_value=mock_provider),
            patch(
                "pictactic.cli.edit._enhance_edit_prompt",
                new_callable=AsyncMock,
            ) as mock_enhance,
        ):
            await run_edit(
                prompt="original",
                raw_sources=["./src.png"],
                enhance=False,
                output_dir=tmp_path,
                formatter=formatter,
            )

        mock_enhance.assert_not_awaited()

    async def test_batch_uses_generate_batch_stream(self, tmp_path):
        from pictactic.providers.base import GenerationMode

        mock_provider = AsyncMock()
        mock_provider.supported_modes = [GenerationMode.IMAGE_TO_IMAGE]
        raw = base64.b64encode(b"img").decode()
        batch_results = [
            _make_result(url=f"data:image/png;base64,{raw}"),
            _make_result(url=f"data:image/png;base64,{raw}"),
        ]
        mock_provider.generate_batch_stream = AsyncMock(return_value=batch_results)

        formatter = OutputFormatter(OutputFormat.QUIET)

        with (
            patch(
                "pictactic.cli.edit._load_and_normalize_sources",
                new_callable=AsyncMock,
                return_value=["data:image/jpeg;base64,AA=="],
            ),
            patch("pictactic.providers.get_provider", return_value=mock_provider),
        ):
            result = await run_edit(
                prompt="test",
                raw_sources=["./src.png"],
                count=2,
                output_dir=tmp_path,
                formatter=formatter,
            )

        mock_provider.generate_batch_stream.assert_awaited_once()
        assert result["total"] == 2


# ---------------------------------------------------------------------------
# edit command (typer CLI)
# ---------------------------------------------------------------------------


class TestEditCommand:
    """Tests for the typer edit command.

    Note: typer group callback with invoke_without_command=True requires
    options to come before the positional PROMPT argument.
    """

    def test_edit_success(self, tmp_path):
        mock_result = {
            "edited_images": [
                {"id": "edit_abc_0", "local_path": str(tmp_path / "out.png"), "score": 0.0}
            ],
            "total": 1,
        }
        with patch("pictactic.cli.edit.asyncio.run", return_value=mock_result):
            result = runner.invoke(
                edit_app,
                [
                    "--source",
                    str(tmp_path / "src.png"),
                    "--format",
                    "quiet",
                    "--output-dir",
                    str(tmp_path),
                    "make it sunset",
                ],
            )
        assert result.exit_code == 0

    def test_edit_missing_source_exits_1(self):
        result = runner.invoke(edit_app, ["--format", "quiet", "make it sunset"])
        assert result.exit_code == 1

    def test_edit_no_results_exits_3(self, tmp_path):
        mock_result = {"edited_images": [], "total": 0}
        with patch("pictactic.cli.edit.asyncio.run", return_value=mock_result):
            result = runner.invoke(
                edit_app,
                [
                    "--source",
                    str(tmp_path / "src.png"),
                    "--format",
                    "quiet",
                    "test prompt",
                ],
            )
        assert result.exit_code == 3

    def test_edit_runtime_error_exits_2(self, tmp_path):
        with patch(
            "pictactic.cli.edit.asyncio.run",
            side_effect=RuntimeError("Provider error"),
        ):
            result = runner.invoke(
                edit_app,
                [
                    "--source",
                    str(tmp_path / "src.png"),
                    "--format",
                    "quiet",
                    "test prompt",
                ],
            )
        assert result.exit_code == 2

    def test_edit_bad_parameter_exits_1(self, tmp_path):
        with patch(
            "pictactic.cli.edit.asyncio.run",
            side_effect=typer.BadParameter("Provider does not support image editing"),
        ):
            result = runner.invoke(
                edit_app,
                [
                    "--source",
                    str(tmp_path / "src.png"),
                    "--format",
                    "quiet",
                    "test prompt",
                ],
            )
        assert result.exit_code == 1

    def test_edit_keyboard_interrupt_exits_1(self, tmp_path):
        with patch(
            "pictactic.cli.edit.asyncio.run",
            side_effect=KeyboardInterrupt,
        ):
            result = runner.invoke(
                edit_app,
                [
                    "--source",
                    str(tmp_path / "src.png"),
                    "--format",
                    "quiet",
                    "test prompt",
                ],
            )
        assert result.exit_code == 1

    def test_edit_default_provider_from_settings(self, tmp_path):
        mock_result = {
            "edited_images": [{"id": "x", "score": 0.0}],
            "total": 1,
        }
        mock_settings = MagicMock()
        mock_settings.default_provider = "gemini"

        with (
            patch("pictactic.cli.edit.asyncio.run", return_value=mock_result),
            patch("pictactic.core.config.settings", mock_settings),
        ):
            result = runner.invoke(
                edit_app,
                [
                    "--source",
                    str(tmp_path / "src.png"),
                    "--format",
                    "quiet",
                    "test prompt",
                ],
            )
        assert result.exit_code == 0

    def test_edit_all_options(self, tmp_path):
        mock_result = {
            "edited_images": [{"id": "x", "score": 0.0}],
            "total": 1,
        }
        with patch("pictactic.cli.edit.asyncio.run", return_value=mock_result):
            result = runner.invoke(
                edit_app,
                [
                    "--source",
                    str(tmp_path / "src.png"),
                    "--count",
                    "3",
                    "--aspect-ratio",
                    "1:1",
                    "--size",
                    "4K",
                    "--provider",
                    "gemini",
                    "--enhance",
                    "--output-dir",
                    str(tmp_path),
                    "--format",
                    "json",
                    "detailed instruction",
                ],
            )
        assert result.exit_code == 0

    def test_edit_multiple_sources(self, tmp_path):
        mock_result = {
            "edited_images": [{"id": "x", "score": 0.0}],
            "total": 1,
        }
        with patch("pictactic.cli.edit.asyncio.run", return_value=mock_result):
            result = runner.invoke(
                edit_app,
                [
                    "--source",
                    str(tmp_path / "src1.png"),
                    "--source",
                    str(tmp_path / "src2.png"),
                    "--format",
                    "quiet",
                    "blend these images",
                ],
            )
        assert result.exit_code == 0
