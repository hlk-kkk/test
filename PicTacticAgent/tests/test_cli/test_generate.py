"""Tests for CLI generate command."""

import base64
import json
from pathlib import Path
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
import typer
from typer.testing import CliRunner

from pictactic.cli.generate import (
    generate_app,
    load_template_images,
    save_images,
    run_generation,
)
from pictactic.cli.output import OutputFormat, OutputFormatter


runner = CliRunner()

# Mock paths for lazy imports in run_generation (local imports at source modules)
_WORKFLOW_PATCH = "pictactic.agents.workflow.create_generation_workflow"
_STATE_PATCH = "pictactic.agents.state.create_initial_state"


class TestLoadTemplateImages:
    """Tests for load_template_images."""

    def test_load_png(self, tmp_path):
        img_file = tmp_path / "test.png"
        img_file.write_bytes(b"\x89PNG\r\n\x1a\n")
        result = load_template_images([img_file])
        assert len(result) == 1
        assert result[0].startswith("data:image/png;base64,")

    def test_load_jpeg(self, tmp_path):
        img_file = tmp_path / "test.jpg"
        img_file.write_bytes(b"\xff\xd8\xff\xe0")
        result = load_template_images([img_file])
        assert result[0].startswith("data:image/jpeg;base64,")

    def test_load_webp(self, tmp_path):
        img_file = tmp_path / "test.webp"
        img_file.write_bytes(b"RIFF\x00\x00\x00\x00WEBP")
        result = load_template_images([img_file])
        assert result[0].startswith("data:image/webp;base64,")

    def test_load_multiple_images(self, tmp_path):
        files = []
        for i in range(3):
            f = tmp_path / f"img{i}.png"
            f.write_bytes(b"test data")
            files.append(f)
        result = load_template_images(files)
        assert len(result) == 3

    def test_file_not_found(self, tmp_path):
        missing = tmp_path / "nonexistent.png"
        with pytest.raises(typer.BadParameter, match="Template image not found"):
            load_template_images([missing])

    def test_base64_encoding(self, tmp_path):
        img_file = tmp_path / "test.png"
        raw_data = b"hello world image data"
        img_file.write_bytes(raw_data)
        result = load_template_images([img_file])
        _, encoded_part = result[0].split(",", 1)
        decoded = base64.b64decode(encoded_part)
        assert decoded == raw_data

    def test_unknown_extension_defaults_to_png(self, tmp_path):
        img_file = tmp_path / "test.bmp"
        img_file.write_bytes(b"BM data")
        result = load_template_images([img_file])
        assert result[0].startswith("data:image/png;base64,")


class TestSaveImages:
    """Tests for save_images."""

    async def test_save_data_uri(self, tmp_path):
        raw = b"image pixel data"
        encoded = base64.b64encode(raw).decode()
        images = [{"id": "img-1", "url": f"data:image/png;base64,{encoded}"}]
        result = await save_images(images, tmp_path)
        assert len(result) == 1
        assert result[0]["local_path"] is not None
        saved = Path(result[0]["local_path"])
        assert saved.exists()
        assert saved.read_bytes() == raw

    async def test_save_http_url(self, tmp_path):
        images = [{"id": "img-2", "url": "https://example.com/image.png"}]
        mock_response = MagicMock()
        mock_response.content = b"downloaded image"
        mock_response.raise_for_status = MagicMock()

        with patch("pictactic.cli.generate.httpx.AsyncClient") as mock_client_cls:
            mock_client = AsyncMock()
            mock_client.get = AsyncMock(return_value=mock_response)
            mock_client.__aenter__ = AsyncMock(return_value=mock_client)
            mock_client.__aexit__ = AsyncMock(return_value=False)
            mock_client_cls.return_value = mock_client

            result = await save_images(images, tmp_path)
            assert result[0].get("local_path") is not None
            saved = Path(result[0]["local_path"])
            assert saved.read_bytes() == b"downloaded image"

    async def test_unknown_url_scheme(self, tmp_path):
        images = [{"id": "img-3", "url": "ftp://server/img.png"}]
        result = await save_images(images, tmp_path)
        assert len(result) == 1
        assert "local_path" not in result[0]

    async def test_creates_output_dir(self, tmp_path):
        out = tmp_path / "nested" / "dir"
        encoded = base64.b64encode(b"data").decode()
        images = [{"id": "a", "url": f"data:image/png;base64,{encoded}"}]
        await save_images(images, out)
        assert out.exists()

    async def test_uses_uuid_for_missing_id(self, tmp_path):
        encoded = base64.b64encode(b"data").decode()
        images = [{"url": f"data:image/png;base64,{encoded}"}]
        result = await save_images(images, tmp_path)
        assert result[0].get("local_path") is not None
        assert "pictactic_" in result[0]["local_path"]

    async def test_handles_download_error(self, tmp_path):
        import httpx as _httpx

        images = [{"id": "fail", "url": "https://example.com/bad.png"}]
        with patch("pictactic.cli.generate.httpx.AsyncClient") as mock_client_cls:
            mock_client = AsyncMock()
            mock_client.get = AsyncMock(
                side_effect=_httpx.HTTPError("Connection error")
            )
            mock_client.__aenter__ = AsyncMock(return_value=mock_client)
            mock_client.__aexit__ = AsyncMock(return_value=False)
            mock_client_cls.return_value = mock_client

            result = await save_images(images, tmp_path)
            assert len(result) == 1


