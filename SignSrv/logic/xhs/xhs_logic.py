# -*- coding: utf-8 -*-
# Copyright (c) 2025 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-SignSrv project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-SignSrv/blob/main/logic/xhs/xhs_logic.py
# GitHub: https://github.com/NanmiCoder
# Licensed under NON-COMMERCIAL LEARNING LICENSE 1.1
#

# ⚠️声明：本代码仅供个人学习和研究目的使用。使用者应遵守以下原则：
# ⚠️1. 不得用于任何商业用途。
# ⚠️2. 使用时应遵守目标平台的使用条款和robots.txt规则。
# ⚠️3. 不得进行大规模爬取或对平台造成运营干扰。
# ⚠️4. 应合理控制请求频率，避免给目标平台带来不必要的负担。
# ⚠️5. 不得用于任何非法或不当的用途。
#
# 详细许可条款请参阅项目根目录下的LICENSE文件。
# 使用本代码即表示您同意遵守上述原则和LICENSE中的所有条款。


from abc import ABC, abstractmethod
from typing import Dict

from playwright.async_api import Page
from tenacity import RetryError, retry, stop_after_attempt, wait_fixed

from constant import XHS_JAVASCRIPT_SIGN, XHS_PLAYWRIGHT_SIGN
from context_vars import xhs_manager_ctx_var
from params.xiaohongshu_sign import XhsSignRequest, XhsSignResponse
from pkg import utils
from pkg.playwright.xhs_manager import XhsPlaywrightManager

from .help import sign as xhs_local_js_sign


class AbstractXhsSign(ABC):
    @abstractmethod
    async def sign(self, req_data: XhsSignRequest, force_init: bool = False) -> XhsSignResponse:
        raise NotImplementedError


class XhsPlaywrightSign(AbstractXhsSign):
    """
    xhs请求签名 - Playwright浏览器方式
    通过真实浏览器环境调用 window._webmsxyw 生成签名
    """

    def __init__(self):
        pass

    @staticmethod
    def _get_a1_params(cookies_str: str) -> str:
        """
        从cookie字符串中提取a1参数
        :param cookies_str: cookie字符串
        :return: a1参数值
        """
        return utils.convert_str_cookie_to_dict(cookies_str).get("a1", "")

    @retry(stop=stop_after_attempt(3), wait=wait_fixed(0.5))
    async def sign(self, req: XhsSignRequest, force_init: bool = False) -> XhsSignResponse:
        """
        xhs请求签名, 如果发生异常默认重试3次，每次间隔500ms
        :param req: 签名请求参数
        :param force_init: 是否强制初始化浏览器上下文
        :return: 签名结果
        """
        xhs_browser_manager: XhsPlaywrightManager = xhs_manager_ctx_var.get()
        if force_init:
            await xhs_browser_manager.reload_page()
        page_obj: Page = xhs_browser_manager.context_page

        encrypt_params = await page_obj.evaluate(
            "([url, data]) => window._webmsxyw(url,data)", [req.uri, req.data]
        )
        sign_result: Dict = xhs_local_js_sign(
            a1=self._get_a1_params(req.cookies),
            b1="I38rHdgsjopgIvesdVwgIC+oIELmBZ5e3VwXLgFTIxS3bqwErFeexd0ekncAzMFYnqthIhJeDfMDKutRI3KsYorWHPtGrbV0P9WfIi/eWc6eYqtyQApPI37ekmR6QL+5Ii6sdneeSfqYHqwl2qt5B0DBIx++GDi/sVtkIxdsxuwr4qtiIhuaIE3e3LV0I3VTIC7e0utl2ADmsLveDSKsSPw5IEvsiVtJOqw8BuwfPpdeTFWOIx4TIiu6ZPwrPut5IvlaLbgs3qtxIxes1VwHIkumIkIyejgsY/WTge7eSqte/D7sDcpipedeYrDtIC6eDVw2IENsSqtlnlSuNjVtIvoekqt3cZ7sVo4gIESyIhEGQoPUIxmPOzmoIicXePwFIviC2BvsDz7sxVtdIv6ed77eYjRGIEde6WbDtPwjIhJs3oes6DveTPtNcU6eDuw5IvYgzA7efuwXg9JsDqwoI3McIxmlsqtsaPwyssHbZAve3p/eDPwxICeskD0eSeHiIi7sjbos3grFIide6uw3IvlynVtugPw4IhJsdVwQIv5e6uw5Ih3sjuw5NqwGoVwuICzTIvRtQeVAGl/siqtFIhPtIieeYuwoeWccpUOsDskuIhRytPwwzqwAIkesWqtuqIAsVF6s1IbLIE0s6edsiPtccPwrICJefVwfIkgs60WoICKedo/eWVt3I37eVqw38BYrIhH+IC6sxIoe69vedrOsYPwVIvm6ICDRzPwcIxcSIiOeWY8WKo0ekPw8IiDynVwImB5ejWOsYVtmePtRI3rdIEkLrqtLeqwSIkgsYaAe6uwQ8qtkIvmHsVtvIkF8wzhWIkdsT7ds1pJsVPtqoVt8IEh+ICAskI==",
            x_s=encrypt_params.get("X-s", ""),
            x_t=str(encrypt_params.get("X-t", ""))
        )
        return XhsSignResponse(
            x_s=str(sign_result.get("x-s")),
            x_t=sign_result.get("x-t"),
            x_s_common=sign_result.get("x-s-common"),
            x_b3_traceid=sign_result.get("x-b3-traceid")
        )


