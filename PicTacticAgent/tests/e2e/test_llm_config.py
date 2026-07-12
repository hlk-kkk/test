"""LLM 配置测试脚本.

测试 create_vision_llm 工厂函数是否能根据 DEFAULT_PROVIDER 正确创建 LLM 实例：
1. 测试 jiekouai provider 的 LLM 配置
2. 测试 antigravity provider 的 LLM 配置
3. 测试 LLM 实际调用能力
"""

import asyncio
import os
import sys
from pathlib import Path

# Add backend to path
sys.path.insert(0, str(Path(__file__).parent.parent.parent / "backend" / "src"))

from dotenv import load_dotenv

load_dotenv()


def reload_config():
    """Reload config module to pick up environment changes."""
    import importlib
    from pictactic.core import config
    config.get_settings.cache_clear()
    importlib.reload(config)
    return config


def test_llm_config_gemini():
    """Test LLM config for gemini provider."""
    print("\n" + "=" * 60)
    print("Test: Gemini 官方 LLM 配置检查")
    print("=" * 60)

    # Temporarily override default_provider
    original_provider = os.environ.get("DEFAULT_PROVIDER")
    os.environ["DEFAULT_PROVIDER"] = "gemini"

    # Reload config module
    config = reload_config()

    print(f"   default_provider: {config.settings.default_provider}")

    llm_config = config.get_llm_config()
    print(f"   LLM config:")
    print(f"      model: {llm_config['model']}")
    print(f"      model_provider: {llm_config['model_provider']}")
    print(f"      base_url: {llm_config['base_url']}")
    print(f"      api_key: {'已配置' if llm_config['api_key'] else '未配置'}")

    # Verify config
    assert llm_config["model_provider"] == "google_genai", "Gemini should use google_genai provider"
    assert llm_config["base_url"] is None, "Gemini should not have base_url"
    assert llm_config["model"] == config.settings.gemini_llm_model

    # Restore original provider
    if original_provider:
        os.environ["DEFAULT_PROVIDER"] = original_provider
    else:
        os.environ.pop("DEFAULT_PROVIDER", None)
    reload_config()

    print("\n   ✅ Gemini 官方 LLM 配置检查通过!")
    return True


def test_llm_config_jiekouai():
    """Test LLM config for jiekouai provider."""
    print("\n" + "=" * 60)
    print("Test 1: Jiekouai LLM 配置检查")
    print("=" * 60)

    # Temporarily override default_provider
    original_provider = os.environ.get("DEFAULT_PROVIDER")
    os.environ["DEFAULT_PROVIDER"] = "jiekouai"

    # Reload config module
    config = reload_config()

    print(f"   default_provider: {config.settings.default_provider}")

    llm_config = config.get_llm_config()
    print(f"   LLM config:")
    print(f"      model: {llm_config['model']}")
    print(f"      model_provider: {llm_config['model_provider']}")
    print(f"      base_url: {llm_config['base_url']}")
    print(f"      api_key: {'已配置' if llm_config['api_key'] else '未配置'}")

    # Verify config
    assert llm_config["model_provider"] == "openai", "Jiekouai should use openai provider"
    assert llm_config["base_url"] == config.settings.jiekouai_llm_base_url
    assert llm_config["model"] == config.settings.jiekouai_llm_model

    # Restore original provider
    if original_provider:
        os.environ["DEFAULT_PROVIDER"] = original_provider
    else:
        os.environ.pop("DEFAULT_PROVIDER", None)
    reload_config()

    print("\n   ✅ Jiekouai LLM 配置检查通过!")
    return True


