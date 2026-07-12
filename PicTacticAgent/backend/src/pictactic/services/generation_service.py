"""Generation service for managing image generation tasks."""

import asyncio
import logging
import uuid
from copy import deepcopy
from datetime import datetime, timezone

from ..agents.workflow import create_generation_workflow
from ..core.exceptions import ImageEditConflictError, ProviderError, TaskNotFoundError
from ..db.repository import TaskRepository
from ..i18n import get_text
from ..models.enums import GenerationStatus, ImageProvider
from ..models.schemas import (
    GeneratedImage,
    GenerationRequest,
    GenerationResult,
    ImageEditRequest,
    ImageEditResult,
    StandaloneEditRequest,
    TaskProgress,
)
from .thumbnail import get_thumbnail_url

logger = logging.getLogger(__name__)

_MAX_EDIT_REFERENCE_LONG_EDGE = 2048


class GenerationService:
    """Service for managing image generation tasks."""

    def __init__(self):
        self._tasks: dict[str, dict] = {}
        self._running_tasks: dict[str, asyncio.Task] = {}
        self._editing_images: set[str] = set()
        self._pending_status_updates: dict[str, tuple[str, str | None]] = {}
        self._status_flush_tasks: dict[str, asyncio.Task] = {}
        self._repo = TaskRepository()
        self._workflow = create_generation_workflow()

    async def _ensure_task_loaded(
        self,
        task_id: str,
        user_id: str | None = None,
    ) -> dict:
        """Ensure task exists in memory; restore from DB when necessary."""
        task_info = self._tasks.get(task_id)
        if task_info:
            if user_id and task_info.get("user_id") != user_id:
                raise TaskNotFoundError(task_id)
            return task_info

        try:
            persisted = await self._repo.get_task_result(task_id, user_id=user_id)
        except RuntimeError as exc:
            # In some unit-test/runtime edge cases DB may be unavailable; treat as not found.
            raise TaskNotFoundError(task_id) from exc
        if not persisted:
            raise TaskNotFoundError(task_id)

        config = persisted.config or {}

        def _img_to_dict(img: GeneratedImage) -> dict:
            score_details = None
            if img.score_details:
                score_details = (
                    img.score_details.model_dump()
                    if hasattr(img.score_details, "model_dump")
                    else img.score_details
                )
            return {
                "id": img.id,
                "url": img.url,
                "local_path": img.local_path,
                "score": img.score,
                "review": "",
                "round_number": img.round_number,
                "provider": img.provider.value
                if hasattr(img.provider, "value")
                else str(img.provider),
                "score_details": score_details,
                "metadata": img.metadata or {},
            }

        generated_images = [
            _img_to_dict(image)
            for round_result in persisted.rounds
            for image in round_result.images
        ]
        best_images = [_img_to_dict(image) for image in persisted.best_images]
        current_round = persisted.total_rounds or max(
            (img.get("round_number", 0) for img in generated_images),
            default=0,
        )
        provider_value = config.get("provider") or ImageProvider.GEMINI.value
        try:
            provider_enum = ImageProvider(provider_value)
        except Exception:
            provider_enum = ImageProvider.GEMINI

        request = GenerationRequest(
            prompt=persisted.original_prompt or "",
            template_images=persisted.template_images or [],
            max_rounds=max(int(persisted.total_rounds or 1), 1),
            images_per_round=int(config.get("images_per_round") or 5),
            quality_threshold=float(config.get("quality_threshold") or 0.7),
            enable_ai_scoring=bool(config.get("enable_ai_scoring", True)),
            aspect_ratio=config.get("aspect_ratio") or "16:9",
            size=config.get("size") or "2K",
            enhance_prompt=bool(config.get("enhance_prompt", True)),
            provider=provider_enum,
        )

        self._tasks[task_id] = {
            "id": task_id,
            "user_id": user_id,
            "status": persisted.status,
            "request": request,
            "state": {
                "messages": [],
                "user_prompt": persisted.original_prompt or "",
                "enhanced_prompt": persisted.enhanced_prompt,
                "template_images": persisted.template_images or [],
                "template_analysis": (
                    persisted.template_analysis.model_dump()
                    if persisted.template_analysis
                    and hasattr(persisted.template_analysis, "model_dump")
                    else persisted.template_analysis
                ),
                "generated_images": generated_images,
                "best_images": best_images,
                "current_round": current_round,
                "status": persisted.status.value
                if hasattr(persisted.status, "value")
                else str(persisted.status),
                "error": persisted.error,
                "max_rounds": max(int(persisted.total_rounds or 1), 1),
                "images_per_round": int(config.get("images_per_round") or 5),
                "aspect_ratio": config.get("aspect_ratio") or "16:9",
                "image_size": config.get("size") or "2K",
                "provider": provider_enum.value,
                "quality_threshold": float(config.get("quality_threshold") or 0.7),
            },
            "created_at": persisted.started_at or datetime.now(timezone.utc),
            "updated_at": persisted.completed_at or datetime.now(timezone.utc),
            "result": persisted,
            "error": persisted.error,
        }
        return self._tasks[task_id]

    async def create_task(self, request: GenerationRequest, user_id: str | None = None) -> str:
        """Create a new generation task."""
        task_id = str(uuid.uuid4())

        self._tasks[task_id] = {
            "id": task_id,
            "user_id": user_id,
            "status": GenerationStatus.PENDING,
            "request": request,
            "state": None,
            "created_at": datetime.now(timezone.utc),
            "updated_at": datetime.now(timezone.utc),
            "result": None,
            "error": None,
        }

        # Persist base64 template images as files, store only URLs in DB
        persisted_template_urls = await self._persist_template_images(
            request.template_images
        )

        try:
            await self._repo.create_task(
                task_id=task_id,
                original_prompt=request.prompt,
                template_images=persisted_template_urls,
                config={
                    "max_rounds": request.max_rounds,
                    "images_per_round": request.images_per_round,
                    "aspect_ratio": request.aspect_ratio,
                    "size": request.size,
                    "provider": request.provider.value,
                },
                user_id=user_id,
            )
        except Exception:
            logger.warning("Failed to persist task %s to DB", task_id, exc_info=True)
            self._tasks.pop(task_id, None)
            raise

        return task_id

    async def _persist_template_images(self, template_images: list[str]) -> list[str]:
        """Save base64 template images to disk and return URL list for DB storage."""
        if not template_images:
            return []

        import base64
        import io
        from pathlib import Path

        from PIL import Image

        from ..core.config import settings
        from ..services.thumbnail import generate_thumbnail

        urls: list[str] = []
        for i, img_data in enumerate(template_images):
            if not img_data.startswith("data:"):
                urls.append(img_data)
                continue

            try:
                # Parse data URI: data:image/jpeg;base64,/9j/4...
                header, b64_data = img_data.split(",", 1)
                image_bytes = base64.b64decode(b64_data)

                storage_path = Path(settings.storage_path)
                storage_path.mkdir(parents=True, exist_ok=True)

                image_id = str(uuid.uuid4())[:8]
                filename = f"template_{image_id}_{i}.jpg"
                file_path = storage_path / filename

                img = Image.open(io.BytesIO(image_bytes))
                if img.mode in ("RGBA", "LA", "P"):
                    img = img.convert("RGB")
                img.save(str(file_path), format="JPEG", quality=90, optimize=True)

                await asyncio.to_thread(generate_thumbnail, str(file_path))
                urls.append(f"/static/images/{filename}")
            except Exception:
                logger.warning("Failed to persist template image %d", i, exc_info=True)
                urls.append(img_data)

        return urls

    async def get_task_owner(self, task_id: str) -> str | None:
        """Get the owner user_id for a task."""
        if task_id in self._tasks:
            return self._tasks[task_id].get("user_id")
        return await self._repo.get_task_owner(task_id)

    async def start_task(self, task_id: str) -> None:
        """Start executing a generation task."""
        if task_id not in self._tasks:
            raise TaskNotFoundError(task_id)

        task_info = self._tasks[task_id]
        request: GenerationRequest = task_info["request"]

        async_task = asyncio.create_task(self._run_task(task_id, request))
        self._running_tasks[task_id] = async_task

    async def _run_task(self, task_id: str, request: GenerationRequest) -> None:
        """Run the generation workflow for a task."""
        try:
            self._update_task_status(task_id, GenerationStatus.ENHANCING)

            from ..api.websocket import get_connection_manager

            manager = get_connection_manager()

            config = {"configurable": {"thread_id": task_id}}

            # Build initial state for the workflow
            template_images = request.template_images or []
            initial_state = {
                "messages": [],
                "user_prompt": request.prompt,
                "template_images": template_images,
                "template_analysis": None,
                "enhanced_prompt": None,
                "generated_images": [],
                "best_images": [],
                "current_round": 0,
                "evaluation_feedback": "",
                "status": "pending",
                "final_result": None,
                "error": None,
                "max_rounds": request.max_rounds,
                "images_per_round": request.images_per_round,
                "aspect_ratio": request.aspect_ratio,
                "image_size": request.size,
                "provider": request.provider.value if request.provider else "gemini",
                "enhance_prompt_enabled": request.enhance_prompt,
                "enable_ai_scoring": request.enable_ai_scoring,
                "quality_threshold": request.quality_threshold,
            }
            self._tasks[task_id]["state"] = deepcopy(initial_state)

            logger.info(
                "Starting workflow for task %s: max_rounds=%d, images_per_round=%d",
                task_id,
                request.max_rounds,
                request.images_per_round,
            )

            # Stream custom events from the workflow
            prev_round = 0
            async for event in self._workflow.astream(
                initial_state,
                config=config,
                stream_mode="custom",
            ):
                if isinstance(event, dict) and event.get("type") == "status_change":
                    self._apply_status_change(task_id, event)
                elif isinstance(event, dict) and event.get("type") == "image_ready":
                    self._update_task_status(task_id, GenerationStatus.GENERATING)
                    image_info = event.get("image", {})
                    self._apply_image_ready(task_id, image_info)


                    img_url = image_info.get("url", "")
                    image_data = {
                        "type": "image_ready",
                        "image": {
                            "id": image_info.get("id", ""),
                            "url": img_url,
                            "thumbnail_url": get_thumbnail_url(img_url) if img_url else None,
                            "local_path": image_info.get("local_path"),
                            "round_number": image_info.get("round_number", 1),
                            "index": 0,
                            "score": image_info.get("score", 0.0),
                            "review": image_info.get("review", ""),
                            "provider": image_info.get("provider", ""),
                            "metadata": image_info.get("metadata", {}),
                        },
                    }
                    await manager.send_progress(task_id, image_data)

                self._tasks[task_id]["updated_at"] = datetime.now(timezone.utc)

                # Persist intermediate round results to DB
                cur_state = self._tasks[task_id].get("state") or {}
                cur_round = cur_state.get("current_round", 0)
                if cur_round > prev_round:
                    prev_round = cur_round
                    try:
                        intermediate = self._build_result(task_id, cur_state)
                        await self._repo.save_result(intermediate)
                    except Exception:
                        logger.warning(
                            "Failed to persist intermediate round for task %s",
                            task_id,
                            exc_info=True,
                        )

            # Get final state
            final_snapshot = await self._workflow.aget_state(config)
            final_state = final_snapshot.values if final_snapshot else {}

            self._tasks[task_id]["state"] = final_state

            # Determine final status BEFORE building result so _build_result
            # captures the correct status for DB persistence.
            final_result = final_state.get("final_result", {})
            if isinstance(final_result, dict) and final_result.get("success") is False:
                self._update_task_status(task_id, GenerationStatus.FAILED)
            elif final_state.get("status") == "completed" or final_state.get("best_images"):
                self._update_task_status(task_id, GenerationStatus.COMPLETED)
            else:
                self._update_task_status(task_id, GenerationStatus.FAILED)

            self._tasks[task_id]["result"] = self._build_result(task_id, final_state)

            logger.info(
                "Workflow completed for task %s: status=%s, rounds=%d, images=%d",
                task_id,
                final_state.get("status"),
                final_state.get("current_round", 0),
                len(final_state.get("generated_images", [])),
            )

            try:
                await self._wait_for_status_persist(task_id)
                await self._repo.save_result(self._tasks[task_id]["result"])
            except Exception:
                logger.warning("Failed to save result for task %s to DB", task_id, exc_info=True)

        except asyncio.CancelledError:
            logger.info("Workflow cancelled for task %s", task_id)
            self._tasks[task_id]["error"] = "Task cancelled"
            self._update_task_status(task_id, GenerationStatus.CANCELLED)
            try:
                await self._wait_for_status_persist(task_id)
            except Exception:
                pass

        except Exception as e:
            logger.error("Workflow failed for task %s: %s", task_id, e, exc_info=True)
            self._tasks[task_id]["error"] = str(e)
            self._update_task_status(task_id, GenerationStatus.FAILED)
            try:
                await self._repo.update_status(task_id, GenerationStatus.FAILED.value, str(e))
            except Exception:
                logger.warning("Failed to update status for task %s in DB", task_id, exc_info=True)
            try:
                from ..api.websocket import get_connection_manager

                await get_connection_manager().send_progress(
                    task_id, {"status": "failed", "error": str(e), "task_id": task_id}
                )
            except Exception:
                pass

        finally:
            if task_id in self._running_tasks:
                del self._running_tasks[task_id]
            self._evict_old_tasks()

    def _evict_old_tasks(self, max_size: int = 200) -> None:
        """Remove completed tasks from memory when threshold exceeded."""
        if len(self._tasks) <= max_size:
            return
        now = datetime.now(timezone.utc)
        evictable = []
        for tid, info in self._tasks.items():
            if tid in self._running_tasks:
                continue
            status = info.get("status")
            status_val = status.value if hasattr(status, "value") else str(status)
            if status_val in ("completed", "failed", "cancelled"):
                age = (now - info.get("updated_at", now)).total_seconds()
                if age > 3600:
                    evictable.append((age, tid))
        evictable.sort(reverse=True)
        for _, tid in evictable[: len(self._tasks) - max_size]:
            self._tasks.pop(tid, None)

    @staticmethod
    def _safe_provider(value: str) -> ImageProvider:
        """Safely convert a string to ImageProvider, defaulting to GEMINI."""
        try:
            return ImageProvider(value)
        except ValueError:
            return ImageProvider.GEMINI

    def _update_task_status(self, task_id: str, status: GenerationStatus) -> None:
        """Update task status."""
        if task_id in self._tasks:
            self._tasks[task_id]["status"] = status
            self._tasks[task_id]["updated_at"] = datetime.now(timezone.utc)
            self._schedule_status_persist(task_id)

    def _schedule_status_persist(self, task_id: str) -> None:
        """Coalesce rapid status changes so SQLite is not hit by overlapping writes."""
        if task_id not in self._tasks:
            return

        try:
            loop = asyncio.get_running_loop()
        except RuntimeError:
            return

        status = self._tasks[task_id].get("status")
        status_value = status.value if hasattr(status, "value") else str(status)
        error = self._tasks[task_id].get("error")
        self._pending_status_updates[task_id] = (status_value, error)

        flush_task = self._status_flush_tasks.get(task_id)
        if flush_task is None or flush_task.done():
            self._status_flush_tasks[task_id] = loop.create_task(
                self._flush_status_updates(task_id)
            )

    async def _flush_status_updates(self, task_id: str) -> None:
        """Persist the latest queued status for a task."""
        try:
            while True:
                payload = self._pending_status_updates.pop(task_id, None)
                if payload is None:
                    return

                status, error = payload
                try:
                    await self._repo.update_status(task_id, status, error)
                except Exception:
                    logger.warning(
                        "Failed to persist status for task %s", task_id, exc_info=True
                    )
        finally:
            self._status_flush_tasks.pop(task_id, None)
            if task_id in self._pending_status_updates:
                self._schedule_status_persist(task_id)

    async def _wait_for_status_persist(self, task_id: str, timeout: float = 10.0) -> None:
        """Wait until any queued status write for a task has drained."""
        deadline = asyncio.get_event_loop().time() + timeout
        while True:
            flush_task = self._status_flush_tasks.get(task_id)
            if flush_task is None:
                if task_id not in self._pending_status_updates:
                    return
                if asyncio.get_event_loop().time() >= deadline:
                    logger.warning("Timed out waiting for status persist for task %s", task_id)
                    return
                await asyncio.sleep(0.01)
                continue
            remaining = deadline - asyncio.get_event_loop().time()
            if remaining <= 0:
                logger.warning("Timed out waiting for status persist for task %s", task_id)
                return
            try:
                await asyncio.wait_for(asyncio.shield(flush_task), timeout=remaining)
            except TimeoutError:
                logger.warning("Timed out waiting for status flush for task %s", task_id)
                return

    def _apply_status_change(self, task_id: str, event: dict) -> None:
        """Apply streamed status updates from workflow nodes."""
        if task_id not in self._tasks:
            return

        status_map = {
            "pending": GenerationStatus.PENDING,
            "enhancing": GenerationStatus.ENHANCING,
            "generating": GenerationStatus.GENERATING,
            "reviewing": GenerationStatus.REVIEWING,
            "completed": GenerationStatus.COMPLETED,
            "failed": GenerationStatus.FAILED,
            "cancelled": GenerationStatus.CANCELLED,
        }
        status_key = str(event.get("status", "")).lower()
        mapped = status_map.get(status_key)
        if mapped:
            self._update_task_status(task_id, mapped)

        state = self._tasks[task_id].get("state")
        if not isinstance(state, dict):
            return

        round_value = event.get("current_round")
        if round_value is None:
            return
        try:
            current_round = int(round_value)
        except (TypeError, ValueError):
            return

        state["current_round"] = max(state.get("current_round", 0), current_round)

    def _apply_image_ready(self, task_id: str, image_info: dict) -> None:
        """Merge streamed image-ready events into in-memory task state."""
        if task_id not in self._tasks:
            return

        state = self._tasks[task_id].get("state")
        if not isinstance(state, dict):
            state = {"generated_images": [], "current_round": 0}
            self._tasks[task_id]["state"] = state

        generated_images = state.setdefault("generated_images", [])
        generated_images.append(
            {
                "id": image_info.get("id", str(uuid.uuid4())),
                "url": image_info.get("url", ""),
                "local_path": image_info.get("local_path"),
                "score": image_info.get("score", 0.0),
                "review": image_info.get("review", ""),
                "round_number": image_info.get("round_number", 1),
                "provider": image_info.get("provider", "gemini"),
                "metadata": image_info.get("metadata", {}),
            }
        )

        try:
            round_number = int(image_info.get("round_number", 1))
        except (TypeError, ValueError):
            round_number = 1
        state["current_round"] = max(state.get("current_round", 0), round_number)

    def _build_result(self, task_id: str, state: dict) -> GenerationResult:
        """Build generation result from final state."""
        from ..models.schemas import RoundResult

        task_info = self._tasks[task_id]
        request: GenerationRequest = task_info["request"]

        generated_images = state.get("generated_images", [])
        enhanced_prompt = state.get("enhanced_prompt") or request.prompt


        # 按轮次分组图片
        rounds_dict: dict[int, list[GeneratedImage]] = {}
        for img_data in generated_images:
            round_num = img_data.get("round_number", 1)
            if round_num not in rounds_dict:
                rounds_dict[round_num] = []

            img_url = img_data.get("url", "")
            img = GeneratedImage(
                id=img_data.get("id", str(uuid.uuid4())),
                url=img_url,
                thumbnail_url=get_thumbnail_url(img_url) if img_url else None,
                local_path=img_data.get("local_path"),
                prompt=enhanced_prompt,
                provider=self._safe_provider(img_data.get("provider", "gemini")),
                round_number=round_num,
                score=img_data.get("score", 0.0),
                score_details=img_data.get("score_details"),
                metadata=img_data.get("metadata", {}),
            )
            rounds_dict[round_num].append(img)

        # 构建轮次结果
        rounds = []
        for round_num in sorted(rounds_dict.keys()):
            round_images = rounds_dict[round_num]
            best_score = max((img.score for img in round_images), default=0.0)
            rounds.append(
                RoundResult(
                    round_number=round_num,
                    images=round_images,
                    best_score=best_score,
                    prompt_used=enhanced_prompt,
                    feedback=None,
                )
            )

        # 构建 best_images 列表
        best_images = []
        for img_data in state.get("best_images", []):
            img_url = img_data.get("url", "")
            best_images.append(
                GeneratedImage(
                    id=img_data.get("id", str(uuid.uuid4())),
                    url=img_url,
                    thumbnail_url=get_thumbnail_url(img_url) if img_url else None,
                    local_path=img_data.get("local_path"),
                    prompt=enhanced_prompt,
                    provider=self._safe_provider(img_data.get("provider", "gemini")),
                    round_number=img_data.get("round_number", 1),
                    score=img_data.get("score", 0.0),
                    score_details=img_data.get("score_details"),
                    metadata=img_data.get("metadata", {}),
                )
            )

        # Strip base64 template images from result (only keep URL references)
        result_template_images = [
            t for t in (request.template_images or []) if not t.startswith("data:")
        ]

        return GenerationResult(
            task_id=task_id,
            status=task_info["status"],
            original_prompt=request.prompt,
            enhanced_prompt=state.get("enhanced_prompt"),
            template_images=result_template_images,
            template_analysis=state.get("template_analysis"),
            config={
                "max_rounds": request.max_rounds,
                "images_per_round": request.images_per_round,
                "aspect_ratio": request.aspect_ratio,
                "size": request.size,
                "provider": request.provider.value if request.provider else "gemini",
            },
            rounds=rounds,
            total_rounds=len(rounds),
            total_images=len(generated_images),
            best_images=best_images,
            started_at=task_info["created_at"],
            completed_at=task_info["updated_at"],
            error=task_info.get("error"),
        )

    async def get_task_status(
        self,
        task_id: str,
        lang: str = "en",
        user_id: str | None = None,
    ) -> TaskProgress:
        """Get current status of a task."""
        task_info = await self._ensure_task_loaded(task_id, user_id)
        request: GenerationRequest = task_info["request"]
        state = task_info.get("state") or {}


        recent_images = []
        for img_data in state.get("generated_images", [])[-5:]:
            img_url = img_data.get("url", "")
            recent_images.append(
                GeneratedImage(
                    id=img_data.get("id", ""),
                    url=img_url,
                    thumbnail_url=get_thumbnail_url(img_url) if img_url else None,
                    local_path=img_data.get("local_path"),
                    prompt=state.get("enhanced_prompt") or request.prompt,
                    provider=self._safe_provider(img_data.get("provider", "gemini")),
                    round_number=img_data.get("round_number", 1),
                    score=img_data.get("score", 0.0),
                    metadata=img_data.get("metadata", {}),
                )
            )

        current_best_score = 0.0
        best_images = state.get("best_images", [])
        if best_images:
            current_best_score = max(img.get("score", 0.0) for img in best_images)
        generated_images = state.get("generated_images", [])
        if generated_images:
            current_best_score = max(
                current_best_score,
                max(img.get("score", 0.0) for img in generated_images),
            )

        return TaskProgress(
            task_id=task_id,
            status=task_info["status"],
            current_round=state.get("current_round", 0),
            max_rounds=request.max_rounds,
            images_per_round=request.images_per_round,
            aspect_ratio=request.aspect_ratio,
            size=request.size,
            template_images=[],
            images_generated=len(generated_images),
            images_in_current_round=len(
                [
                    img
                    for img in generated_images
                    if img.get("round_number") == state.get("current_round", 0)
                ]
            ),
            current_best_score=current_best_score,
            quality_threshold=request.quality_threshold,
            message=self._get_status_message(
                task_info["status"], state, task_info.get("error"), lang=lang
            ),
            original_prompt=request.prompt,
            enhanced_prompt=state.get("enhanced_prompt"),
            recent_images=recent_images,
            error=task_info.get("error"),
        )

    def _get_status_message(
        self, status: GenerationStatus, state: dict, error: str | None = None, lang: str = "en"
    ) -> str:
        """Generate a human-readable status message."""
        if status == GenerationStatus.PENDING:
            return get_text("status.pending", lang)
        elif status == GenerationStatus.ENHANCING:
            return get_text("status.enhancing", lang)
        elif status == GenerationStatus.GENERATING:
            round_num = state.get("current_round", 1)
            return get_text("status.generating", lang, round_num=round_num)
        elif status == GenerationStatus.REVIEWING:
            best = max(
                (img.get("score", 0.0) for img in state.get("generated_images", [])),
                default=0.0,
            )
            return get_text("status.reviewing", lang, best=best)
        elif status == GenerationStatus.COMPLETED:
            best = 0.0
            best_images = state.get("best_images", [])
            if best_images:
                best = max(img.get("score", 0.0) for img in best_images)
            return get_text("status.completed", lang, best=best)
        elif status == GenerationStatus.FAILED:
            if error:
                return get_text("status.failed", lang, error=error)
            return get_text("status.failed_generic", lang)
        elif status == GenerationStatus.CANCELLED:
            return get_text("status.cancelled", lang)
        return get_text("status.unknown", lang)

    async def get_task_result(self, task_id: str, user_id: str | None = None) -> GenerationResult:
        """Get the final result of a completed task."""
        if task_id in self._tasks:
            task_info = self._tasks[task_id]
            if user_id and task_info.get("user_id") != user_id:
                raise TaskNotFoundError(task_id)
            if task_info.get("result"):
                return task_info["result"]
            return self._build_result(task_id, task_info.get("state") or {})

        result = await self._repo.get_task_result(task_id, user_id=user_id)
        if result:
            return result
        raise TaskNotFoundError(task_id)

    async def edit_image(
        self,
        task_id: str,
        image_id: str,
        edit_request: ImageEditRequest,
        user_id: str | None = None,
    ) -> ImageEditResult:
        """Edit an existing generated image using image-to-image generation.

        Args:
            task_id: The task that owns the image.
            image_id: ID of the image to edit.
            edit_request: Edit prompt and options.
            user_id: Optional user ID for ownership check.

        Returns:
            ImageEditResult with the new image.
        """
        task_info = await self._ensure_task_loaded(task_id, user_id=user_id)

        # Find the original image in generated_images or best_images
        state = task_info.get("state") or {}
        all_images = state.get("generated_images", []) + state.get("best_images", [])
        original_image = None
        for img in all_images:
            if img.get("id") == image_id:
                original_image = img
                break

        if not original_image:
            from ..core.exceptions import PicTacticError

            raise PicTacticError(f"Image not found: {image_id}")

        # Concurrent edit protection
        if image_id in self._editing_images:
            raise ImageEditConflictError(image_id)
        self._editing_images.add(image_id)

        # Get original image URL/path as reference
        image_source = original_image.get("local_path") or original_image.get("url", "")
        if not image_source:
            self._editing_images.discard(image_id)
            from ..core.exceptions import PicTacticError

            raise PicTacticError(f"Image source not available for: {image_id}")

        # Notify via WebSocket: edit started
        from ..api.websocket import get_connection_manager

        ws_manager = get_connection_manager()
        await ws_manager.send_progress(
            task_id,
            {
                "type": "image_edit_start",
                "image_id": image_id,
            },
        )

        # Optionally enhance the edit prompt
        enhanced_edit_prompt = None
        edit_prompt = edit_request.edit_prompt
        if edit_request.enhance_prompt:
            enhanced_edit_prompt = await self._enhance_edit_prompt(edit_prompt)
            edit_prompt = enhanced_edit_prompt

        # Normalize the original image for use as reference
        from ..agents.tools._image_utils import normalize_reference_images

        try:
            # For edit flow, always normalize to a compact data URI.
            # This avoids proxy-side URL fetch failures and oversized payload issues.
            reference_images = await normalize_reference_images(
                [image_source],
                max_long_edge=_MAX_EDIT_REFERENCE_LONG_EDGE,
                keep_http_urls=False,
            )
        except Exception as e:
            self._editing_images.discard(image_id)
            req = task_info["request"]
            raise ProviderError(
                f"Failed to prepare reference image for editing: {e}",
                provider=req.provider.value if req.provider else "gemini",
            ) from e

        # Get provider and generation config from the task's request
        request: GenerationRequest = task_info["request"]
        provider_name = request.provider.value if request.provider else "gemini"

        from ..providers import get_provider
        from ..providers.base import GenerationParams

        provider_instance = get_provider(provider_name)
        params = GenerationParams(
            prompt=edit_prompt,
            aspect_ratio=request.aspect_ratio,
            size=request.size,
            reference_images=reference_images,
        )

        # Generate the edited image (single image)
        try:
            result = await provider_instance.generate(params, index=0)
        except Exception as e:
            self._editing_images.discard(image_id)
            raise ProviderError(
                f"Image generation failed: {e}",
                provider=provider_name,
            ) from e

        # Build the new image record

        new_image_id = str(uuid.uuid4())
        new_image = GeneratedImage(
            id=new_image_id,
            url=result.url,
            thumbnail_url=get_thumbnail_url(result.url) if result.url else None,
            local_path=result.local_path,
            prompt=edit_prompt,
            provider=self._safe_provider(provider_name),
            round_number=original_image.get("round_number", 1),
            score=0.0,
            metadata={
                **(result.metadata or {}),
                "edited_from": image_id,
                "edit_prompt": edit_request.edit_prompt,
            },
        )

        new_image_dict = {
            "id": new_image_id,
            "url": result.url,
            "local_path": result.local_path,
            "score": 0.0,
            "review": "",
            "round_number": original_image.get("round_number", 1),
            "provider": provider_name,
            "metadata": new_image.metadata,
        }

        # Add to task state's generated_images
        generated_images = state.setdefault("generated_images", [])
        generated_images.append(new_image_dict)

        # Update best_images: replace the original image with the new one
        best_images = state.get("best_images", [])
        for i, img in enumerate(best_images):
            if img.get("id") == image_id:
                best_images[i] = new_image_dict
                break

        # Release edit lock
        self._editing_images.discard(image_id)

        # Notify via WebSocket: edit complete
        await ws_manager.send_progress(
            task_id,
            {
                "type": "image_edit_complete",
                "image_id": image_id,
                "new_image": {
                    "id": new_image_id,
                    "url": result.url,
                    "local_path": result.local_path,
                    "score": 0.0,
                    "round_number": original_image.get("round_number", 1),
                    "provider": provider_name,
                    "metadata": new_image.metadata,
                },
            },
        )

        return ImageEditResult(
            original_image_id=image_id,
            new_image=new_image,
            edit_prompt=edit_request.edit_prompt,
            enhanced_edit_prompt=enhanced_edit_prompt,
        )

    async def _enhance_edit_prompt(self, edit_prompt: str) -> str:
        """Enhance an edit prompt using LLM."""
        from langchain_core.messages import HumanMessage, SystemMessage

        from ..core.config import create_vision_llm

        system_prompt = """You are an expert at crafting image editing prompts.

Your goal is to transform edit instructions into optimized prompts that produce:
- Precise edits matching the user's intent
- High visual quality in the modified areas
- Natural blending with the rest of the image

Guidelines:
- Keep the user's core edit intent
- Add specific visual details for the edit
- Maintain consistency with the original image
- Keep prompts focused and concise (max 100 words)

Output ONLY the enhanced prompt, nothing else."""

        user_message = f"""Enhance this image editing prompt:

Original edit instruction: {edit_prompt}

Create an enhanced version that will produce better editing results.
Output ONLY the enhanced prompt, nothing else."""

        try:
            llm = create_vision_llm(temperature=0.7, max_retries=3)
            response = await llm.ainvoke(
                [
                    SystemMessage(content=system_prompt),
                    HumanMessage(content=user_message),
                ]
            )
            content = response.content
            if isinstance(content, list):
                content = " ".join(
                    block.get("text", "") if isinstance(block, dict) else str(block)
                    for block in content
                )
            return content.strip().strip("\"'")
        except Exception as e:
            logger.warning("Failed to enhance edit prompt, using original: %s", e)
            return edit_prompt

    async def create_edit_task(
        self, request: StandaloneEditRequest, user_id: str | None = None
    ) -> str:
        """Create a standalone edit task (for CLI submit edit)."""
        task_id = str(uuid.uuid4())

        # Build a synthetic GenerationRequest for storage compatibility
        gen_request = GenerationRequest(
            prompt=request.prompt,
            template_images=[],
            max_rounds=1,
            images_per_round=request.count,
            quality_threshold=1.0,
            enable_ai_scoring=False,
            aspect_ratio=request.aspect_ratio,
            size=request.size,
            enhance_prompt=request.enhance_prompt,
            provider=request.provider,
        )

        self._tasks[task_id] = {
            "id": task_id,
            "user_id": user_id,
            "status": GenerationStatus.PENDING,
            "request": gen_request,
            "edit_request": request,
            "state": None,
            "created_at": datetime.now(timezone.utc),
            "updated_at": datetime.now(timezone.utc),
            "result": None,
            "error": None,
        }

        try:
            await self._repo.create_task(
                task_id=task_id,
                original_prompt=request.prompt,
                template_images=[],
                config={
                    "max_rounds": 1,
                    "images_per_round": request.count,
                    "aspect_ratio": request.aspect_ratio,
                    "size": request.size,
                    "provider": request.provider.value,
                    "mode": "edit",
                },
                user_id=user_id,
            )
        except Exception:
            logger.warning("Failed to persist edit task %s to DB", task_id, exc_info=True)

        return task_id

    async def run_edit_task(self, task_id: str) -> None:
        """Run a standalone edit task in background."""
        if task_id not in self._tasks:
            raise TaskNotFoundError(task_id)

        task_info = self._tasks[task_id]
        edit_request: StandaloneEditRequest = task_info["edit_request"]

        try:
            self._update_task_status(task_id, GenerationStatus.GENERATING)

            from ..agents.tools._image_utils import normalize_reference_images
            from ..providers import GenerationMode, get_provider
            from ..providers.base import GenerationParams

            # Normalize source images
            source_images = await normalize_reference_images(
                edit_request.source_images,
                max_long_edge=2048,
                keep_http_urls=False,
            )

            # Validate provider supports IMAGE_TO_IMAGE
            provider_name = (
                edit_request.provider.value if edit_request.provider else "gemini"
            )
            provider = get_provider(provider_name)
            if GenerationMode.IMAGE_TO_IMAGE not in provider.supported_modes:
                raise ProviderError(
                    f"Provider '{provider_name}' does not support IMAGE_TO_IMAGE mode",
                    provider=provider_name,
                )

            # Optionally enhance prompt
            active_prompt = edit_request.prompt
            if edit_request.enhance_prompt:
                self._update_task_status(task_id, GenerationStatus.ENHANCING)
                active_prompt = await self._enhance_edit_prompt(edit_request.prompt)

            # Build params
            params = GenerationParams(
                prompt=active_prompt,
                reference_images=source_images,
                aspect_ratio=edit_request.aspect_ratio,
                size=edit_request.size,
            )

            # Generate
            self._update_task_status(task_id, GenerationStatus.GENERATING)
            if edit_request.count == 1:
                result = await provider.generate(params, index=0)
                results = [result]
            else:
                results = await provider.generate_batch_stream(
                    params, count=edit_request.count
                )

            # Build result images
            generated_images = []
            best_images = []
            for i, r in enumerate(results):
                img_id = str(uuid.uuid4())
                score = round(0.95 - i * 0.05, 2)
                img_dict = {
                    "id": img_id,
                    "url": r.url,
                    "local_path": r.local_path,
                    "score": score,
                    "review": "",
                    "round_number": 1,
                    "provider": provider_name,
                    "metadata": r.metadata or {},
                }
                generated_images.append(img_dict)
                best_images.append(img_dict)

            # Store in task state
            state = {
                "messages": [],
                "user_prompt": edit_request.prompt,
                "enhanced_prompt": active_prompt if edit_request.enhance_prompt else None,
                "template_images": [],
                "template_analysis": None,
                "generated_images": generated_images,
                "best_images": best_images,
                "current_round": 1,
                "status": "completed",
                "error": None,
                "max_rounds": 1,
                "images_per_round": edit_request.count,
                "aspect_ratio": edit_request.aspect_ratio,
                "image_size": edit_request.size,
                "provider": provider_name,
                "quality_threshold": 1.0,
            }
            self._tasks[task_id]["state"] = state
            self._update_task_status(task_id, GenerationStatus.COMPLETED)
            self._tasks[task_id]["result"] = self._build_result(task_id, state)

            try:
                await self._wait_for_status_persist(task_id)
                await self._repo.save_result(self._tasks[task_id]["result"])
            except Exception:
                logger.warning(
                    "Failed to save edit result for task %s to DB", task_id, exc_info=True
                )

        except Exception as e:
            logger.error("Edit task %s failed: %s", task_id, e, exc_info=True)
            self._tasks[task_id]["error"] = str(e)
            self._update_task_status(task_id, GenerationStatus.FAILED)
            try:
                await self._repo.update_status(
                    task_id, GenerationStatus.FAILED.value, str(e)
                )
            except Exception:
                logger.warning(
                    "Failed to update status for edit task %s in DB",
                    task_id,
                    exc_info=True,
                )

    async def cancel_task(self, task_id: str, user_id: str | None = None) -> bool:
        """Cancel a running task."""
        await self._ensure_task_loaded(task_id, user_id)

        if task_id in self._running_tasks:
            task_info = self._tasks[task_id]
            if user_id and task_info.get("user_id") != user_id:
                raise TaskNotFoundError(task_id)
            self._running_tasks[task_id].cancel()
            del self._running_tasks[task_id]
            self._update_task_status(task_id, GenerationStatus.CANCELLED)
            await self._wait_for_status_persist(task_id)
            return True

        return False

    async def delete_task(self, task_id: str, user_id: str | None = None) -> bool:
        # Ownership check: if user_id is set, verify it matches
        task_info = self._tasks.get(task_id)
        if user_id and task_info and task_info.get("user_id") != user_id:
            raise TaskNotFoundError(task_id)
        self._tasks.pop(task_id, None)
        self._pending_status_updates.pop(task_id, None)
        flush = self._status_flush_tasks.pop(task_id, None)
        if flush and not flush.done():
            flush.cancel()

        running = self._running_tasks.pop(task_id, None)
        if running and not running.done():
            running.cancel()

        # For API key auth (user_id=None), look up the owner from DB
        effective_uid = user_id
        if effective_uid is None:
            effective_uid = await self._repo.get_task_owner(task_id)
        local_paths = await self._repo.delete_task(task_id, effective_uid or "")

        from pathlib import Path

        for path_str in local_paths:
            try:
                p = Path(path_str)
                if p.exists():
                    p.unlink()
                    logger.info(f"Deleted image file: {path_str}")
            except OSError as e:
                logger.warning(f"Failed to delete image file {path_str}: {e}")

        return True

    async def list_tasks(self, limit: int = 20, user_id: str | None = None) -> list[TaskProgress]:
        """List recent in-memory tasks."""
        tasks = sorted(
            [
                task
                for task in self._tasks.values()
                if not user_id or task.get("user_id") == user_id
            ],
            key=lambda x: x["created_at"],
            reverse=True,
        )[:limit]

        return [await self.get_task_status(t["id"], user_id=user_id) for t in tasks]

    async def list_history(
        self, limit: int = 50, offset: int = 0, user_id: str | None = None
    ) -> list[dict]:
        """List task history from DB for sidebar.

        Only overlay *running* in-memory tasks onto the DB results so that the
        scan is bounded by the number of active tasks rather than all tasks ever
        cached.  Completed/failed/cancelled tasks are authoritative in DB.
        """
        db_tasks = await self._repo.list_tasks(limit=limit, offset=offset, user_id=user_id)
        tasks_by_id = {t["task_id"]: t for t in db_tasks}

        _running = {
            GenerationStatus.PENDING,
            GenerationStatus.ENHANCING,
            GenerationStatus.GENERATING,
            GenerationStatus.REVIEWING,
        }

        for task_id in list(self._running_tasks.keys()):
            task_data = self._tasks.get(task_id)
            if not task_data:
                continue
            if user_id and task_data.get("user_id") != user_id:
                continue
            status = task_data.get("status")
            if status not in _running:
                continue
            status_value = status.value if hasattr(status, "value") else status

            state = task_data.get("state") or {}
            best_images = state.get("best_images", []) or []
            generated_images = state.get("generated_images", []) or []
            best_score = max((img.get("score", 0.0) for img in best_images), default=None)
            total_rounds = state.get("current_round")
            total_images = len(generated_images) if generated_images else None

            raw_thumb = None
            if best_images:
                raw_thumb = best_images[0].get("url")
            elif generated_images:
                raw_thumb = generated_images[-1].get("url")
            thumbnail_url = get_thumbnail_url(raw_thumb) if raw_thumb else None
            thumbnail_original_url = raw_thumb

            if task_id in tasks_by_id:
                entry = tasks_by_id[task_id]
                if status_value:
                    entry["status"] = status_value
                if task_data.get("error"):
                    entry["error"] = task_data.get("error")
                request: GenerationRequest | None = task_data.get("request")
                if request:
                    entry["images_per_round"] = request.images_per_round
                    entry["aspect_ratio"] = request.aspect_ratio
                    entry["size"] = request.size
                    entry["template_images"] = []
                if best_score is not None:
                    entry["best_score"] = best_score
                if total_rounds is not None:
                    entry["total_rounds"] = total_rounds
                if total_images is not None:
                    entry["total_images"] = total_images
                if thumbnail_url:
                    entry["thumbnail_url"] = thumbnail_url
                if thumbnail_original_url:
                    entry["thumbnail_original_url"] = thumbnail_original_url
            else:
                request: GenerationRequest | None = task_data.get("request")
                tasks_by_id[task_id] = {
                    "task_id": task_id,
                    "status": status_value or "pending",
                    "original_prompt": request.prompt if request else "",
                    "enhanced_prompt": state.get("enhanced_prompt"),
                    "images_per_round": request.images_per_round if request else None,
                    "aspect_ratio": request.aspect_ratio if request else None,
                    "size": request.size if request else None,
                    "template_images": [],
                    "best_score": best_score or 0.0,
                    "total_rounds": total_rounds or 0,
                    "total_images": total_images or 0,
                    "thumbnail_url": thumbnail_url,
                    "thumbnail_original_url": thumbnail_original_url,
                    "created_at": task_data.get("created_at").isoformat()
                    if task_data.get("created_at")
                    else None,
                    "completed_at": task_data.get("updated_at").isoformat()
                    if task_data.get("updated_at")
                    else None,
                    "task_type": "image",
                }

        merged = list(tasks_by_id.values())
        merged.sort(key=lambda x: x.get("created_at") or "", reverse=True)
        return merged[:limit]


# Global service instance
_service: GenerationService | None = None


def get_generation_service() -> GenerationService:
    """Get the global generation service instance."""
    global _service
    if _service is None:
        _service = GenerationService()
    return _service
