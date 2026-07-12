# -*- coding: utf-8 -*-
# Copyright (c) 2025 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-Python project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Python/blob/main/media_platform/tieba/help.py
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


# -*- coding: utf-8 -*-
import html
import json
import re
from typing import Any, Dict, List, Tuple
from urllib.parse import quote, urljoin

from parsel import Selector

from constant import baidu_tieba as const
from model.m_baidu_tieba import TiebaComment, TiebaCreator, TiebaNote
from pkg.tools import utils
from pkg.tools.user_privacy import anonymize_user_id, mask_nickname


class TieBaExtractor:
    def __init__(self):
        pass

    @staticmethod
    def _class_contains(class_name: str) -> str:
        return f"contains(concat(' ', normalize-space(@class), ' '), ' {class_name} ')"

    @staticmethod
    def _normalize_text(text: str) -> str:
        return re.sub(r"\s+", " ", text or "").strip()

    @classmethod
    def _selector_text(cls, selector: Selector, xpath: str) -> str:
        node = selector.xpath(xpath)
        if not node:
            return ""
        return cls._normalize_text(node[0].xpath("string(.)").get(default=""))

    @staticmethod
    def _absolute_url(url: str) -> str:
        return urljoin(const.TIEBA_URL, (url or "").strip())

    @staticmethod
    def _extract_note_id_from_url(url: str) -> str:
        note_id_match = re.search(r"/p/(\d+)", url or "")
        return note_id_match.group(1) if note_id_match else ""

    @staticmethod
    def _text_to_int(text: str) -> int:
        match = re.search(r"\d+", text or "")
        return int(match.group(0)) if match else 0

    @staticmethod
    def _ensure_tieba_suffix(tieba_name: str) -> str:
        tieba_name = (tieba_name or "").strip()
        return tieba_name if not tieba_name or tieba_name.endswith("吧") else f"{tieba_name}吧"

    @classmethod
    def _tieba_link_from_name(cls, tieba_name: str) -> str:
        if not tieba_name:
            return const.TIEBA_URL
        return f"{const.TIEBA_URL}/f?kw={quote(tieba_name.removesuffix('吧'))}"

    @classmethod
    def _extract_api_content_text(cls, content: Any) -> str:
        if isinstance(content, str):
            return cls._normalize_text(content)
        if not isinstance(content, list):
            return ""
        text_list: List[str] = []
        for item in content:
            if not isinstance(item, dict):
                continue
            text = item.get("text") or item.get("c") or ""
            if text:
                text_list.append(str(text))
        return cls._normalize_text("".join(text_list))

    @staticmethod
    def _api_user_map(api_data: Dict) -> Dict[str, Dict]:
        return {
            str(user.get("id")): user
            for user in api_data.get("user_list", [])
            if user.get("id")
        }

    def extract_search_note_list_from_api(self, api_data: Dict) -> List[TiebaNote]:
        result: List[TiebaNote] = []
        cards = api_data.get("data", {}).get("card_list", [])
        for card in cards:
            if card.get("cardInfo") != "thread" and card.get("cardStyle") != "thread":
                continue
            item = card.get("data") or {}
            note_id = str(item.get("tid") or "")
            if not note_id:
                continue
            user = item.get("user") or {}
            tieba_name = self._ensure_tieba_suffix(item.get("forum_name") or "")
            result.append(
                TiebaNote(
                    note_id=note_id,
                    title=self._normalize_text(item.get("title") or ""),
                    desc=self._normalize_text(item.get("content") or ""),
                    note_url=f"{const.TIEBA_URL}/p/{note_id}",
                    publish_time=utils.get_time_str_from_unix_time(
                        item.get("time") or item.get("create_time") or 0
                    ),
                    creator_hash=anonymize_user_id(user.get("id", "")),
                    user_nickname=mask_nickname(
                        user.get("show_nickname") or user.get("user_name") or ""
                    ),
                    tieba_name=tieba_name,
                    tieba_link=self._tieba_link_from_name(tieba_name),
                    total_replay_num=item.get("post_num") or 0,
                )
            )
        return result

    def extract_note_detail_from_api(self, api_data: Dict) -> TiebaNote:
        thread = api_data.get("thread") or {}
        first_floor = api_data.get("first_floor") or {}
        forum = api_data.get("forum") or api_data.get("display_forum") or {}
        page = api_data.get("page") or {}
        user_map = self._api_user_map(api_data)
        author = user_map.get(str(first_floor.get("author_id"))) or {}
        note_id = str(
            thread.get("id") or thread.get("tid") or first_floor.get("tid") or ""
        )
        tieba_name = self._ensure_tieba_suffix(forum.get("name") or "")
        return TiebaNote(
            note_id=note_id,
            title=self._clean_title(
                thread.get("title") or first_floor.get("title") or "", tieba_name
            ),
            desc=self._extract_api_content_text(
                first_floor.get("content")
                or thread.get("origin_thread_info", {}).get("abstract")
                or thread.get("origin_thread_info", {}).get("content")
            ),
            note_url=f"{const.TIEBA_URL}/p/{note_id}",
            publish_time=utils.get_time_str_from_unix_time(
                first_floor.get("time") or thread.get("create_time") or 0
            ),
            creator_hash=anonymize_user_id(first_floor.get("author_id")),
            user_nickname=mask_nickname(author.get("name_show") or author.get("name") or ""),
            tieba_name=tieba_name,
            tieba_link=self._tieba_link_from_name(tieba_name),
            total_replay_num=thread.get("reply_num") or 0,
            total_replay_page=page.get("total_page") or 0,
        )

    def extract_tieba_note_parent_comments_from_api(
        self, api_data: Dict, note_detail: TiebaNote
    ) -> List[TiebaComment]:
        forum = api_data.get("forum") or api_data.get("display_forum") or {}
        tieba_id = str(forum.get("id") or "")
        tieba_name = note_detail.tieba_name or self._ensure_tieba_suffix(
            forum.get("name") or ""
        )
        tieba_link = note_detail.tieba_link or self._tieba_link_from_name(tieba_name)
        user_map = self._api_user_map(api_data)
        result: List[TiebaComment] = []
        for item in api_data.get("post_list", []):
            comment_id = str(item.get("id") or "")
            if not comment_id:
                continue
            user = user_map.get(str(item.get("author_id"))) or {}
            result.append(
                TiebaComment(
                    comment_id=comment_id,
                    sub_comment_count=item.get("sub_post_number") or 0,
                    content=self._extract_api_content_text(item.get("content")),
                    note_url=note_detail.note_url,
                    creator_hash=anonymize_user_id(item.get("author_id")),
                    user_nickname=mask_nickname(user.get("name_show") or user.get("name") or ""),
                    tieba_id=tieba_id,
                    tieba_name=tieba_name,
                    tieba_link=tieba_link,
                    publish_time=utils.get_time_str_from_unix_time(item.get("time") or 0),
                    note_id=note_detail.note_id,
                )
            )
        return result

    def extract_creator_info_from_api(self, api_data: Dict) -> TiebaCreator:
        user = api_data.get("data", {}).get("user", {})
        if not user:
            raise ValueError(f"Creator API response does not contain user info: {api_data}")
        return TiebaCreator(
            creator_hash=anonymize_user_id(user.get("id", "")),
            nickname=mask_nickname(user.get("name_show") or user.get("name") or ""),
            follows=int(user.get("concern_num") or 0),
            fans=int(user.get("fans_num") or 0),
            registration_duration=str(user.get("tb_age", "")),
        )

    @staticmethod
    def extract_creator_thread_id_list_from_api(api_data: Dict) -> List[str]:
        thread_ids: List[str] = []
        for item in api_data.get("data", {}).get("list", []):
            thread_info = item.get("thread_info") or {}
            thread_id = thread_info.get("tid") or thread_info.get("id")
            if thread_id:
                thread_ids.append(str(thread_id))
        return thread_ids

    def extract_tieba_note_list_from_frs_api(self, api_data: Dict) -> List[TiebaNote]:
        forum = api_data.get("forum", {})
        tieba_name = self._ensure_tieba_suffix(forum.get("name") or "")
        tieba_link = self._tieba_link_from_name(tieba_name)
        tids = [
            tid.strip()
            for tid in str(forum.get("tids") or "").split(",")
            if tid.strip()
        ]
        return [
            TiebaNote(
                note_id=tid,
                title="",
                desc="",
                note_url=f"{const.TIEBA_URL}/p/{tid}",
                tieba_name=tieba_name,
                tieba_link=tieba_link,
            )
            for tid in tids
        ]

    @classmethod
    def _clean_title(cls, title: str, tieba_name: str = "") -> str:
        title = cls._normalize_text(title)
        title = re.sub(r"_(?:百度贴吧|Baidu Tieba)$", "", title).strip()
        for name in {tieba_name, tieba_name.removesuffix("吧")}:
            if name:
                title = title.replace(f"【{name}】", "").strip()
        return title

    @staticmethod
    def extract_search_note_list(page_content: str) -> List[TiebaNote]:
        """
        提取贴吧帖子列表，这里提取的关键词搜索结果页的数据，还缺少帖子的回复数和回复页等数据
        Args:
            page_content: 页面内容的HTML字符串

        Returns:
            包含帖子信息的字典列表
        """
        xpath_selector = "//div[@class='s_post']"
        post_list = Selector(text=page_content).xpath(xpath_selector)
        result: List[TiebaNote] = []
        for post in post_list:
            author_link = post.xpath(".//a[starts-with(@href, '/home/main')]/@href").get(
                default=""
            )
            tieba_note = TiebaNote(
                note_id=post.xpath(".//span[@class='p_title']/a/@data-tid")
                .get(default="")
                .strip(),
                title=post.xpath(".//span[@class='p_title']/a/text()")
                .get(default="")
                .strip(),
                desc=post.xpath(".//div[@class='p_content']/text()")
                .get(default="")
                .strip(),
                note_url=const.TIEBA_URL
                + post.xpath(".//span[@class='p_title']/a/@href").get(default=""),
                creator_hash=anonymize_user_id(author_link),
                user_nickname=mask_nickname(
                    post.xpath(".//a[starts-with(@href, '/home/main')]/font/text()")
                    .get(default="")
                    .strip()
                ),
                tieba_name=post.xpath(".//a[@class='p_forum']/font/text()")
                .get(default="")
                .strip(),
                tieba_link=const.TIEBA_URL
                + post.xpath(".//a[@class='p_forum']/@href").get(default=""),
                publish_time=post.xpath(".//font[@class='p_green p_date']/text()")
                .get(default="")
                .strip(),
            )
            result.append(tieba_note)
        return result

    def extract_tieba_note_list(self, page_content: str) -> List[TiebaNote]:
        """
        提取贴吧帖子列表
        Args:
            page_content:

        Returns:

        """
        page_content = page_content.replace("<!--", "")
        content_selector = Selector(text=page_content)
        xpath_selector = "//ul[@id='thread_list']/li"
        post_list = content_selector.xpath(xpath_selector)
        result: List[TiebaNote] = []
        for post_selector in post_list:
            post_field_value: Dict = self.extract_data_field_value(post_selector)
            if not post_field_value:
                continue
            note_id = str(post_field_value.get("id"))
            author_info = post_field_value.get("author") or {}
            tieba_note = TiebaNote(
                note_id=note_id,
                title=post_selector.xpath(".//a[@class='j_th_tit ']/text()")
                .get(default="")
                .strip(),
                desc=post_selector.xpath(
                    ".//div[@class='threadlist_abs threadlist_abs_onlyline ']/text()"
                )
                .get(default="")
                .strip(),
                note_url=const.TIEBA_URL + f"/p/{note_id}",
                creator_hash=anonymize_user_id(
                    author_info.get("user_id")
                    or author_info.get("id")
                    or post_field_value.get("author_id")
                ),
                user_nickname=mask_nickname(
                    post_field_value.get("authoer_nickname")
                    or post_field_value.get("author_name")
                ),
                tieba_name=content_selector.xpath(
                    "//a[@class='card_title_fname']/text()"
                )
                .get(default="")
                .strip(),
                tieba_link=const.TIEBA_URL
                + content_selector.xpath("//a[@class='card_title_fname']/@href").get(
                    default=""
                ),
                total_replay_num=post_field_value.get("reply_num", 0),
            )
            result.append(tieba_note)
        return result

    def extract_note_detail(self, page_content: str) -> TiebaNote:
        """
        提取贴吧帖子详情
        Args:
            page_content:

        Returns:

        """
        content_selector = Selector(text=page_content)
        first_floor_selector = content_selector.xpath("//div[@class='p_postlist'][1]")
        only_view_author_link = (
            content_selector.xpath("//*[@id='lzonly_cntn']/@href")
            .get(default="")
            .strip()
        )
        note_id = only_view_author_link.split("?")[0].split("/")[-1]
        # 帖子回复数、回复页数
        thread_num_infos = content_selector.xpath(
            "//div[@id='thread_theme_5']//li[@class='l_reply_num']//span[@class='red']"
        )
        # 发表时间
        other_info_content = (
            content_selector.xpath(".//div[@class='post-tail-wrap']")
            .get(default="")
            .strip()
        )
        publish_time = self.extract_pub_time(other_info_content)
        first_floor_field_value = self.extract_data_field_value(first_floor_selector)
        author_info = first_floor_field_value.get("author") or {}
        note = TiebaNote(
            note_id=note_id,
            title=content_selector.xpath("//title/text()").get(default="").strip(),
            desc=content_selector.xpath("//meta[@name='description']/@content")
            .get(default="")
            .strip(),
            note_url=const.TIEBA_URL + f"/p/{note_id}",
            creator_hash=anonymize_user_id(
                author_info.get("user_id")
                or author_info.get("id")
                or first_floor_field_value.get("author_id")
            ),
            user_nickname=mask_nickname(
                first_floor_selector.xpath(
                    ".//a[@class='p_author_name j_user_card']/text()"
                )
                .get(default="")
                .strip()
            ),
            tieba_name=content_selector.xpath("//a[@class='card_title_fname']/text()")
            .get(default="")
            .strip(),
            tieba_link=const.TIEBA_URL
            + content_selector.xpath("//a[@class='card_title_fname']/@href").get(
                default=""
            ),
            publish_time=publish_time,
            total_replay_num=thread_num_infos[0]
            .xpath("./text()")
            .get(default="")
            .strip(),
            total_replay_page=thread_num_infos[1]
            .xpath("./text()")
            .get(default="")
            .strip(),
        )
        note.title = note.title.replace(f"【{note.tieba_name}】_百度贴吧", "")
        return note

    def extract_tieba_note_parment_comments(
        self, page_content: str, note_id: str
    ) -> List[TiebaComment]:
        """
        提取贴吧帖子一级评论
        Args:
            page_content:
            note_id:

        Returns:

        """
        xpath_selector = "//div[@class='l_post l_post_bright j_l_post clearfix  ']"
        comment_list = Selector(text=page_content).xpath(xpath_selector)
        result: List[TiebaComment] = []
        for comment_selector in comment_list:
            comment_field_value: Dict = self.extract_data_field_value(comment_selector)
            if not comment_field_value:
                continue
            tieba_name = (
                comment_selector.xpath("//a[@class='card_title_fname']/text()")
                .get(default="")
                .strip()
            )
            other_info_content = (
                comment_selector.xpath(".//div[@class='post-tail-wrap']")
                .get(default="")
                .strip()
            )
            publish_time = self.extract_pub_time(other_info_content)
            author_info = comment_field_value.get("author") or {}
            tieba_comment = TiebaComment(
                comment_id=str(comment_field_value.get("content").get("post_id")),
                sub_comment_count=comment_field_value.get("content").get("comment_num"),
                content=utils.extract_text_from_html(
                    comment_field_value.get("content").get("content")
                ),
                note_url=const.TIEBA_URL + f"/p/{note_id}",
                creator_hash=anonymize_user_id(
                    author_info.get("user_id")
                    or author_info.get("id")
                    or comment_field_value.get("author_id")
                ),
                user_nickname=mask_nickname(
                    comment_selector.xpath(
                        ".//a[@class='p_author_name j_user_card']/text()"
                    )
                    .get(default="")
                    .strip()
                ),
                tieba_id=str(comment_field_value.get("content").get("forum_id", "")),
                tieba_name=tieba_name,
                tieba_link=f"https://tieba.baidu.com/f?kw={tieba_name}",
                publish_time=publish_time,
                note_id=note_id,
            )
            result.append(tieba_comment)
        return result

    def extract_tieba_note_sub_comments(
        self, page_content: str, parent_comment: TiebaComment
    ) -> List[TiebaComment]:
        """
        提取贴吧帖子二级评论
        Args:
            page_content:
            parent_comment:

        Returns:

        """
        selector = Selector(page_content)
        comments = []
        comment_ele_list = selector.xpath(
            "//li[@class='lzl_single_post j_lzl_s_p first_no_border']"
        )
        comment_ele_list.extend(
            selector.xpath("//li[@class='lzl_single_post j_lzl_s_p ']")
        )
        for comment_ele in comment_ele_list:
            comment_value = self.extract_data_field_value(comment_ele)
            if not comment_value:
                continue
            comment_user_a_selector = comment_ele.xpath(
                "./a[@class='j_user_card lzl_p_p']"
            )[0]
            content = utils.extract_text_from_html(
                comment_ele.xpath(".//span[@class='lzl_content_main']").get(default="")
            )
            comment = TiebaComment(
                comment_id=str(comment_value.get("spid")),
                content=content,
                creator_hash=anonymize_user_id(
                    comment_value.get("user_id") or comment_value.get("id")
                ),
                user_nickname=mask_nickname(comment_value.get("showname")),
                publish_time=comment_ele.xpath(".//span[@class='lzl_time']/text()")
                .get(default="")
                .strip(),
                parent_comment_id=parent_comment.comment_id,
                note_id=parent_comment.note_id,
                note_url=parent_comment.note_url,
                tieba_id=parent_comment.tieba_id,
                tieba_name=parent_comment.tieba_name,
                tieba_link=parent_comment.tieba_link,
            )
            comments.append(comment)

        return comments

    def extract_creator_info(self, html_content: str) -> TiebaCreator:
        """
        提取贴吧创作者信息
        Args:
            html_content:

        Returns:

        """
        selector = Selector(text=html_content)
        userinfo_userdata_selector = selector.xpath("//div[@class='userinfo_userdata']")
        follow_fans_selector = selector.xpath("//span[@class='concern_num']")
        follows, fans = 0, 0
        if len(follow_fans_selector) == 2:
            follows, fans = self.extract_follow_and_fans(follow_fans_selector)
        user_content = userinfo_userdata_selector.get(default="")
        return TiebaCreator(
            creator_hash=anonymize_user_id(
                selector.xpath("//p[@class='space']/a/@href").get(default="")
            ),
            nickname=mask_nickname(
                selector.xpath(".//span[@class='userinfo_username ']/text()")
                .get(default="")
                .strip()
            ),
            follows=follows,
            fans=fans,
            registration_duration=self.extract_registration_duration(user_content),
        )

    def extract_tieba_thread_id_list_from_creator_page(
        self, html_content: str
    ) -> List[str]:
        """
        提取贴吧创作者主页的帖子列表
        Args:
            html_content:

        Returns:

        """
        selector = Selector(text=html_content)
        thread_id_list = []
        xpath_selector = (
            "//ul[@class='new_list clearfix']//div[@class='thread_name']/a[1]/@href"
        )
        thread_url_list = selector.xpath(xpath_selector).getall()
        for thread_url in thread_url_list:
            thread_id = thread_url.split("?")[0].split("/")[-1]
            thread_id_list.append(thread_id)
        return thread_id_list

    def extract_pub_time(self, html_content: str) -> str:
        """
        提取发布时间
        Args:
            html_content:

        Returns:

        """
        pattern_pub_time = re.compile(
            r'<span class="tail-info">(\d{4}-\d{2}-\d{2} \d{2}:\d{2})</span>'
        )
        time_match = pattern_pub_time.search(html_content)
        pub_time = time_match.group(1) if time_match else ""
        return pub_time

    @staticmethod
    def extract_follow_and_fans(selectors: List[Selector]) -> Tuple[str, str]:
        """
        提取关注数和粉丝数
        Args:
            selectors:

        Returns:

        """
        pattern = re.compile(r'<span class="concern_num">\(<a[^>]*>(\d+)</a>\)</span>')
        follow_match = pattern.findall(selectors[0].get())
        fans_match = pattern.findall(selectors[1].get())
        follows = follow_match[0] if follow_match else 0
        fans = fans_match[0] if fans_match else 0
        return follows, fans

    @staticmethod
    def extract_registration_duration(html_content: str) -> str:
        """
        "<span>吧龄:1.9年</span>"
        Returns: 1.9年

        """
        pattern = re.compile(r"<span>吧龄:(\S+)</span>")
        match = pattern.search(html_content)
        return match.group(1) if match else ""

    @staticmethod
    def extract_data_field_value(selector: Selector) -> Dict:
        """
        提取data-field的值
        Args:
            selector:

        Returns:

        """
        data_field_value = selector.xpath("./@data-field").get(default="").strip()
        if not data_field_value or data_field_value == "{}":
            return {}
        try:
            # 先使用 html.unescape 处理转义字符 再json.loads 将 JSON 字符串转换为 Python 字典
            unescaped_json_str = html.unescape(data_field_value)
            data_field_dict_value = json.loads(unescaped_json_str)
        except Exception as ex:
            print(f"extract_data_field_value，错误信息：{ex}, 尝试使用其他方式解析")
            data_field_dict_value = {}
        return data_field_dict_value


