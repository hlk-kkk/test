"""WebSocket endpoints for real-time progress updates."""

import asyncio
import logging
from typing import Optional

logger = logging.getLogger(__name__)

from fastapi import APIRouter, WebSocket, WebSocketDisconnect

from ..core.exceptions import TaskNotFoundError
from ..services.auth_service import auth_service
from ..services.generation_service import get_generation_service

router = APIRouter()


class ConnectionManager:
    """Manages WebSocket connections."""

    def __init__(self):
        self.active_connections: dict[str, list[WebSocket]] = {}

    async def connect(self, websocket: WebSocket, task_id: str):
        """Accept a new WebSocket connection."""
        await websocket.accept()
        if task_id not in self.active_connections:
            self.active_connections[task_id] = []
        self.active_connections[task_id].append(websocket)

    def disconnect(self, websocket: WebSocket, task_id: str):
        """Remove a WebSocket connection."""
        if task_id in self.active_connections:
            if websocket in self.active_connections[task_id]:
                self.active_connections[task_id].remove(websocket)
            if not self.active_connections[task_id]:
                del self.active_connections[task_id]

    async def send_progress(self, task_id: str, data: dict):
        """Send progress update to all connections for a task."""
        if task_id in self.active_connections:
            dead: list[WebSocket] = []
            for connection in self.active_connections[task_id]:
                try:
                    await connection.send_json(data)
                except Exception:
                    dead.append(connection)
            for ws in dead:
                self.disconnect(ws, task_id)

    async def broadcast(self, message: dict):
        """Broadcast a message to all connections."""
        for connections in self.active_connections.values():
            for connection in connections:
                try:
                    await connection.send_json(message)
                except Exception:
                    pass


manager = ConnectionManager()


@router.websocket("/ws/progress/{task_id}")
async def websocket_progress(websocket: WebSocket, task_id: str):
    """
    WebSocket endpoint for real-time task progress updates.

    Connect to receive live updates for a specific task:
    - Status changes
    - New images generated
    - Score updates
    - Completion notification
    """
    # Extract lang and token from query parameters
    lang = websocket.query_params.get("lang", "en")
    token = websocket.query_params.get("token")
    if not token:
        auth_header = websocket.headers.get("authorization")
        if auth_header and auth_header.lower().startswith("bearer "):
            token = auth_header.split(" ", 1)[1].strip()

    if not token:
        await websocket.close(code=1008)
        return

    user_id = auth_service.verify_access_token(token)
    if not user_id:
        await websocket.close(code=1008)
        return

    service = get_generation_service()
    owner_id = await service.get_task_owner(task_id)
    if not owner_id or owner_id != user_id:
        await websocket.close(code=1008)
        return

    await manager.connect(websocket, task_id)

    try:
        while True:
            try:
                # Get current progress
                progress = await service.get_task_status(task_id, lang=lang)

                # Send update
                payload = progress.model_dump(mode="json")
                # Avoid repeatedly sending large template image payloads over WS
                payload["template_images"] = []
                await websocket.send_json(payload)

                # Check if task is complete
                if progress.status in ["completed", "failed", "cancelled"]:
                    break

                # Wait before next update
                await asyncio.sleep(1)

            except TaskNotFoundError:
                from ..i18n import get_text
                try:
                    await websocket.send_json({
                        "error": get_text("ws.task_not_found"),
                        "task_id": task_id,
                    })
                except Exception:
                    pass
                break
            except Exception as e:
                logger.error("WebSocket error for task %s: %s", task_id, e, exc_info=True)
                try:
                    await websocket.send_json({
                        "error": "Internal server error",
                        "task_id": task_id,
                    })
                except Exception:
                    pass
                break

    except (WebSocketDisconnect, RuntimeError):
        pass
    finally:
        manager.disconnect(websocket, task_id)


@router.websocket("/ws/video/{task_id}")
async def websocket_video_progress(websocket: WebSocket, task_id: str):
    """WebSocket endpoint for real-time video task progress updates."""
    lang = websocket.query_params.get("lang", "en")
    token = websocket.query_params.get("token")
    if not token:
        auth_header = websocket.headers.get("authorization")
        if auth_header and auth_header.lower().startswith("bearer "):
            token = auth_header.split(" ", 1)[1].strip()

    if not token:
        await websocket.close(code=1008)
        return

    user_id = auth_service.verify_access_token(token)
    if not user_id:
        await websocket.close(code=1008)
        return

    from ..services.video_generation_service import get_video_generation_service

    service = get_video_generation_service()
    owner_id = await service.get_task_owner(task_id)
    if not owner_id or owner_id != user_id:
        await websocket.close(code=1008)
        return

    await manager.connect(websocket, task_id)

    try:
        while True:
            try:
                progress = await service.get_task_status(task_id, lang=lang)
                payload = progress.model_dump(mode="json")
                await websocket.send_json(payload)

                if progress.status in ["completed", "failed", "cancelled"]:
                    break

                await asyncio.sleep(2)

            except TaskNotFoundError:
                from ..i18n import get_text

                try:
                    await websocket.send_json({
                        "error": get_text("ws.task_not_found"),
                        "task_id": task_id,
                    })
                except Exception:
                    pass
                break
            except Exception as e:
                logger.error("Video WebSocket error for task %s: %s", task_id, e, exc_info=True)
                try:
                    await websocket.send_json({
                        "error": "Internal server error",
                        "task_id": task_id,
                    })
                except Exception:
                    pass
                break

    except (WebSocketDisconnect, RuntimeError):
        pass
    finally:
        manager.disconnect(websocket, task_id)


def get_connection_manager() -> ConnectionManager:
    """Get the global connection manager."""
    return manager
