"""Video generation service for managing video generation tasks."""

import asyncio
import logging
import uuid
from copy import deepcopy
from datetime import datetime, timezone

from ..core.exceptions import TaskNotFoundError
from ..i18n import get_text
from ..models.video_schemas import (
    GeneratedVideo,
    VideoGenerationRequest,
    VideoGenerationResult,
    VideoRoundResult,
    VideoTaskProgress,
)

logger = logging.getLogger(__name__)


class VideoGenerationService:
    """Service for managing video generation tasks."""

    def __init__(self):
        self._tasks: dict[str, dict] = {}
        self._running_tasks: dict[str, asyncio.Task] = {}
        self._pending_status_updates: dict[str, tuple[str, str | None]] = {}
        self._status_flush_tasks: dict[str, asyncio.Task] = {}
        self._repo = self._create_repo()

    @staticmethod
    def _create_repo():
        from ..db.video_repository import VideoTaskRepository

        return VideoTaskRepository()

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
            raise TaskNotFoundError(task_id) from exc
        if not persisted:
            raise TaskNotFoundError(task_id)

        config = persisted.config or {}
        provider_value = config.get("provider", "kling")

        from ..models.video_schemas import VideoProvider

        try:
            provider_enum = VideoProvider(provider_value)
        except Exception:
            provider_enum = VideoProvider.KLING

        request = VideoGenerationRequest(
            prompt=persisted.original_prompt or "",
            start_image=persisted.start_image,
            end_image=persisted.end_image,
            max_rounds=max(int(config.get("max_rounds") or 1), 1),
            videos_per_round=int(config.get("videos_per_round") or 2),
            duration=int(config.get("duration") or 5),
            resolution=config.get("resolution"),
            aspect_ratio=config.get("aspect_ratio") or "16:9",
            generate_audio=bool(config.get("generate_audio", False)),
            provider=provider_enum,
        )

        # Rebuild generated_videos list from persisted rounds
        generated_videos = []
        for round_result in persisted.rounds:
            for vid in round_result.videos:
                generated_videos.append(
                    {
                        "id": vid.id,
                        "url": vid.url,
                        "local_path": vid.local_path,
                        "video_type": vid.video_type,
                        "duration": vid.duration,
                        "round_number": vid.round_number,
                        "provider": vid.provider,
                        "provider_task_id": vid.provider_task_id,
                        "metadata": vid.metadata or {},
                    }
                )

        self._tasks[task_id] = {
            "id": task_id,
            "user_id": user_id,
            "status": persisted.status,
            "request": request,
            "state": {
                "generated_videos": generated_videos,
                "enhanced_prompt": persisted.enhanced_prompt,
                "current_round": persisted.total_rounds or 0,
            },
            "created_at": persisted.started_at or datetime.now(timezone.utc),
            "updated_at": persisted.completed_at or datetime.now(timezone.utc),
            "result": persisted,
            "error": persisted.error,
        }
        return self._tasks[task_id]

    async def create_task(
        self, request: VideoGenerationRequest, user_id: str | None = None
    ) -> str:
        """Create a new video generation task."""
        task_id = str(uuid.uuid4())

        self._tasks[task_id] = {
            "id": task_id,
            "user_id": user_id,
            "status": "pending",
            "request": request,
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
                start_image=request.start_image,
                end_image=request.end_image,
                config={
                    "max_rounds": request.max_rounds,
                    "videos_per_round": request.videos_per_round,
                    "duration": request.duration,
                    "resolution": request.resolution,
                    "aspect_ratio": request.aspect_ratio,
                    "generate_audio": request.generate_audio,
                    "provider": (request.provider.value if request.provider else "kling"),
                },
                user_id=user_id,
            )
        except Exception:
            logger.warning("Failed to persist video task %s to DB", task_id, exc_info=True)
            self._tasks.pop(task_id, None)
            raise

        return task_id

    async def get_task_owner(self, task_id: str) -> str | None:
        """Get the owner user_id for a task."""
        if task_id in self._tasks:
            return self._tasks[task_id].get("user_id")
        return await self._repo.get_task_owner(task_id)

    async def start_task(self, task_id: str) -> None:
        """Start executing a video generation task."""
        if task_id not in self._tasks:
            raise TaskNotFoundError(task_id)

        task_info = self._tasks[task_id]
        request: VideoGenerationRequest = task_info["request"]

        async_task = asyncio.create_task(self._run_task(task_id, request))
        self._running_tasks[task_id] = async_task

    async def _run_task(self, task_id: str, request: VideoGenerationRequest) -> None:
        """Run the video generation workflow for a task."""
        try:
            self._update_task_status(task_id, "enhancing")

            from ..agents.video_state import create_initial_video_state
            from ..agents.video_workflow import create_video_generation_workflow
            from ..api.websocket import get_connection_manager

            manager = get_connection_manager()
            workflow = create_video_generation_workflow()

            config = {"configurable": {"thread_id": task_id}}

            # Build initial state from request
            initial_state = create_initial_video_state(
                user_prompt=request.prompt,
                start_image=request.start_image,
                end_image=request.end_image,
                reference_images=request.reference_images or [],
                max_rounds=request.max_rounds,
                videos_per_round=request.videos_per_round,
                duration=request.duration,
                resolution=request.resolution,
                aspect_ratio=request.aspect_ratio,
                generate_audio=request.generate_audio,
                provider=request.provider.value if request.provider else "kling",
                enhance_prompt_enabled=request.enhance_prompt,
                negative_prompt=request.negative_prompt,
                cfg_scale=request.cfg_scale,
                professional=request.professional,
                seed=request.seed,
                sample_count=request.sample_count,
            )
            self._tasks[task_id]["state"] = deepcopy(initial_state)

            logger.info(
                "Starting video workflow for task %s: max_rounds=%d, "
                "videos_per_round=%d, duration=%ds",
                task_id,
                request.max_rounds,
                request.videos_per_round,
                request.duration,
            )

            # Stream custom events from the workflow
            prev_round = 0
            async for event in workflow.astream(
                initial_state,
                config=config,
                stream_mode="custom",
            ):
                if isinstance(event, dict) and event.get("type") == "status_change":
                    self._apply_status_change(task_id, event)
                elif isinstance(event, dict) and event.get("type") == "video_ready":
                    self._update_task_status(task_id, "generating")
                    video_info = event.get("video", {})
                    self._apply_video_ready(task_id, video_info)
                    video_data = {
                        "type": "video_ready",
                        "video": {
                            "id": video_info.get("id", ""),
                            "url": video_info.get("url", ""),
                            "local_path": video_info.get("local_path"),
                            "video_type": video_info.get("video_type", "mp4"),
                            "duration": video_info.get("duration"),
                            "round_number": video_info.get("round_number", 1),
                            "provider": video_info.get("provider", ""),
                            "provider_task_id": video_info.get("provider_task_id"),
                            "metadata": video_info.get("metadata", {}),
                        },
                    }
                    await manager.send_progress(task_id, video_data)
                elif isinstance(event, dict) and event.get("type") == "video_progress":
                    # Video generation progress update (polling status)
                    progress_data = {
                        "type": "video_progress",
                        "progress_percent": event.get("progress_percent", 0),
                        "eta": event.get("eta", 0),
                        "status": event.get("status", "processing"),
                    }
                    await manager.send_progress(task_id, progress_data)
                    # Store latest progress in state for status queries
                    state = self._tasks[task_id].get("state")
                    if isinstance(state, dict):
                        state["_video_progress"] = event.get("progress_percent", 0)
                        state["_video_eta"] = event.get("eta", 0)

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
                            "Failed to persist intermediate video round for task %s",
                            task_id,
                            exc_info=True,
                        )

            # Get final state
            final_snapshot = await workflow.aget_state(config)
            final_state = final_snapshot.values if final_snapshot else {}

            self._tasks[task_id]["state"] = final_state

            # Determine final status BEFORE building result so _build_result
            # captures the correct status for DB persistence.
            final_result = final_state.get("final_result", {})
            if isinstance(final_result, dict) and final_result.get("success") is False:
                self._update_task_status(task_id, "failed")
            elif final_state.get("status") == "completed" or final_state.get(
                "generated_videos"
            ):
                self._update_task_status(task_id, "completed")
            else:
                self._update_task_status(task_id, "failed")

            self._tasks[task_id]["result"] = self._build_result(task_id, final_state)

            logger.info(
                "Video workflow completed for task %s: status=%s, rounds=%d, videos=%d",
                task_id,
                final_state.get("status"),
                final_state.get("current_round", 0),
                len(final_state.get("generated_videos", [])),
            )

            # Persist result to DB
            try:
                await self._wait_for_status_persist(task_id)
                await self._repo.save_result(self._tasks[task_id]["result"])
            except Exception:
                logger.warning(
                    "Failed to save video result for task %s to DB", task_id, exc_info=True
                )

        except asyncio.CancelledError:
            logger.info("Video workflow cancelled for task %s", task_id)
            self._tasks[task_id]["error"] = "Task cancelled"
            self._update_task_status(task_id, "cancelled")
            try:
                await self._wait_for_status_persist(task_id)
            except Exception:
                pass

        except Exception as e:
            logger.error(
                "Video workflow failed for task %s: %s",
                task_id,
                e,
                exc_info=True,
            )
            self._tasks[task_id]["error"] = str(e)
            self._update_task_status(task_id, "failed")
            try:
                await self._repo.update_status(task_id, "failed", str(e))
            except Exception:
                logger.warning(
                    "Failed to update video status for task %s in DB", task_id, exc_info=True
                )
            try:
                from ..api.websocket import get_connection_manager

                await get_connection_manager().send_progress(
                    task_id, {"error": str(e), "task_id": task_id}
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
            status_val = str(info.get("status", ""))
            if status_val in ("completed", "failed", "cancelled"):
                age = (now - info.get("updated_at", now)).total_seconds()
                if age > 3600:
                    evictable.append((age, tid))
        evictable.sort(reverse=True)
        for _, tid in evictable[: len(self._tasks) - max_size]:
            self._tasks.pop(tid, None)

    def _update_task_status(self, task_id: str, status: str) -> None:
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

        status = self._tasks[task_id].get("status", "pending")
        error = self._tasks[task_id].get("error")
        self._pending_status_updates[task_id] = (status, error)

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
                        "Failed to persist video status for task %s", task_id, exc_info=True
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
                    logger.warning(
                        "Timed out waiting for video status persist: %s", task_id
                    )
                    return
                await asyncio.sleep(0.01)
                continue
            remaining = deadline - asyncio.get_event_loop().time()
            if remaining <= 0:
                logger.warning(
                    "Timed out waiting for video status persist: %s", task_id
                )
                return
            try:
                await asyncio.wait_for(asyncio.shield(flush_task), timeout=remaining)
            except TimeoutError:
                logger.warning(
                    "Timed out waiting for video status flush: %s", task_id
                )
                return

    def _apply_status_change(self, task_id: str, event: dict) -> None:
        """Apply streamed status updates from workflow nodes."""
        if task_id not in self._tasks:
            return

        status_key = str(event.get("status", "")).lower()
        valid_statuses = {
            "pending",
            "enhancing",
            "generating",
            "polling",
            "completed",
            "failed",
            "cancelled",
        }
        if status_key in valid_statuses:
            self._update_task_status(task_id, status_key)

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

    def _apply_video_ready(self, task_id: str, video_info: dict) -> None:
        """Merge streamed video-ready events into in-memory task state."""
        if task_id not in self._tasks:
            return

        state = self._tasks[task_id].get("state")
        if not isinstance(state, dict):
            state = {"generated_videos": [], "current_round": 0}
            self._tasks[task_id]["state"] = state

        generated_videos = state.setdefault("generated_videos", [])
        generated_videos.append(
            {
                "id": video_info.get("id", str(uuid.uuid4())),
                "url": video_info.get("url", ""),
                "local_path": video_info.get("local_path"),
                "video_type": video_info.get("video_type", "mp4"),
                "duration": video_info.get("duration"),
                "round_number": video_info.get("round_number", 1),
                "provider": video_info.get("provider", "kling"),
                "provider_task_id": video_info.get("provider_task_id"),
                "metadata": video_info.get("metadata", {}),
            }
        )

        try:
            round_number = int(video_info.get("round_number", 1))
        except (TypeError, ValueError):
            round_number = 1
        state["current_round"] = max(state.get("current_round", 0), round_number)

    def _build_result(self, task_id: str, state: dict) -> VideoGenerationResult:
        """Build video generation result from final state."""
        task_info = self._tasks[task_id]
        request: VideoGenerationRequest = task_info["request"]

        generated_videos = state.get("generated_videos", [])

        # Group videos by round
        rounds_dict: dict[int, list[GeneratedVideo]] = {}
        all_videos: list[GeneratedVideo] = []
        for vid_data in generated_videos:
            round_num = vid_data.get("round_number", 1)
            if round_num not in rounds_dict:
                rounds_dict[round_num] = []

            video = GeneratedVideo(
                id=vid_data.get("id", str(uuid.uuid4())),
                url=vid_data.get("url", ""),
                local_path=vid_data.get("local_path"),
                video_type=vid_data.get("video_type", "mp4"),
                duration=vid_data.get("duration"),
                round_number=round_num,
                provider=vid_data.get("provider", "kling"),
                provider_task_id=vid_data.get("provider_task_id"),
                metadata=vid_data.get("metadata", {}),
            )
            rounds_dict[round_num].append(video)
            all_videos.append(video)

        # Build round results
        rounds = []
        for round_num in sorted(rounds_dict.keys()):
            rounds.append(
                VideoRoundResult(
                    round_number=round_num,
                    videos=rounds_dict[round_num],
                )
            )

        return VideoGenerationResult(
            task_id=task_id,
            status=task_info["status"],
            original_prompt=request.prompt,
            enhanced_prompt=state.get("enhanced_prompt"),
            start_image=request.start_image,
            end_image=request.end_image,
            config={
                "max_rounds": request.max_rounds,
                "videos_per_round": request.videos_per_round,
                "duration": request.duration,
                "resolution": request.resolution,
                "aspect_ratio": request.aspect_ratio,
                "generate_audio": request.generate_audio,
                "provider": (request.provider.value if request.provider else "kling"),
            },
            rounds=rounds,
            total_rounds=len(rounds),
            total_videos=len(generated_videos),
            videos=all_videos,
            started_at=task_info["created_at"],
            completed_at=task_info["updated_at"],
            error=task_info.get("error"),
        )

    async def get_task_status(
        self,
        task_id: str,
        lang: str = "en",
        user_id: str | None = None,
    ) -> VideoTaskProgress:
        """Get current status of a video task."""
        task_info = await self._ensure_task_loaded(task_id, user_id)
        request: VideoGenerationRequest = task_info["request"]
        state = task_info.get("state") or {}

        recent_videos = []
        for vid_data in state.get("generated_videos", [])[-5:]:
            recent_videos.append(
                GeneratedVideo(
                    id=vid_data.get("id", ""),
                    url=vid_data.get("url", ""),
                    local_path=vid_data.get("local_path"),
                    video_type=vid_data.get("video_type", "mp4"),
                    duration=vid_data.get("duration"),
                    round_number=vid_data.get("round_number", 1),
                    provider=vid_data.get("provider", "kling"),
                    provider_task_id=vid_data.get("provider_task_id"),
                    metadata=vid_data.get("metadata", {}),
                )
            )

        generated_videos = state.get("generated_videos", [])
        current_round = state.get("current_round", 0)

        return VideoTaskProgress(
            task_id=task_id,
            status=task_info["status"],
            current_round=current_round,
            max_rounds=request.max_rounds,
            videos_per_round=request.videos_per_round,
            videos_generated=len(generated_videos),
            videos_in_current_round=len(
                [v for v in generated_videos if v.get("round_number") == current_round]
            ),
            message=self._get_status_message(
                task_info["status"], state, task_info.get("error"), lang=lang
            ),
            original_prompt=request.prompt,
            enhanced_prompt=state.get("enhanced_prompt"),
            recent_videos=recent_videos,
            current_video_progress=state.get("_video_progress"),
            current_video_eta=state.get("_video_eta"),
            error=task_info.get("error"),
        )

    def _get_status_message(
        self,
        status: str,
        state: dict,
        error: str | None = None,
        lang: str = "en",
    ) -> str:
        """Generate a human-readable status message for video tasks."""
        if status == "pending":
            return get_text("video.status.pending", lang)
        elif status == "enhancing":
            return get_text("video.status.enhancing", lang)
        elif status == "generating":
            round_num = state.get("current_round", 1)
            return get_text("video.status.generating", lang, round_num=round_num)
        elif status == "polling":
            progress = state.get("_video_progress", 0)
            return get_text("video.status.polling", lang, progress=progress)
        elif status == "completed":
            total = len(state.get("generated_videos", []))
            return get_text("video.status.completed", lang, total=total)
        elif status == "failed":
            if error:
                return get_text("video.status.failed", lang, error=error)
            return get_text("video.status.failed_generic", lang)
        elif status == "cancelled":
            return get_text("video.status.cancelled", lang)
        return get_text("status.unknown", lang)

    async def get_task_result(
        self, task_id: str, user_id: str | None = None
    ) -> VideoGenerationResult:
        """Get the final result of a completed video task."""
        task_info = await self._ensure_task_loaded(task_id, user_id)
        if task_info.get("result"):
            return task_info["result"]
        return self._build_result(task_id, task_info.get("state") or {})

    async def cancel_task(self, task_id: str, user_id: str | None = None) -> bool:
        """Cancel a running video task."""
        await self._ensure_task_loaded(task_id, user_id)

        if task_id in self._running_tasks:
            self._running_tasks[task_id].cancel()
            del self._running_tasks[task_id]
            self._update_task_status(task_id, "cancelled")
            await self._wait_for_status_persist(task_id)
            return True

        return False

    async def delete_task(self, task_id: str, user_id: str | None = None) -> bool:
        """Delete a video task and clean up resources."""
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
                    logger.info("Deleted video file: %s", path_str)
            except OSError as e:
                logger.warning("Failed to delete video file %s: %s", path_str, e)

        return True

    def list_tasks(self, limit: int = 20, user_id: str | None = None) -> list[VideoTaskProgress]:
        """List recent in-memory video tasks (used for active task listing)."""
        tasks = sorted(
            [
                task
                for task in self._tasks.values()
                if not user_id or task.get("user_id") == user_id
            ],
            key=lambda x: x["created_at"],
            reverse=True,
        )[:limit]

        # Build progress objects inline to avoid async call
        result = []
        for t in tasks:
            request: VideoGenerationRequest = t["request"]
            state = t.get("state") or {}
            generated_videos = state.get("generated_videos", [])
            current_round = state.get("current_round", 0)

            recent_videos = []
            for vid_data in generated_videos[-5:]:
                recent_videos.append(
                    GeneratedVideo(
                        id=vid_data.get("id", ""),
                        url=vid_data.get("url", ""),
                        local_path=vid_data.get("local_path"),
                        video_type=vid_data.get("video_type", "mp4"),
                        duration=vid_data.get("duration"),
                        round_number=vid_data.get("round_number", 1),
                        provider=vid_data.get("provider", "kling"),
                        provider_task_id=vid_data.get("provider_task_id"),
                        metadata=vid_data.get("metadata", {}),
                    )
                )

            result.append(
                VideoTaskProgress(
                    task_id=t["id"],
                    status=t["status"],
                    current_round=current_round,
                    max_rounds=request.max_rounds,
                    videos_per_round=request.videos_per_round,
                    videos_generated=len(generated_videos),
                    videos_in_current_round=len(
                        [v for v in generated_videos if v.get("round_number") == current_round]
                    ),
                    message="",
                    original_prompt=request.prompt,
                    enhanced_prompt=state.get("enhanced_prompt"),
                    recent_videos=recent_videos,
                    current_video_progress=state.get("_video_progress"),
                    current_video_eta=state.get("_video_eta"),
                    error=t.get("error"),
                )
            )
        return result

    async def list_history(
        self, limit: int = 50, offset: int = 0, user_id: str | None = None
    ) -> list[dict]:
        """List video task history from DB for sidebar, merged with in-memory running tasks."""
        db_tasks = await self._repo.list_tasks(limit=limit, offset=offset, user_id=user_id)
        tasks_by_id = {t["task_id"]: t for t in db_tasks}

        # Only overlay running tasks from memory — completed tasks rely on DB.
        for task_id in self._running_tasks:
            task_info = self._tasks.get(task_id)
            if not task_info:
                continue
            if user_id and task_info.get("user_id") != user_id:
                continue

            state = task_info.get("state") or {}
            generated_videos = state.get("generated_videos", [])
            thumbnail_url = None
            if generated_videos:
                thumbnail_url = generated_videos[-1].get("url")

            if task_id in tasks_by_id:
                entry = tasks_by_id[task_id]
                entry["status"] = task_info.get("status", "pending")
                if task_info.get("error"):
                    entry["error"] = task_info.get("error")
                if generated_videos:
                    entry["total_videos"] = len(generated_videos)
                if thumbnail_url:
                    entry["thumbnail_url"] = thumbnail_url
                entry["total_rounds"] = state.get("current_round", 0)
            else:
                request: VideoGenerationRequest | None = task_info.get("request")
                tasks_by_id[task_id] = {
                    "task_id": task_id,
                    "status": task_info.get("status", "pending"),
                    "original_prompt": request.prompt if request else "",
                    "enhanced_prompt": state.get("enhanced_prompt"),
                    "best_score": 0.0,
                    "total_rounds": state.get("current_round", 0),
                    "total_videos": len(generated_videos),
                    "thumbnail_url": thumbnail_url,
                    "created_at": task_info.get("created_at").isoformat()
                    if task_info.get("created_at")
                    else None,
                    "completed_at": task_info.get("updated_at").isoformat()
                    if task_info.get("updated_at")
                    else None,
                    "task_type": "video",
                }

        merged = list(tasks_by_id.values())
        merged.sort(key=lambda x: x.get("created_at") or "", reverse=True)
        return merged[:limit]


_video_generation_service: VideoGenerationService | None = None


def get_video_generation_service() -> VideoGenerationService:
    """Get the singleton VideoGenerationService instance."""
    global _video_generation_service
    if _video_generation_service is None:
        _video_generation_service = VideoGenerationService()
    return _video_generation_service