def test_llm_config_antigravity():
    """Test LLM config for antigravity provider."""
    print("\n" + "=" * 60)
    print("Test 2: Antigravity LLM 配置检查")
    print("=" * 60)

    # Temporarily override default_provider
    original_provider = os.environ.get("DEFAULT_PROVIDER")
    os.environ["DEFAULT_PROVIDER"] = "antigravity"

    # Reload config module
    config = reload_config()

    print(f"   default_provider: {config.settings.default_provider}")

    llm_config = config.get_llm_config()
    print(f"   LLM config:")
    print(f"      model: {llm_config['model']}")
    print(f"      model_provider: {llm_config['model_provider']}")
    print(f"      base_url: {llm_config['base_url']}")
    print(f"      api_key: {'已配置' if llm_config['api_key'] else '未配置'}")

    # Verify config
    assert llm_config["model_provider"] == "openai", "Antigravity should use openai provider"
    assert llm_config["base_url"] == config.settings.antigravity_api_base_url
    assert llm_config["model"] == config.settings.antigravity_llm_model

    # Restore original provider
    if original_provider:
        os.environ["DEFAULT_PROVIDER"] = original_provider
    else:
        os.environ.pop("DEFAULT_PROVIDER", None)
    reload_config()

    print("\n   ✅ Antigravity LLM 配置检查通过!")
    return True


async def test_gemini_llm_call():
    """Test actual LLM call with gemini provider."""
    print("\n" + "=" * 60)
    print("Test: Gemini 官方 LLM 实际调用测试")
    print("=" * 60)

    api_key = os.getenv("GEMINI_API_KEY")
    if not api_key:
        print("   ⚠️ SKIP: GEMINI_API_KEY not set")
        return None

    # Temporarily override default_provider
    original_provider = os.environ.get("DEFAULT_PROVIDER")
    os.environ["DEFAULT_PROVIDER"] = "gemini"

    # Reload config module
    config = reload_config()

    try:
        print("   正在创建 LLM 实例...")
        llm = config.create_vision_llm(temperature=0.3, max_retries=2)

        print(f"   LLM 类型: {type(llm).__name__}")

        print("   正在调用 LLM...")
        from langchain_core.messages import HumanMessage

        response = await llm.ainvoke([
            HumanMessage(content="请用一句话描述一张美丽的日落图片。")
        ])

        print(f"   响应: {response.content[:200]}...")
        print("\n   ✅ Gemini 官方 LLM 调用成功!")
        result = True

    except Exception as e:
        print(f"   ❌ 错误: {type(e).__name__}: {e}")
        result = False

    # Restore original provider
    if original_provider:
        os.environ["DEFAULT_PROVIDER"] = original_provider
    else:
        os.environ.pop("DEFAULT_PROVIDER", None)
    reload_config()

    return result


async def test_gemini_vision_llm():
    """Test vision LLM call with gemini provider using an image."""
    print("\n" + "=" * 60)
    print("Test: Gemini 官方 Vision LLM 图片分析测试")
    print("=" * 60)

    api_key = os.getenv("GEMINI_API_KEY")
    if not api_key:
        print("   ⚠️ SKIP: GEMINI_API_KEY not set")
        return None

    # Temporarily override default_provider
    original_provider = os.environ.get("DEFAULT_PROVIDER")
    os.environ["DEFAULT_PROVIDER"] = "gemini"

    # Reload config module
    config = reload_config()

    # Test image URL
    test_image_url = "https://picsum.photos/400/300"

    try:
        print("   正在创建 Vision LLM 实例...")
        llm = config.create_vision_llm(temperature=0.3, max_retries=2)

        print(f"   测试图片: {test_image_url}")
        print("   正在分析图片...")

        from langchain_core.messages import HumanMessage

        message_content = [
            {"type": "text", "text": "请用中文简要描述这张图片的内容和风格。"},
            {"type": "image_url", "image_url": {"url": test_image_url}},
        ]

        response = await llm.ainvoke([HumanMessage(content=message_content)])

        print(f"   响应: {response.content[:300]}...")
        print("\n   ✅ Gemini 官方 Vision LLM 图片分析成功!")
        result = True

    except Exception as e:
        print(f"   ❌ 错误: {type(e).__name__}: {e}")
        result = False

    # Restore original provider
    if original_provider:
        os.environ["DEFAULT_PROVIDER"] = original_provider
    else:
        os.environ.pop("DEFAULT_PROVIDER", None)
    reload_config()

    return result