def _patch_xhshow_a3_hash():
    """
    修复 xhshow 库 build_payload_array 中 a3_hash 计算的 bug。
    xhshow 原实现对所有请求使用 MD5(extract_api_path(content_string)) 计算 a3_hash,
    其中 extract_api_path 会同时去掉 "?" 后的查询参数和 "{" 后的 JSON body。
    但浏览器的实际行为是:
      - POST: a3 使用 MD5(api_path), 即去掉 JSON body 后的路径 → 原实现正确
      - GET:  a3 使用 MD5(完整 URL + 查询参数) → 原实现错误, 因为也去掉了查询参数
    修复方式: 对 GET 请求(content_string 不含 "{"), 使用完整 content_string 的 MD5;
              对 POST 请求(content_string 含 "{"), 保持原始行为。
    相关 issue: https://github.com/Cloxl/xhshow/issues/104
    """
    import hashlib
    from xhshow.core.crypto import CryptoProcessor

    _original_build = CryptoProcessor.build_payload_array

    def _patched_build(self, hex_parameter, a1_value, app_identifier="xhs-pc-web",
                       string_param="", timestamp=None, sign_state=None):
        payload = _original_build(self, hex_parameter, a1_value, app_identifier,
                                  string_param, timestamp, sign_state)

        # 仅当 content_string 不含 "{" 时修复 (即 GET 请求)
        # POST 请求的 content_string 包含 JSON body (以 "{" 开始), 原实现已正确
        if "{" not in string_param:
            # GET 请求: a3 应使用完整 content_string 的 MD5, 而非 api_path 的 MD5
            correct_md5_hex = hashlib.md5(string_param.encode("utf-8")).hexdigest()
            correct_md5_bytes = [int(correct_md5_hex[i:i + 2], 16) for i in range(0, 32, 2)]

            seed_byte = payload[4]
            ts_bytes = payload[8:16]

            correct_a3_hash = self._custom_hash_v2(list(ts_bytes) + correct_md5_bytes)
            for i in range(16):
                payload[128 + i] = correct_a3_hash[i] ^ seed_byte

        return payload

    CryptoProcessor.build_payload_array = _patched_build


# 启动时应用 monkey-patch
_patch_xhshow_a3_hash()


