# -*- coding: utf-8 -*-
# Copyright (c) 2025 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-Python project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Python/blob/main/media_platform/tieba/client.py
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

import hashlib
import json
import traceback
from typing import Any, Dict, List, Optional, Union
from urllib.parse import parse_qs, quote, unquote, urlencode, urlparse

from curl_cffi.requests import AsyncSession, Response
from tenacity import RetryError, retry, stop_after_attempt, wait_fixed

import config
from base.base_crawler import AbstractApiClient
from constant.baidu_tieba import TIEBA_URL
from model.m_baidu_tieba import TiebaNote
from pkg.account_pool import AccountWithIpModel
from pkg.account_pool.pool import AccountWithIpPoolManager
from pkg.tools import utils

from .field import SearchNoteType, SearchSortType
from .help import TieBaExtractor

PC_SIGN_SECRET = "36770b1f34c9bbf2e7d1a99d2b82fa9e"


class BaiduTieBaClient(AbstractApiClient):
    def __init__(
            self,
            timeout: int = 10,
            user_agent: str = None,
            account_with_ip_pool: AccountWithIpPoolManager = None,
    ):
        """
        tieba client constructor
        Args:
            timeout:
            user_agent:
            account_with_ip_pool:
        """
        self.timeout = timeout
        self._user_agent = user_agent or utils.get_user_agent()
        self.page_extractor = TieBaExtractor()
        self.account_with_ip_pool = account_with_ip_pool
        self.account_info: Optional[AccountWithIpModel] = None
        self._pc_tbs = ""

    @property
    def headers(self):
        return {
            "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
            "Cookie": self._cookies,
            "origin": "https://tieba.baidu.com",
            "referer": "https://tieba.baidu.com",
            "user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
        }

    @property
    def _proxies(self):
        return (
            self.account_info.ip_info.format_httpx_proxy()
            if self.account_info.ip_info
            else None
        )

    @property
    def _cookies(self):
        # return ""
        return self.account_info.account.cookies

    @staticmethod
    def _sign_pc_params(params: Dict[str, Any]) -> str:
        sign_text = ""
        for key in sorted(params):
            if key in {"sign", "sig"} or params[key] is None:
                continue
            sign_text += f"{key}={params[key]}"
        sign_text += PC_SIGN_SECRET
        return hashlib.md5(sign_text.encode("utf-8")).hexdigest()

    async def _fetch_pc_json(
        self,
        uri: str,
        method: str = "GET",
        params: Optional[Dict[str, Any]] = None,
        data: Optional[Dict[str, Any]] = None,
        use_sign: bool = False,
    ) -> Dict:
        """
        Fetch Tieba's current PC JSON APIs with the account-pool cookie.

        Tieba's 2026 PC rewrite moved search/detail/forum pages behind signed
        JSON APIs. These endpoints accept normal browser cookies, so Pro can use
        its existing cookie account pool instead of requiring Playwright.
        """
        await self.check_ip_expired()
        params = {k: v for k, v in (params or {}).items() if v is not None}
        data = {k: v for k, v in (data or {}).items() if v is not None}
        if use_sign:
            sign_source = data if method.upper() == "POST" else params
            sign_source.setdefault("subapp_type", "pc")
            sign_source.setdefault("_client_type", "20")
            sign_source["sign"] = self._sign_pc_params(sign_source)

        url = f"{TIEBA_URL}{uri}"
        if params:
            url = f"{url}?{urlencode(params)}"

        headers = dict(self.headers)
        body = None
        if method.upper() == "POST":
            headers["Content-Type"] = "application/x-www-form-urlencoded;charset=UTF-8"
            body = urlencode(data)

        async with AsyncSession(impersonate="chrome") as client:
            response = await client.request(
                method,
                url,
                timeout=self.timeout,
                headers=headers,
                data=body,
                proxy=self._proxies,
            )

        if response.status_code != 200:
            utils.logger.error(
                f"Tieba PC API failed, method: {method}, url: {url}, status code: {response.status_code}"
            )
            utils.logger.error(f"Tieba PC API failed, response: {response.text}")
            raise Exception(
                f"Tieba PC API failed, method: {method}, url: {url}, status code: {response.status_code}"
            )

        try:
            json_data = response.json()
        except Exception as exc:
            raise Exception(
                f"Tieba PC API returned non-JSON, url={url}, body={response.text[:500]}"
            ) from exc

        error_code = json_data.get("error_code", json_data.get("no", 0))
        if str(error_code) not in {"0", "None"}:
            raise Exception(f"Tieba PC API error, url={url}, response={json_data}")
        return json_data

    async def _get_pc_tbs(self) -> str:
        if self._pc_tbs:
            return self._pc_tbs
        sync_data = await self._fetch_pc_json(
            "/c/s/pc/sync",
            params={"subapp_type": "pc", "_client_type": "20"},
            use_sign=True,
        )
        self._pc_tbs = sync_data.get("data", {}).get("anti", {}).get("tbs", "")
        if not self._pc_tbs:
            raise Exception(f"Can not get Tieba tbs from pc sync API: {sync_data}")
        return self._pc_tbs

    async def _get_pc_page_data(self, note_id: str, page: int = 1) -> Dict:
        tbs = await self._get_pc_tbs()
        return await self._fetch_pc_json(
            "/c/f/pb/page_pc",
            method="POST",
            data={
                "pn": page,
                "lz": 0,
                "r": 2,
                "mark_type": 0,
                "back": 0,
                "fr": "",
                "kz": note_id,
                "session_request_times": 1,
                "tbs": tbs,
                "subapp_type": "pc",
                "_client_type": "20",
            },
            use_sign=True,
        )

    @staticmethod
    def _extract_creator_portrait(creator_url: str) -> str:
        creator_url = (creator_url or "").strip()
        if not creator_url:
            return ""
        if not creator_url.startswith(("http://", "https://")):
            return creator_url.split("?")[0]
        parsed = urlparse(creator_url)
        query = parse_qs(parsed.query)
        portrait = (
            query.get("id", [""])[0]
            or query.get("portrait", [""])[0]
            or query.get("un", [""])[0]
        )
        return unquote(portrait).split("?")[0]

    async def update_account_info(self):
        """
        更新客户端的账号信息, 该方法会一直尝试获取新的账号信息，直到获取到一个有效的账号信息
        Returns:

        """
        have_account = False
        while not have_account:
            utils.logger.info(
                f"[BaiduTieBaClient.update_account_info] try to get a new account"
            )
            self.account_info = (
                await self.account_with_ip_pool.get_account_with_ip_info()
            )
            have_account = await self.pong()
            if not have_account:
                utils.logger.info(
                    f"[BaiduTieBaClient.update_account_info] current account {self.account_info.account.account_name} is invalid, try to get a new one"
                )

    async def mark_account_invalid(self, account_with_ip: AccountWithIpModel):
        """
        标记账号为无效
        Args:
            account_with_ip:

        Returns:

        """
        if self.account_with_ip_pool:
            await self.account_with_ip_pool.mark_account_invalid(
                account_with_ip.account
            )
            await self.account_with_ip_pool.mark_ip_invalid(account_with_ip.ip_info)

    async def check_ip_expired(self):
        """
        检查IP是否过期, 由于IP的过期时间在运行中是不确定的，所以每次请求都需要验证下IP是否过期
        如果过期了，那么需要重新获取一个新的IP，赋值给当前账号信息
        Returns:

        """
        if (
                config.ENABLE_IP_PROXY
                and self.account_info.ip_info
                and self.account_info.ip_info.is_expired
        ):
            utils.logger.info(
                f"[BaiduTieBaClient.request] current ip {self.account_info.ip_info.ip} is expired, "
                f"mark it invalid and try to get a new one"
            )
            await self.account_with_ip_pool.mark_ip_invalid(self.account_info.ip_info)
            self.account_info.ip_info = (
                await self.account_with_ip_pool.proxy_ip_pool.get_proxy()
            )

    @retry(stop=stop_after_attempt(5), wait=wait_fixed(1))
    async def request(self, method, url, **kwargs) -> Union[Response, Dict]:
        """
        封装curl_cffi的公共请求方法，对请求响应做一些处理
        Args:
            method: 请求方法
            url: 请求的URL
            **kwargs: 其他请求参数，例如请求头、请求体等

        Returns:

        """
        await self.check_ip_expired()
        need_return_ori_response = kwargs.get("return_response", False)
        if "return_response" in kwargs:
            del kwargs["return_response"]

        async with AsyncSession(impersonate="chrome") as client:
            response = await client.request(
                method, url, timeout=self.timeout, headers=self.headers, proxy=self._proxies, **kwargs
            )

        if response.status_code != 200:
            utils.logger.error(
                f"Request failed, method: {method}, url: {url}, status code: {response.status_code}"
            )
            utils.logger.error(f"Request failed, response: {response.text}")
            raise Exception(
                f"Request failed, method: {method}, url: {url}, status code: {response.status_code}"
            )

        if response.text == "" or response.text == "blocked":
            utils.logger.error(f"request params incrr, response.text: {response.text}")
            raise Exception("account blocked")

        if need_return_ori_response:
            return response

        return response.json()

    async def get(self, uri: str, params=None, **kwargs) -> Union[Response, Dict]:
        """
        GET请求，对请求头签名
        Args:
            uri: 请求路由
            params: 请求参数

        Returns:

        """
        final_uri = uri
        if params and isinstance(params, dict):
            final_uri = f"{uri}?" f"{urlencode(params)}"
        try:
            return await self.request(
                method="GET", url=f"{TIEBA_URL}{final_uri}", **kwargs
            )
        except RetryError as e:
            # 获取原始异常
            original_exception = e.last_attempt.exception()
            traceback.print_exception(
                type(original_exception),
                original_exception,
                original_exception.__traceback__,
            )

            utils.logger.error(
                f"[BaiduTieBaClient.get] 请求uri:{uri} 重试均失败了，尝试更换账号与IP再次发起重试"
            )
            try:
                utils.logger.info(
                    f"[BaiduTieBaClient.get] 请求uri:{uri} 尝试更换IP再次发起重试..."
                )
                await self.account_with_ip_pool.mark_ip_invalid(
                    self.account_info.ip_info
                )
                if config.ENABLE_IP_PROXY:
                    self.account_info.ip_info = (
                        await self.account_with_ip_pool.proxy_ip_pool.get_proxy()
                    )
                    return await self.request(
                        method="GET", url=f"{TIEBA_URL}{final_uri}", **kwargs
                    )

            except RetryError as ee:
                # 获取原始异常
                original_exception = ee.last_attempt.exception()
                traceback.print_exception(
                    type(original_exception),
                    original_exception,
                    original_exception.__traceback__,
                )

                utils.logger.error(
                    f"[BaiduTieBaClient.get] 请求uri:{uri}，IP更换后还是失败，尝试更换账号与IP再次发起重试"
                )
                await self.mark_account_invalid(self.account_info)
                await self.update_account_info()
                return await self.request(
                    method="GET", url=f"{TIEBA_URL}{final_uri}", **kwargs
                )

    async def post(self, uri: str, data: dict, **kwargs) -> Dict:
        """
        POST请求，对请求头签名
        Args:
            uri: 请求路由
            data: 请求体参数

        Returns:

        """
        json_str = json.dumps(data, separators=(",", ":"), ensure_ascii=False)
        try:
            return await self.request(
                method="POST", url=f"{TIEBA_URL}{uri}", data=json_str, **kwargs
            )
        except RetryError as e:
            # 获取原始异常
            original_exception = e.last_attempt.exception()
            traceback.print_exception(
                type(original_exception),
                original_exception,
                original_exception.__traceback__,
            )

            utils.logger.error(
                f"[BaiduTieBaClient.post] 请求uri:{uri} 重试均失败了，尝试更换账号与IP再次发起重试"
            )
            try:
                utils.logger.info(
                    f"[BaiduTieBaClient.post] 请求uri:{uri} 尝试更换IP再次发起重试..."
                )
                await self.account_with_ip_pool.mark_ip_invalid(
                    self.account_info.ip_info
                )
                self.account_info.ip_info = (
                    await self.account_with_ip_pool.proxy_ip_pool.get_proxy()
                )
                return await self.request(
                    method="POST", url=f"{TIEBA_URL}{uri}", data=json_str, **kwargs
                )
            except RetryError as ee:
                # 获取原始异常
                original_exception = ee.last_attempt.exception()
                traceback.print_exception(
                    type(original_exception),
                    original_exception,
                    original_exception.__traceback__,
                )

                utils.logger.error(
                    f"[BaiduTieBaClient.post]请求uri:{uri}，IP更换后还是失败，尝试更换账号与IP再次发起重试"
                )
                await self.mark_account_invalid(self.account_info)
                await self.update_account_info()
                return await self.request(
                    method="POST", url=f"{TIEBA_URL}{uri}", data=json_str, **kwargs
                )

    async def pong(self) -> bool:
        """
        用于检查登录态是否失效了
        Returns:

        """
        utils.logger.info("[BaiduTieBaClient.pong] Begin to pong tieba...")
        try:
            res = await self._fetch_pc_json(
                "/c/s/pc/sync",
                params={"subapp_type": "pc", "_client_type": "20"},
                use_sign=True,
            )
            user_info = res.get("data", {}).get("user", {})
            ping_flag = bool(user_info.get("is_login"))
            if not ping_flag:
                utils.logger.warning(
                    "[BaiduTieBaClient.pong] user not login, will try to login again..."
                )
        except Exception as e:
            utils.logger.error(
                f"[BaiduTieBaClient.pong] Ping xhs failed: {e},current account: {self.account_info.account.account_name} and try to login again..."
            )
            ping_flag = False
        return ping_flag

    async def get_notes_by_keyword(
            self,
            keyword: str,
            page: int = 1,
            page_size: int = 10,
            sort: SearchSortType = SearchSortType.TIME_DESC,
            note_type: SearchNoteType = SearchNoteType.FIXED_THREAD,
    ) -> List[TiebaNote]:
        """
        根据关键词搜索贴吧帖子
        Args:
            keyword: 关键词
            page: 分页第几页
            page_size: 每页大小
            sort: 结果排序方式
            note_type: 帖子类型（主题贴｜主题+回复混合模式）
        Returns:

        """
        params = {
            "rn": max(page_size, 20),
            "st": sort.value,
            "word": keyword,
            "needbrand": 1,
            "sug_type": 2,
            "pn": page,
            "come_from": "search",
            "subapp_type": "pc",
            "_client_type": "20",
        }
        api_data = await self._fetch_pc_json(
            "/mo/q/search/multsearch",
            params=params,
            use_sign=True,
        )
        return self.page_extractor.extract_search_note_list_from_api(api_data)[:page_size]

    async def get_note_by_id(self, note_id: str) -> TiebaNote:
        """
        根据帖子ID获取帖子详情
        Args:
            note_id:

        Returns:

        """
        api_data = await self._get_pc_page_data(note_id=note_id, page=1)
        return self.page_extractor.extract_note_detail_from_api(api_data)

    async def get_note_comments(self, note_id: str, page: int = 1) -> Dict:
        """
        获取指定帖子下的所有一级评论
        Args:
            note_id: 帖子ID
            page: 页码

        Returns:

        """
        return await self._get_pc_page_data(note_id=note_id, page=page)

    async def get_note_sub_comments(self, note_id: str, parment_comment_id: str, tieba_id: str, page: int = 1) -> str:
        """
        获取指定帖子下的所有子评论
        Args:
            note_id: 帖子ID
            parment_comment_id: 父评论ID
            tieba_id: 贴吧ID
            page: 页码

        Returns:

        """
        uri = f"/p/comment"
        params = {
            "tid": note_id,
            "pid": parment_comment_id,
            "fid": tieba_id,
            "pn": page,
        }
        response = await self.get(uri, params=params, return_response=True)
        return response.text

    async def get_notes_by_tieba_name(
            self, tieba_name: str, page_num: int
    ) -> List[TiebaNote]:
        """
        根据贴吧名称获取帖子列表
        Args:
            tieba_name: 贴吧名称
            page_num: 分页数量

        Returns:

        """
        page_size = 30
        api_page = page_num // page_size + 1
        tbs = await self._get_pc_tbs()
        api_data = await self._fetch_pc_json(
            "/c/f/frs/page_pc",
            method="POST",
            data={
                "kw": quote(tieba_name),
                "pn": api_page,
                "sort_type": -1,
                "is_newfrs": 1,
                "is_newfeed": 1,
                "rn": page_size,
                "rn_need": 10,
                "tbs": tbs,
                "subapp_type": "pc",
                "_client_type": "20",
            },
            use_sign=True,
        )
        return self.page_extractor.extract_tieba_note_list_from_frs_api(api_data)[:page_size]

    async def get_creator_info_by_url(self, creator_url: str):
        """
        根据创作者ID获取创作者信息
        Args:
            creator_url: 创作者主页URL

        Returns:

        """
        portrait = self._extract_creator_portrait(creator_url)
        if not portrait:
            raise Exception(f"Can not extract Tieba creator portrait from url: {creator_url}")
        api_data = await self._fetch_pc_json(
            "/c/u/pc/homeSidebarRight",
            params={
                "portrait": portrait,
                "un": "",
                "subapp_type": "pc",
                "_client_type": "20",
            },
            use_sign=True,
        )
        return self.page_extractor.extract_creator_info_from_api(api_data)

    async def get_notes_by_creator_portrait(
        self, portrait: str, page_number: int, page_size: int = 20
    ) -> Dict:
        """
        根据创作者 portrait 获取帖子列表
        """
        return await self._fetch_pc_json(
            "/c/u/feed/myThread",
            params={
                "pn": page_number,
                "rn": page_size,
                "portrait": portrait,
                "type": 1,
                "un": "",
                "subapp_type": "pc",
                "_client_type": "20",
            },
            use_sign=True,
        )

    async def get_notes_by_creator(self, user_name: str, page_number: int) -> Dict:
        """
        根据创作者获取创作者的所有帖子
        Args:
            user_name:
            page_number:

        Returns:

        """
        uri = f"/home/get/getthread"
        params = {
            "un": user_name,
            "pn": page_number,
            "ie": "utf-8",
            "_": utils.get_current_timestamp(),
        }
        return await self.get(uri, params=params)