async def test_jiekouai_llm_call():
    """Test actual LLM call with jiekouai provider."""
    print("\n" + "=" * 60)
    print("Test 3: Jiekouai LLM 实际调用测试")
    print("=" * 60)

    api_key = os.getenv("JIEKOUAI_API_KEY")
    if not api_key:
        print("   ⚠️ SKIP: JIEKOUAI_API_KEY not set")
        return None

    # Temporarily override default_provider
    original_provider = os.environ.get("DEFAULT_PROVIDER")
    os.environ["DEFAULT_PROVIDER"] = "jiekouai"

    # Reload config module
    config = reload_config()

    try:
        print("   正在创建 LLM 实例...")
        llm = config.create_vision_llm(temperature=0.3, max_retries=2)

        print(f"   LLM 类型: {type(llm).__name__}")

        print("   正在调用 LLM...")
        from langchain_core.messages import HumanMessage

        response = await llm.ainvoke([
            HumanMessage(content="请用一句话描述一张美丽的日落图片。")
        ])

        print(f"   响应: {response.content[:200]}...")
        print("\n   ✅ Jiekouai LLM 调用成功!")
        result = True

    except Exception as e:
        print(f"   ❌ 错误: {type(e).__name__}: {e}")
        result = False

    # Restore original provider
    if original_provider:
        os.environ["DEFAULT_PROVIDER"] = original_provider
    else:
        os.environ.pop("DEFAULT_PROVIDER", None)
    reload_config()

    return result


async def test_antigravity_llm_call():
    """Test actual LLM call with antigravity provider."""
    print("\n" + "=" * 60)
    print("Test 4: Antigravity LLM 实际调用测试")
    print("=" * 60)

    api_key = os.getenv("ANTIGRAVITY_API_KEY")
    if not api_key:
        print("   ⚠️ SKIP: ANTIGRAVITY_API_KEY not set")
        return None

    # Temporarily override default_provider
    original_provider = os.environ.get("DEFAULT_PROVIDER")
    os.environ["DEFAULT_PROVIDER"] = "antigravity"

    # Reload config module
    config = reload_config()

    try:
        print("   正在创建 LLM 实例...")
        llm = config.create_vision_llm(temperature=0.3, max_retries=2)

        print(f"   LLM 类型: {type(llm).__name__}")

        print("   正在调用 LLM...")
        from langchain_core.messages import HumanMessage

        response = await llm.ainvoke([
            HumanMessage(content="请用一句话描述一张美丽的日落图片。")
        ])

        print(f"   响应: {response.content[:200]}...")
        print("\n   ✅ Antigravity LLM 调用成功!")
        result = True

    except Exception as e:
        print(f"   ❌ 错误: {type(e).__name__}: {e}")
        result = False

    # Restore original provider
    if original_provider:
        os.environ["DEFAULT_PROVIDER"] = original_provider
    else:
        os.environ.pop("DEFAULT_PROVIDER", None)
    reload_config()

    return result


async def test_jiekouai_vision_llm():
    """Test vision LLM call with jiekouai provider using an image."""
    print("\n" + "=" * 60)
    print("Test 5: Jiekouai Vision LLM 图片分析测试")
    print("=" * 60)

    api_key = os.getenv("JIEKOUAI_API_KEY")
    if not api_key:
        print("   ⚠️ SKIP: JIEKOUAI_API_KEY not set")
        return None

    # Temporarily override default_provider
    original_provider = os.environ.get("DEFAULT_PROVIDER")
    os.environ["DEFAULT_PROVIDER"] = "jiekouai"

    # Reload config module
    config = reload_config()

    # Test image URL
    test_image_url = "https://picsum.photos/400/300"

    try:
        print("   正在创建 Vision LLM 实例...")
        llm = config.create_vision_llm(temperature=0.3, max_retries=2)

        print(f"   测试图片: {test_image_url}")
        print("   正在分析图片...")

        from langchain_core.messages import HumanMessage

        message_content = [
            {"type": "text", "text": "请用中文简要描述这张图片的内容和风格。"},
            {"type": "image_url", "image_url": {"url": test_image_url}},
        ]

        response = await llm.ainvoke([HumanMessage(content=message_content)])

        print(f"   响应: {response.content[:300]}...")
        print("\n   ✅ Jiekouai Vision LLM 图片分析成功!")
        result = True

    except Exception as e:
        print(f"   ❌ 错误: {type(e).__name__}: {e}")
        result = False

    # Restore original provider
    if original_provider:
        os.environ["DEFAULT_PROVIDER"] = original_provider
    else:
        os.environ.pop("DEFAULT_PROVIDER", None)
    reload_config()

    return result