class XhsJavascriptSign(AbstractXhsSign):
    """
    xhs请求签名 - 纯算法方式 (基于xhshow库)
    使用纯Python算法还原小红书签名, 无需JS运行时或浏览器依赖
    生成 XYS_ 格式签名, 内含 mns0301_ 版本签名算法

    致谢：
        本签名实现依赖 xhshow 开源库, 由 Cloxl 提供。
        仓库地址: https://github.com/Cloxl/xhshow
        许可协议: MIT License
        使用时请遵守 xhshow 项目的 MIT 许可条款,
        同时遵守本项目的 NON-COMMERCIAL LEARNING LICENSE 1.1 许可协议。
        本代码仅供学习研究使用, 严禁用于商业用途。
    """

    def __init__(self):
        from xhshow import Xhshow
        self.xhshow_client = Xhshow()

    @retry(stop=stop_after_attempt(3), wait=wait_fixed(0.5))
    async def sign(self, req: XhsSignRequest, force_init: bool = False) -> XhsSignResponse:
        """
        xhs请求签名, 纯算版本
        :param req: 签名请求参数
        :param force_init: 未使用, 保留兼容接口
        :return: 签名结果
        """
        import hashlib
        import json
        import time

        is_post = isinstance(req.data, dict) and bool(req.data)
        if is_post:
            headers = self.xhshow_client.sign_headers_post(
                uri=req.uri,
                cookies=req.cookies,
                payload=req.data,
            )
        else:
            # GET 请求: 直接使用原始 URI 作为 content_string, 避免 parse_qs + quote 改变编码
            # (xhshow 内部的 _build_content_string 会用 urllib.parse.quote 重编码参数,
            #  可能将 '=' 编码为 '%3D' 等, 导致与 crawler 实际请求 URL 不一致)
            content_string = req.uri  # 保持原始编码

            cookie_dict = self.xhshow_client._parse_cookies(req.cookies)
            a1_value = cookie_dict.get("a1", "")

            ts = time.time()
            d_value = hashlib.md5(content_string.encode("utf-8")).hexdigest()

            # 调用底层签名方法, 传入原始 content_string 确保签名匹配
            payload_array = self.xhshow_client.crypto_processor.build_payload_array(
                d_value, a1_value, "xhs-pc-web", content_string, ts
            )
            xor_result = self.xhshow_client.crypto_processor.bit_ops.xor_transform_array(payload_array)
            config = self.xhshow_client.config
            x3_b64 = self.xhshow_client.crypto_processor.b64encoder.encode_x3(
                xor_result[:config.PAYLOAD_LENGTH]
            )
            sig_data = config.SIGNATURE_DATA_TEMPLATE.copy()
            sig_data["x3"] = config.X3_PREFIX + x3_b64
            x_s = config.XYS_PREFIX + self.xhshow_client.crypto_processor.b64encoder.encode(
                json.dumps(sig_data, separators=(",", ":"), ensure_ascii=False)
            )
            headers = {
                "x-s": x_s,
                "x-s-common": self.xhshow_client.sign_xs_common(cookie_dict),
                "x-t": str(self.xhshow_client.get_x_t(ts)),
                "x-b3-traceid": self.xhshow_client.get_b3_trace_id(),
            }

        return XhsSignResponse(
            x_s=headers.get("x-s", ""),
            x_t=headers.get("x-t", ""),
            x_s_common=headers.get("x-s-common", ""),
            x_b3_traceid=headers.get("x-b3-traceid", ""),
        )


class XhsSignFactory:
    @staticmethod
    def get_sign(sign_type: str) -> AbstractXhsSign:
        if sign_type == XHS_PLAYWRIGHT_SIGN:
            return XhsPlaywrightSign()
        elif sign_type == XHS_JAVASCRIPT_SIGN:
            return XhsJavascriptSign()
        else:
            raise NotImplementedError


class XhsSignLogic:
    def __init__(self, sign_type: str):
        """
        初始化
        :param sign_type: 签名类型, 可选 "javascript" 或 "playwright"
        """
        self.sign_server = XhsSignFactory.get_sign(sign_type)

    async def sign(self, req_data: XhsSignRequest) -> XhsSignResponse:
        """
        签名入口, 失败后会强制重新初始化再重试
        :param req_data: 签名请求参数
        :return: 签名结果
        """
        try:
            return await self.sign_server.sign(req_data)
        except RetryError:
            return await self.sign_server.sign(req_data, force_init=True)
