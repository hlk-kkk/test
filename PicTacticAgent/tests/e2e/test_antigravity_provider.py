"""Antigravity Provider 测试脚本.

测试 antigravity 反代是否可以正常生成图片：
1. Provider 类初始化和属性检查
2. 直接调用 OpenAI API 测试反代连通性
3. 通过 Provider 类生成图片
4. 验证响应格式和图片保存
"""

import asyncio
import os
import sys
from pathlib import Path

# Add backend to path
sys.path.insert(0, str(Path(__file__).parent.parent.parent / "backend" / "src"))

from dotenv import load_dotenv

load_dotenv()


def get_antigravity_config():
    """Get Antigravity API configuration from environment."""
    api_key = os.getenv("ANTIGRAVITY_API_KEY")
    base_url = os.getenv("ANTIGRAVITY_API_BASE_URL", "http://127.0.0.1:8045/v1")
    model = os.getenv("ANTIGRAVITY_MODEL", "gemini-3-pro-image")
    return api_key, base_url, model


def test_provider_class_properties():
    """Test AntigravityImageProvider class properties."""
    print("\n" + "=" * 60)
    print("Test 1: Provider 类属性检查")
    print("=" * 60)

    from pictactic.providers.antigravity import AntigravityImageProvider
    from pictactic.providers.base import GenerationMode

    provider = AntigravityImageProvider()

    print(f"   Provider name: {provider.name}")
    assert provider.name == "antigravity", "Provider name should be 'antigravity'"

    print(f"   Supports batch: {provider.supports_batch}")
    assert provider.supports_batch is True, "Should support batch generation"

    print(f"   Max concurrent: {provider.max_concurrent}")
    assert provider.max_concurrent > 0, "Max concurrent should be positive"

    print(f"   Supported modes: {[m.value for m in provider.supported_modes]}")
    assert GenerationMode.TEXT_TO_IMAGE in provider.supported_modes
    assert GenerationMode.IMAGE_TO_IMAGE not in provider.supported_modes

    print("\n   ✅ Provider 类属性检查通过!")
    return True


async def test_direct_openai_api():
    """Test direct OpenAI API call to verify reverse proxy connectivity."""
    print("\n" + "=" * 60)
    print("Test 2: 直接调用 OpenAI API 测试反代连通性")
    print("=" * 60)

    api_key, base_url, model = get_antigravity_config()

    print(f"   Base URL: {base_url}")
    print(f"   Model: {model}")
    print(f"   API Key: {api_key[:20] if api_key else 'NOT SET'}...")

    if not api_key:
        print("   ⚠️ SKIP: ANTIGRAVITY_API_KEY not set")
        return False

    try:
        from openai import OpenAI

        client = OpenAI(base_url=base_url, api_key=api_key)

        print("\n   正在调用 API...")
        response = await asyncio.to_thread(
            client.chat.completions.create,
            model=model,
            messages=[{"role": "user", "content": "Draw a simple red circle on white background"}],
            extra_body={"size": "1024x1024"},
        )

        content = response.choices[0].message.content
        print(f"   Response received, content length: {len(content) if content else 0} chars")

        if content:
            # Check response format
            if "![" in content and "data:image" in content:
                print("   Response format: Markdown with data URI")
            elif content.startswith("data:image"):
                print("   Response format: Data URI")
            elif len(content) > 1000:
                print("   Response format: Likely raw Base64")
            else:
                print(f"   Response preview: {content[:200]}...")

            print("\n   ✅ 反代连通性测试通过!")
            return True
        else:
            print("   ❌ API 返回空响应")
            return False

    except Exception as e:
        print(f"   ❌ 错误: {type(e).__name__}: {e}")
        return False


async def test_provider_generate():
    """Test image generation through AntigravityImageProvider."""
    print("\n" + "=" * 60)
    print("Test 3: 通过 Provider 类生成图片")
    print("=" * 60)

    api_key, _, _ = get_antigravity_config()
    if not api_key:
        print("   ⚠️ SKIP: ANTIGRAVITY_API_KEY not set")
        return False

    from pictactic.providers.antigravity import AntigravityImageProvider
    from pictactic.providers.base import GenerationParams

    provider = AntigravityImageProvider()

    params = GenerationParams(
        prompt="A minimalist logo design with geometric shapes, blue and white colors",
        aspect_ratio="1:1",
        size="1K",
        reference_images=[],  # No reference images for text-to-image
    )

    print(f"   Prompt: {params.prompt}")
    print(f"   Aspect ratio: {params.aspect_ratio}")
    print(f"   Mode: {params.get_mode().value}")

    try:
        print("\n   正在生成图片...")
        result = await provider.generate(params, index=0)

        print(f"   URL: {result.url}")
        print(f"   Local path: {result.local_path}")
        print(f"   Metadata: {result.metadata}")

        # Verify local file exists
        if result.local_path:
            local_file = Path(result.local_path)
            if local_file.exists():
                file_size = local_file.stat().st_size
                print(f"   File size: {file_size:,} bytes")
                print("\n   ✅ 图片生成成功!")
                return True
            else:
                print(f"   ❌ 本地文件不存在: {result.local_path}")
                return False
        else:
            print("   ⚠️ 未保存到本地（可能是非 local 存储模式）")
            return True

    except Exception as e:
        print(f"   ❌ 生成失败: {type(e).__name__}: {e}")
        import traceback
        traceback.print_exc()
        return False