async def test_antigravity_vision_llm():
    """Test vision LLM call with antigravity provider using an image."""
    print("\n" + "=" * 60)
    print("Test 6: Antigravity Vision LLM 图片分析测试")
    print("=" * 60)

    api_key = os.getenv("ANTIGRAVITY_API_KEY")
    if not api_key:
        print("   ⚠️ SKIP: ANTIGRAVITY_API_KEY not set")
        return None

    # Temporarily override default_provider
    original_provider = os.environ.get("DEFAULT_PROVIDER")
    os.environ["DEFAULT_PROVIDER"] = "antigravity"

    # Reload config module
    config = reload_config()

    # Test image URL
    test_image_url = "https://picsum.photos/400/300"

    try:
        print("   正在创建 Vision LLM 实例...")
        llm = config.create_vision_llm(temperature=0.3, max_retries=2)

        print(f"   测试图片: {test_image_url}")
        print("   正在分析图片...")

        from langchain_core.messages import HumanMessage

        message_content = [
            {"type": "text", "text": "请用中文简要描述这张图片的内容和风格。"},
            {"type": "image_url", "image_url": {"url": test_image_url}},
        ]

        response = await llm.ainvoke([HumanMessage(content=message_content)])

        print(f"   响应: {response.content[:300]}...")
        print("\n   ✅ Antigravity Vision LLM 图片分析成功!")
        result = True

    except Exception as e:
        print(f"   ❌ 错误: {type(e).__name__}: {e}")
        result = False

    # Restore original provider
    if original_provider:
        os.environ["DEFAULT_PROVIDER"] = original_provider
    else:
        os.environ.pop("DEFAULT_PROVIDER", None)
    reload_config()

    return result


async def run_all_tests():
    """Run all LLM config tests."""
    print("\n" + "=" * 60)
    print("PicTactic LLM 配置测试")
    print("=" * 60)

    gemini_key = os.getenv("GEMINI_API_KEY")
    jiekouai_key = os.getenv("JIEKOUAI_API_KEY")
    antigravity_key = os.getenv("ANTIGRAVITY_API_KEY")

    print(f"\n配置信息:")
    print(f"  GEMINI_API_KEY: {'已配置' if gemini_key else '未配置'}")
    print(f"  JIEKOUAI_API_KEY: {'已配置' if jiekouai_key else '未配置'}")
    print(f"  ANTIGRAVITY_API_KEY: {'已配置' if antigravity_key else '未配置'}")

    results = {}

    # Config tests (no API call)
    results["gemini_config"] = test_llm_config_gemini()
    results["jiekouai_config"] = test_llm_config_jiekouai()
    results["antigravity_config"] = test_llm_config_antigravity()

    # API tests
    print("\n" + "-" * 60)
    print("API 测试")
    print("-" * 60)

    # Gemini tests
    result = await test_gemini_llm_call()
    if result is not None:
        results["gemini_llm_call"] = result

    result = await test_gemini_vision_llm()
    if result is not None:
        results["gemini_vision_llm"] = result

    # Jiekouai tests
    result = await test_jiekouai_llm_call()
    if result is not None:
        results["jiekouai_llm_call"] = result

    result = await test_jiekouai_vision_llm()
    if result is not None:
        results["jiekouai_vision_llm"] = result

    # Antigravity tests
    result = await test_antigravity_llm_call()
    if result is not None:
        results["antigravity_llm_call"] = result

    result = await test_antigravity_vision_llm()
    if result is not None:
        results["antigravity_vision_llm"] = result

    # Summary
    print("\n" + "=" * 60)
    print("测试结果汇总")
    print("=" * 60)

    for test_name, passed in results.items():
        status = "✅ 通过" if passed else "❌ 失败"
        print(f"  {test_name}: {status}")

    all_passed = all(results.values())
    print("\n" + ("✅ 所有测试通过!" if all_passed else "❌ 部分测试失败"))

    return all_passed


if __name__ == "__main__":
    success = asyncio.run(run_all_tests())
    sys.exit(0 if success else 1)
