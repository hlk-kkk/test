# -*- coding: utf-8 -*-
# Copyright (c) 2025 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-CookieBridge project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-CookieBridge/blob/main/server/apis/base_handler.py
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

import json
import uuid
from typing import Any, Dict, Union

from pydantic import BaseModel, ValidationError
from tornado.web import RequestHandler

import constant
import context_vars
from constant.error_code import ApiCode
from models.base_model import ErrorResponseModel, OkResponseModel


class BaseRequestHandler(RequestHandler):
    def prepare(self):
        request_id: str = (
            self.request.headers.get(constant.REQUEST_ID_HEADERS_KEY)
            or uuid.uuid4().hex[:16]
        )
        context_vars.request_id_var.set(request_id)

    def set_default_headers(self):
        self.set_header("Access-Control-Allow-Origin", "*")
        self.set_header(
            "Access-Control-Allow-Headers",
            "Content-Type, Authorization, X-Requested-With",
        )
        self.set_header(
            "Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS"
        )

    def options(self, *args, **kwargs):
        self.set_status(204)
        self.finish()


class BaseAPIHandler(BaseRequestHandler):

    def check_xsrf_cookie(self):
        pass

    def return_error(self, errorcode, extra=None):
        if extra is None:
            extra = {}
        if isinstance(errorcode, ApiCode):
            errorcode_val = errorcode
        else:
            errorcode_val = errorcode
        message = ApiCode.get_message(errorcode_val)
        response = ErrorResponseModel(biz_code=errorcode_val.value, msg=message, extra=extra)
        self.set_header("Content-Type", "application/json")
        self.write(response.model_dump_json())
        self.finish()

    def return_error_msg(self, errorcode, msg, extra=None):
        if extra is None:
            extra = {}
        message = ApiCode.get_message(errorcode)
        full_msg = f"{message} {msg}" if msg else message
        response = ErrorResponseModel(biz_code=errorcode.value, msg=full_msg, extra=extra)
        self.set_header("Content-Type", "application/json")
        self.write(response.model_dump_json())
        self.finish()

    def return_ok(self, data=None, msg="OK!"):
        if data is None:
            data = {}
        if isinstance(data, BaseModel):
            data = data.model_dump()
        response = OkResponseModel(data=data, msg=msg)
        self.set_header("Content-Type", "application/json")
        self.write(response.model_dump_json())
        self.finish()


class TornadoBaseReqHandler(BaseAPIHandler):

    def prepare(self):
        super().prepare()
        self.request_data = {}
        if self.request.headers.get("Content-Type", "").find("application/json") != -1:
            try:
                body_dict = json.loads(self.request.body.decode("utf-8"))
                self.request_data.update(body_dict)
            except (json.JSONDecodeError, UnicodeDecodeError):
                pass
            return

        self.request_data.update(
            {k: self.get_argument(k) for k in self.request.arguments}
        )


class PongHandler(BaseAPIHandler):
    async def get(self):
        self.return_ok(data={"message": "pong"})
