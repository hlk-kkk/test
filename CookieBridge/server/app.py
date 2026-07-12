# -*- coding: utf-8 -*-
# Copyright (c) 2025 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-CookieBridge project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-CookieBridge/blob/main/server/app.py
# GitHub: https://github.com/NanmiCoder
# Licensed under NON-COMMERCIAL LEARNING LICENSE 1.1
#
# 声明：本代码仅供学习和研究目的使用。使用者应遵守以下原则：
# 1. 不得用于任何商业用途。
# 2. 使用时应遵守目标平台的使用条款和robots.txt规则。
# 3. 不得进行大规模爬取或对平台造成运营干扰。
# 4. 应合理控制请求频率，避免给目标平台带来不必要的负担。
# 5. 不得用于任何非法或不当的用途。
#
# 详细许可条款请参阅项目根目录下的LICENSE文件。
# 使用本代码即表示您同意遵守上述原则和LICENSE中的所有条款。

import asyncio
import logging
import os
import sys

import tornado.web
from tornado.options import define, options

import config
import router
from context_vars import request_id_var

logger = logging.getLogger(__name__)


def register_all_handlers(all_handlers):
    all_handlers += router.all_router


class LoggerCustomFilter(logging.Filter):
    def filter(self, record):
        record.request_id = request_id_var.get("-")
        return True


class Application(tornado.web.Application):
    all_handlers = []

    def __init__(self):
        register_all_handlers(Application.all_handlers)
        settings = dict(
            debug=config.IS_DEBUG,
            gzip=True,
            autoreload=True,
        )
        super(Application, self).__init__(Application.all_handlers, **settings)


async def init():
    logging.basicConfig(
        datefmt="%Y-%m-%d %H:%M:%S",
        level=config.LOGGER_LEVEL,
        format="%(asctime)s %(levelname)s %(filename)s:%(lineno)d req_id: %(request_id)s %(message)s",
    )
    log_filter = LoggerCustomFilter()
    for handler in logging.getLogger().handlers:
        handler.addFilter(log_filter)

    project_root = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
    sys.path.insert(0, project_root)


define(name="port", default=config.APP_PORT, type=int, help="app http listen port")
define(name="host", default=config.APP_HOST, type=str, help="app http listen host")


async def create_app():
    await init()
    app = Application()
    app.listen(port=options.port, address=options.host)
    logger.info("CookieBridge Server running at port %s", options.port)
    if config.IS_DEBUG:
        logger.info("CookieBridge Server running in debug mode, debug url: http://localhost:%s", options.port)
    await asyncio.Event().wait()


if __name__ == "__main__":
    asyncio.run(create_app())
