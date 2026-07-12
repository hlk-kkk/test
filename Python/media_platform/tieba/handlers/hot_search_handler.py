# -*- coding: utf-8 -*-

from typing import TYPE_CHECKING

from pkg.tools import utils

from .base_handler import BaseHandler

if TYPE_CHECKING:
    from repo.checkpoint.checkpoint_store import CheckpointRepoManager

    from ..client import BaiduTieBaClient
    from ..processors.comment_processor import CommentProcessor
    from ..processors.note_processor import NoteProcessor


class HotSearchHandler(BaseHandler):
    """Placeholder handler for Tieba hot search thread lists."""

    def __init__(
        self,
        tieba_client: "BaiduTieBaClient",
        checkpoint_manager: "CheckpointRepoManager",
        note_processor: "NoteProcessor",
        comment_processor: "CommentProcessor",
    ):
        super().__init__(
            tieba_client, checkpoint_manager, note_processor, comment_processor
        )

    async def handle(self) -> None:
        await self.get_hot_search_notes()

    async def get_hot_search_notes(self) -> None:
        utils.logger.info(
            "[HotSearchHandler.get_hot_search_notes] Tieba hot search "
            "placeholder is ready; platform-specific crawling is not implemented yet"
        )