def test_extract_search_note_list():
    with open("test_data/search_keyword_notes.html", "r", encoding="utf-8") as f:
        content = f.read()
        extractor = TieBaExtractor()
        result = extractor.extract_search_note_list(content)
        print(result)


def test_extract_note_detail():
    with open("test_data/note_detail.html", "r", encoding="utf-8") as f:
        content = f.read()
        extractor = TieBaExtractor()
        result = extractor.extract_note_detail(content)
        print(result.model_dump())


def test_extract_tieba_note_parment_comments():
    with open("test_data/note_comments.html", "r", encoding="utf-8") as f:
        content = f.read()
        extractor = TieBaExtractor()
        result = extractor.extract_tieba_note_parment_comments(content, "123456")
        print(result)


def test_extract_tieba_note_sub_comments():
    with open("test_data/note_sub_comments.html", "r", encoding="utf-8") as f:
        content = f.read()
        extractor = TieBaExtractor()
        fake_parment_comment = TiebaComment(
            comment_id="123456",
            content="content",
            user_nickname="user_nickname",
            publish_time="publish_time",
            parent_comment_id="parent_comment_id",
            note_id="note_id",
            note_url="note_url",
            tieba_id="tieba_id",
            tieba_name="tieba_name",
        )
        result = extractor.extract_tieba_note_sub_comments(
            content, fake_parment_comment
        )
        print(result)


def test_extract_tieba_note_list():
    with open("test_data/tieba_note_list.html", "r", encoding="utf-8") as f:
        content = f.read()
        extractor = TieBaExtractor()
        result = extractor.extract_tieba_note_list(content)
        print(result)
    pass


if __name__ == "__main__":
    # test_extract_search_note_list()
    # test_extract_note_detail()
    # test_extract_tieba_note_parment_comments()
    test_extract_tieba_note_list()