class TestRunGeneration:
    """Tests for run_generation async function."""

    async def test_basic_run(self, tmp_path):
        """Test basic generation workflow run."""
        mock_workflow = AsyncMock()

        async def mock_stream(*args, **kwargs):
            yield {"type": "status_change", "status": "enhancing", "round": 1}
            yield {"type": "image_ready", "image": {
                "id": "img-1", "url": "data:image/png;base64,AA==",
                "score": 0.85, "round_number": 1,
            }}

        mock_workflow.astream = mock_stream

        mock_snapshot = MagicMock()
        mock_snapshot.values = {
            "best_images": [
                {"id": "img-1", "url": "data:image/png;base64,AA==", "score": 0.85},
            ],
        }
        mock_workflow.aget_state = AsyncMock(return_value=mock_snapshot)

        formatter = OutputFormatter(OutputFormat.QUIET)

        with patch(_WORKFLOW_PATCH, return_value=mock_workflow), \
             patch(_STATE_PATCH, return_value={"messages": []}):
            result = await run_generation(
                prompt="test prompt",
                output_dir=tmp_path,
                formatter=formatter,
            )

        assert result["total_images"] == 1
        assert len(result["best_images"]) == 1

    async def test_run_with_no_best_images(self, tmp_path):
        """When best_images is empty, falls back to all collected images."""
        mock_workflow = AsyncMock()

        async def mock_stream(*args, **kwargs):
            yield {"type": "image_ready", "image": {
                "id": "img-1", "url": "data:image/png;base64,AA==",
                "score": 0.5, "round_number": 1,
            }}

        mock_workflow.astream = mock_stream

        mock_snapshot = MagicMock()
        mock_snapshot.values = {"best_images": []}
        mock_workflow.aget_state = AsyncMock(return_value=mock_snapshot)

        formatter = OutputFormatter(OutputFormat.QUIET)

        with patch(_WORKFLOW_PATCH, return_value=mock_workflow), \
             patch(_STATE_PATCH, return_value={"messages": []}):
            result = await run_generation(
                prompt="test",
                output_dir=tmp_path,
                formatter=formatter,
            )

        assert result["total_images"] == 1
        assert len(result["best_images"]) >= 1

    async def test_run_shows_config(self, tmp_path):
        """Verify formatter.show_config is called with expected keys."""
        mock_workflow = AsyncMock()

        async def mock_stream(*args, **kwargs):
            return
            yield

        mock_workflow.astream = mock_stream

        mock_snapshot = MagicMock()
        mock_snapshot.values = {"best_images": []}
        mock_workflow.aget_state = AsyncMock(return_value=mock_snapshot)

        formatter = MagicMock(spec=OutputFormatter)

        with patch(_WORKFLOW_PATCH, return_value=mock_workflow), \
             patch(_STATE_PATCH, return_value={"messages": []}):
            await run_generation(
                prompt="my prompt",
                provider="jiekouai",
                output_dir=tmp_path,
                formatter=formatter,
            )

        formatter.show_config.assert_called_once()
        config_arg = formatter.show_config.call_args[0][0]
        assert config_arg["Prompt"] == "my prompt"
        assert config_arg["Provider"] == "jiekouai"

    async def test_run_handles_non_dict_events(self, tmp_path):
        """Non-dict events from workflow stream should be ignored."""
        mock_workflow = AsyncMock()

        async def mock_stream(*args, **kwargs):
            yield "not a dict"
            yield 42
            yield None

        mock_workflow.astream = mock_stream

        mock_snapshot = MagicMock()
        mock_snapshot.values = {"best_images": []}
        mock_workflow.aget_state = AsyncMock(return_value=mock_snapshot)

        formatter = OutputFormatter(OutputFormat.QUIET)

        with patch(_WORKFLOW_PATCH, return_value=mock_workflow), \
             patch(_STATE_PATCH, return_value={"messages": []}):
            result = await run_generation(
                prompt="test",
                output_dir=tmp_path,
                formatter=formatter,
            )
        assert result["total_images"] == 0

    async def test_run_tracks_rounds(self, tmp_path):
        """Verify round tracking via status_change events."""
        mock_workflow = AsyncMock()

        async def mock_stream(*args, **kwargs):
            yield {"type": "status_change", "status": "generating", "round": 1}
            yield {"type": "status_change", "status": "evaluating", "round": 1}
            yield {"type": "status_change", "status": "generating", "round": 2}

        mock_workflow.astream = mock_stream
        mock_snapshot = MagicMock()
        mock_snapshot.values = {"best_images": []}
        mock_workflow.aget_state = AsyncMock(return_value=mock_snapshot)

        formatter = MagicMock(spec=OutputFormatter)

        with patch(_WORKFLOW_PATCH, return_value=mock_workflow), \
             patch(_STATE_PATCH, return_value={"messages": []}):
            await run_generation(
                prompt="test",
                output_dir=tmp_path,
                formatter=formatter,
            )

        round_calls = [c[0][0] for c in formatter.show_round_start.call_args_list]
        assert 1 in round_calls
        assert 2 in round_calls

    async def test_run_null_snapshot(self, tmp_path):
        """Handle case where aget_state returns None."""
        mock_workflow = AsyncMock()

        async def mock_stream(*args, **kwargs):
            return
            yield

        mock_workflow.astream = mock_stream
        mock_workflow.aget_state = AsyncMock(return_value=None)

        formatter = OutputFormatter(OutputFormat.QUIET)

        with patch(_WORKFLOW_PATCH, return_value=mock_workflow), \
             patch(_STATE_PATCH, return_value={"messages": []}):
            result = await run_generation(
                prompt="test",
                output_dir=tmp_path,
                formatter=formatter,
            )
        assert result["best_images"] == []