async def test_provider_batch_generate():
    """Test batch image generation."""
    print("\n" + "=" * 60)
    print("Test 4: 批量生成测试")
    print("=" * 60)

    api_key, _, _ = get_antigravity_config()
    if not api_key:
        print("   ⚠️ SKIP: ANTIGRAVITY_API_KEY not set")
        return False

    from pictactic.providers.antigravity import AntigravityImageProvider
    from pictactic.providers.base import GenerationParams

    provider = AntigravityImageProvider()

    params = GenerationParams(
        prompt="Abstract art with flowing lines and gradient colors",
        aspect_ratio="16:9",
        size="1K",
        reference_images=[],
    )

    num_images = 2
    print(f"   Generating {num_images} images...")

    completed = []

    async def on_ready(idx, result):
        completed.append(idx)
        print(f"   Image {idx + 1}/{num_images} ready: {result.url}")

    try:
        results = await provider.generate_batch_stream(
            params, num_images, on_image_ready=on_ready
        )

        print(f"\n   Generated {len(results)} images")

        for i, result in enumerate(results):
            if result.local_path and Path(result.local_path).exists():
                print(f"   [{i}] ✅ {result.local_path}")
            else:
                print(f"   [{i}] URL: {result.url}")

        print("\n   ✅ 批量生成测试通过!")
        return True

    except Exception as e:
        print(f"   ❌ 批量生成失败: {type(e).__name__}: {e}")
        return False


async def test_mode_validation():
    """Test that IMAGE_TO_IMAGE mode is rejected."""
    print("\n" + "=" * 60)
    print("Test 5: 模式验证测试（IMAGE_TO_IMAGE 应被拒绝）")
    print("=" * 60)

    from pictactic.providers.antigravity import AntigravityImageProvider
    from pictactic.providers.base import GenerationParams
    from pictactic.core.exceptions import ProviderError

    provider = AntigravityImageProvider()

    # Create params with reference images (triggers IMAGE_TO_IMAGE mode)
    params = GenerationParams(
        prompt="Test prompt",
        aspect_ratio="1:1",
        size="1K",
        reference_images=["https://example.com/image.jpg"],  # This triggers IMAGE_TO_IMAGE
    )

    print(f"   Mode detected: {params.get_mode().value}")
    print(f"   Provider supported modes: {[m.value for m in provider.supported_modes]}")

    try:
        # This should raise ProviderError
        params.validate_mode(provider.supported_modes)
        print("   ❌ 应该抛出 ProviderError 但没有!")
        return False
    except ProviderError as e:
        print(f"   ✅ 正确抛出 ProviderError: {e.message}")
        return True
    except Exception as e:
        print(f"   ❌ 意外错误: {type(e).__name__}: {e}")
        return False


async def run_all_tests():
    """Run all Antigravity provider tests."""
    print("\n" + "=" * 60)
    print("PicTactic Antigravity Provider 测试")
    print("=" * 60)

    api_key, base_url, model = get_antigravity_config()
    print(f"\n配置信息:")
    print(f"  Base URL: {base_url}")
    print(f"  Model: {model}")
    print(f"  API Key: {'已配置' if api_key else '未配置'}")

    results = {}

    # Test 1: Provider class properties (no API call)
    results["properties"] = test_provider_class_properties()

    # Test 5: Mode validation (no API call)
    results["mode_validation"] = await test_mode_validation()

    # API tests
    print("\n" + "-" * 60)
    print("API 测试（需要运行反代服务）")
    print("-" * 60)

    if not api_key:
        print("\n⚠️ 跳过 API 测试 - ANTIGRAVITY_API_KEY 未设置")
    else:
        # Test 2: Direct OpenAI API call
        results["direct_api"] = await test_direct_openai_api()

        if results.get("direct_api"):
            # Test 3: Provider generate
            results["provider_generate"] = await test_provider_generate()

            # Test 4: Batch generate (only if single generate works)
            if results.get("provider_generate"):
                results["batch_generate"] = await test_provider_batch_generate()

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
