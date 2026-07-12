# -*- coding: utf-8 -*-

from media_platform.tieba.help import TieBaExtractor


def test_extract_search_note_list_from_current_pc_api():
    api_data = {
        "no": 0,
        "error": "success",
        "data": {
            "card_list": [
                {"cardInfo": "related_user", "cardStyle": "related_user", "data": {}},
                {
                    "cardInfo": "thread",
                    "cardStyle": "thread",
                    "data": {
                        "tid": "10559655942",
                        "title": "数，英，编程老师",
                        "content": "培训班需求，数学，英语，编程老师，专职兼职都可",
                        "time": 1773552643,
                        "user": {"show_nickname": "754023117"},
                        "post_num": 19,
                        "forum_name": "诸城",
                    },
                },
            ]
        },
    }

    notes = TieBaExtractor().extract_search_note_list_from_api(api_data)

    assert len(notes) == 1
    assert notes[0].note_id == "10559655942"
    assert notes[0].title == "数，英，编程老师"
    assert notes[0].tieba_name == "诸城吧"
    assert notes[0].total_replay_num == 19
    assert notes[0].publish_time


def test_extract_note_detail_and_comments_from_current_pc_api():
    api_data = {
        "error_code": 0,
        "thread": {
            "id": 10451142633,
            "title": "这X尔斯对比巴尔斯，我只能说ID正确，允许居功自傲",
            "reply_num": 15,
            "create_time": 1769951446,
        },
        "forum": {"id": 1627732, "name": "dota2"},
        "page": {"total_page": 1},
        "first_floor": {
            "id": 153154064746,
            "author_id": 4089186644,
            "time": 1769951446,
            "content": [{"type": 0, "text": "皮队败决处刑德国编程钢琴师兼职数学家"}],
        },
        "post_list": [
            {
                "id": 153154097267,
                "author_id": 6614897968,
                "time": 1769952062,
                "content": [{"type": 0, "text": "xg现在大树阵容另一个辅助不选控制"}],
                "sub_post_number": 4,
            }
        ],
        "user_list": [
            {
                "id": 4089186644,
                "name_show": "泰高祖蒙斯克",
                "portrait": "tb.1.f893a7af",
                "ip_address": "广东",
            },
            {
                "id": 6614897968,
                "name_show": "期胡希3",
                "portrait": "tb.1.4d0471d4",
                "ip_address": "河北",
            },
        ],
    }

    extractor = TieBaExtractor()
    note = extractor.extract_note_detail_from_api(api_data)
    comments = extractor.extract_tieba_note_parent_comments_from_api(api_data, note)

    assert note.note_id == "10451142633"
    assert note.title == "这X尔斯对比巴尔斯，我只能说ID正确，允许居功自傲"
    assert note.desc == "皮队败决处刑德国编程钢琴师兼职数学家"
    assert note.user_nickname == "泰***克"
    assert note.creator_hash
    assert "user_link" not in note.model_dump()
    assert "user_avatar" not in note.model_dump()
    assert note.tieba_name == "dota2吧"
    assert note.total_replay_num == 15
    assert note.total_replay_page == 1
    assert "ip_location" not in note.model_dump()
    assert len(comments) == 1
    assert comments[0].comment_id == "153154097267"
    assert comments[0].content == "xg现在大树阵容另一个辅助不选控制"
    assert comments[0].user_nickname == "期***3"
    assert comments[0].creator_hash
    assert comments[0].sub_comment_count == 4
    assert "ip_location" not in comments[0].model_dump()


def test_extract_creator_info_and_threads_from_current_pc_api():
    creator_api = {
        "error_code": 0,
        "data": {
            "user": {
                "id": 3546493137,
                "name": "拜月教Alice",
                "name_show": "米米世界大手子",
                "portrait": "tb.1.6ad0cd4a",
                "fans_num": 58,
                "concern_num": 1,
                "sex": 1,
                "tb_age": "7.8",
                "ip_address": "广东",
            }
        },
    }
    feed_api = {
        "error_code": 0,
        "data": {
            "list": [
                {"type": 1, "thread_info": {"id": 10208192951, "tid": 10208192951}},
                {"type": 1, "thread_info": {"id": 9835114923}},
            ]
        },
    }

    extractor = TieBaExtractor()
    creator = extractor.extract_creator_info_from_api(creator_api)
    thread_ids = extractor.extract_creator_thread_id_list_from_api(feed_api)

    assert creator.creator_hash
    assert creator.nickname == "米***子"
    assert "user_id" not in creator.model_dump()
    assert "user_name" not in creator.model_dump()
    assert "gender" not in creator.model_dump()
    assert creator.fans == 58
    assert creator.follows == 1
    assert "ip_location" not in creator.model_dump()
    assert creator.registration_duration == "7.8"
    assert thread_ids == ["10208192951", "9835114923"]


def test_extract_tieba_note_list_from_current_frs_api():
    api_data = {
        "error_code": 0,
        "forum": {
            "id": 351091,
            "name": "加工中心",
            "tids": "10376710029,10636556989,",
        },
    }

    notes = TieBaExtractor().extract_tieba_note_list_from_frs_api(api_data)

    assert [note.note_id for note in notes] == ["10376710029", "10636556989"]
    assert notes[0].note_url == "https://tieba.baidu.com/p/10376710029"
    assert notes[0].tieba_name == "加工中心吧"
    assert notes[0].tieba_link.endswith("kw=%E5%8A%A0%E5%B7%A5%E4%B8%AD%E5%BF%83")
