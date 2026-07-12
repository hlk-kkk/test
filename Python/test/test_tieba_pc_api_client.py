# -*- coding: utf-8 -*-

import asyncio

from media_platform.tieba.client import BaiduTieBaClient
from model.m_baidu_tieba import TiebaNote


def test_search_uses_current_pc_api_page_number():
    client = BaiduTieBaClient()
    calls = []

    async def fake_fetch(uri, method="GET", params=None, data=None, use_sign=False):
        calls.append((uri, params, use_sign))
        return {"no": 0, "data": {"card_list": []}}

    client._fetch_pc_json = fake_fetch

    asyncio.run(client.get_notes_by_keyword("编程兼职", page=2, page_size=10))

    assert calls[0][0] == "/mo/q/search/multsearch"
    assert calls[0][1]["pn"] == 2
    assert calls[0][2] is True


def test_get_note_comments_reads_requested_pc_page():
    client = BaiduTieBaClient()
    pages = []

    async def fake_get_page_data(note_id, page=1):
        pages.append((note_id, page))
        return {"error_code": 0, "post_list": []}

    client._get_pc_page_data = fake_get_page_data

    asyncio.run(client.get_note_comments("9835114923", page=2))

    assert pages == [("9835114923", 2)]


def test_creator_feed_walks_current_pc_api():
    client = BaiduTieBaClient()
    pages = []

    async def fake_fetch(uri, method="GET", params=None, data=None, use_sign=False):
        pages.append(params["pn"])
        return {
            "error_code": 0,
            "data": {
                "has_more": 0,
                "list": [{"thread_info": {"id": "1001", "tid": "1001"}}],
            },
        }

    client._fetch_pc_json = fake_fetch

    result = asyncio.run(client.get_notes_by_creator_portrait("tb.1.creator", 1))
    thread_ids = client.page_extractor.extract_creator_thread_id_list_from_api(result)

    assert pages == [1]
    assert thread_ids == ["1001"]


def test_tieba_url_normalizers():
    assert (
        BaiduTieBaClient._extract_creator_portrait(
            "https://tieba.baidu.com/home/main?id=tb.1.raw&fr=frs"
        )
        == "tb.1.raw"
    )
    assert BaiduTieBaClient._extract_creator_portrait("tb.1.example") == "tb.1.example"


def test_note_model_accepts_frs_stub():
    note = TiebaNote(
        note_id="10376710029",
        title="",
        note_url="https://tieba.baidu.com/p/10376710029",
        tieba_name="加工中心吧",
        tieba_link="https://tieba.baidu.com/f?kw=%E5%8A%A0%E5%B7%A5%E4%B8%AD%E5%BF%83",
    )

    assert note.note_id == "10376710029"