class TestGenerateCommand:
    """Tests for the typer generate command.

    Note: typer group callback with invoke_without_command=True requires
    options to come before the positional PROMPT argument in the args list.
    """

    def test_generate_success(self, tmp_path):
        """Test generate command with successful result."""
        mock_result = {
            "best_images": [{"id": "img-1", "local_path": str(tmp_path / "img.png"), "score": 0.9}],
            "total_images": 5,
            "final_state": {},
        }

        with patch("pictactic.cli.generate.asyncio.run", return_value=mock_result):
            result = runner.invoke(
                generate_app,
                ["--format", "quiet", "--output-dir", str(tmp_path), "test prompt"],
            )
            assert result.exit_code == 0

    def test_generate_no_results_exit_code_3(self, tmp_path):
        """Exit code 3 when no best images are generated."""
        mock_result = {
            "best_images": [],
            "total_images": 0,
            "final_state": {},
        }

        with patch("pictactic.cli.generate.asyncio.run", return_value=mock_result):
            result = runner.invoke(
                generate_app,
                ["--format", "quiet", "--output-dir", str(tmp_path), "test prompt"],
            )
            assert result.exit_code == 3

    def test_generate_runtime_error_exit_code_2(self, tmp_path):
        """Exit code 2 on unexpected error."""
        with patch(
            "pictactic.cli.generate.asyncio.run",
            side_effect=RuntimeError("Provider error"),
        ):
            result = runner.invoke(
                generate_app,
                ["--format", "quiet", "--output-dir", str(tmp_path), "test prompt"],
            )
            assert result.exit_code == 2

    def test_generate_keyboard_interrupt(self, tmp_path):
        """Exit code 1 on keyboard interrupt."""
        with patch(
            "pictactic.cli.generate.asyncio.run",
            side_effect=KeyboardInterrupt,
        ):
            result = runner.invoke(
                generate_app,
                ["--format", "quiet", "--output-dir", str(tmp_path), "test prompt"],
            )
            assert result.exit_code == 1

    def test_generate_template_not_found(self, tmp_path):
        """Exit code 1 when template image file doesn't exist."""
        result = runner.invoke(
            generate_app,
            [
                "--template", str(tmp_path / "nonexistent.png"),
                "--format", "quiet",
                "test prompt",
            ],
        )
        assert result.exit_code == 1

    def test_generate_with_valid_template(self, tmp_path):
        """Generate with valid template image."""
        tmpl = tmp_path / "template.png"
        tmpl.write_bytes(b"\x89PNG\r\n\x1a\n")

        mock_result = {
            "best_images": [{"id": "img-1", "score": 0.9}],
            "total_images": 1,
            "final_state": {},
        }

        with patch("pictactic.cli.generate.asyncio.run", return_value=mock_result):
            result = runner.invoke(
                generate_app,
                [
                    "--template", str(tmpl),
                    "--format", "quiet",
                    "--output-dir", str(tmp_path),
                    "test prompt",
                ],
            )
            assert result.exit_code == 0

    def test_generate_default_provider_from_settings(self, tmp_path):
        """When --provider is not specified, uses settings.default_provider."""
        mock_result = {
            "best_images": [{"id": "img-1", "score": 0.9}],
            "total_images": 1,
            "final_state": {},
        }

        mock_settings = MagicMock()
        mock_settings.default_provider = "jiekouai"

        with patch(
            "pictactic.cli.generate.asyncio.run", return_value=mock_result
        ), patch(
            "pictactic.core.config.settings", mock_settings
        ):
            result = runner.invoke(
                generate_app,
                ["--format", "quiet", "--output-dir", str(tmp_path), "test prompt"],
            )
            assert result.exit_code == 0

    def test_generate_explicit_provider(self, tmp_path):
        """When --provider is specified, uses that value."""
        mock_result = {
            "best_images": [{"id": "img-1", "score": 0.9}],
            "total_images": 1,
            "final_state": {},
        }

        with patch("pictactic.cli.generate.asyncio.run", return_value=mock_result):
            result = runner.invoke(
                generate_app,
                [
                    "--provider", "antigravity",
                    "--format", "quiet",
                    "--output-dir", str(tmp_path),
                    "test prompt",
                ],
            )
            assert result.exit_code == 0

    def test_generate_all_options(self, tmp_path):
        """Test all CLI options together."""
        mock_result = {
            "best_images": [{"id": "x", "score": 0.8}],
            "total_images": 1,
            "final_state": {},
        }

        with patch("pictactic.cli.generate.asyncio.run", return_value=mock_result):
            result = runner.invoke(
                generate_app,
                [
                    "--rounds", "5",
                    "--images", "3",
                    "--threshold", "0.8",
                    "--aspect-ratio", "1:1",
                    "--size", "4K",
                    "--provider", "gemini",
                    "--no-enhance",
                    "--no-ai-scoring",
                    "--output-dir", str(tmp_path),
                    "--format", "json",
                    "a detailed prompt",
                ],
            )
            assert result.exit_code == 0

    def test_generate_json_output(self, tmp_path):
        """Test JSON format output from generate command."""
        mock_result = {
            "best_images": [{"id": "img-1", "local_path": "/tmp/img.png", "score": 0.9}],
            "total_images": 3,
            "final_state": {},
        }

        with patch("pictactic.cli.generate.asyncio.run", return_value=mock_result):
            result = runner.invoke(
                generate_app,
                ["--format", "json", "--output-dir", str(tmp_path), "test prompt"],
            )
            assert result.exit_code == 0
            lines = [ln for ln in result.output.strip().split("\n") if ln.strip()]
            if lines:
                first = json.loads(lines[0])
                assert first["event"] == "start"
