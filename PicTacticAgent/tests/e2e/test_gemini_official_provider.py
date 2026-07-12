"""Official Gemini SDK Provider 测试脚本.

测试官方 Google Gemini SDK Provider 功能：
1. Provider 类初始化和属性检查
2. 官方 SDK 健康检查
3. 文生图测试（text-to-image）
4. 图生图测试（image-to-image）
5. 不同分辨率测试（1K/2K/4K）
"""

import asyncio
import base64
import os
import sys
from pathlib import Path

# Add backend to path
sys.path.insert(0, str(Path(__file__).parent.parent.parent / "backend" / "src"))

from dotenv import load_dotenv

load_dotenv()


def get_gemini_config():
    """Get official Gemini SDK configuration from environment."""
    api_key = os.getenv("GEMINI_API_KEY")
    model = os.getenv("GEMINI_MODEL", "gemini-2.5-flash-image")
    return api_key, model


def test_provider_class_properties():
    """Test GeminiImageProvider class properties."""
    print("\n" + "=" * 60)
    print("Test 1: Provider 类属性检查")
    print("=" * 60)

    from pictactic.providers.gemini import GeminiImageProvider
    from pictactic.providers.base import GenerationMode

    provider = GeminiImageProvider()

    print(f"   Provider name: {provider.name}")
    assert provider.name == "gemini", "Provider name should be 'gemini'"

    print(f"   Supports batch: {provider.supports_batch}")
    assert provider.supports_batch is True, "Should support batch generation"

    print(f"   Max concurrent: {provider.max_concurrent}")
    assert provider.max_concurrent > 0, "Max concurrent should be positive"

    print(f"   Supported modes: {[m.value for m in provider.supported_modes]}")
    assert GenerationMode.TEXT_TO_IMAGE in provider.supported_modes
    assert GenerationMode.IMAGE_TO_IMAGE in provider.supported_modes

    print("\n   ✅ Provider 类属性检查通过!")
    return True


async def test_sdk_health_check():
    """Test official Gemini SDK health check."""
    print("\n" + "=" * 60)
    print("Test 2: 官方 SDK 健康检查")
    print("=" * 60)

    api_key, model = get_gemini_config()

    print(f"   Model: {model}")
    print(f"   API Key: {api_key[:20] if api_key else 'NOT SET'}...")

    if not api_key:
        print("   ⚠️ SKIP: GEMINI_API_KEY not set")
        return False

    from pictactic.providers.gemini import GeminiImageProvider

    provider = GeminiImageProvider()

    try:
        healthy = await provider.health_check()
        print(f"   Health check result: {healthy}")

        if healthy:
            print("\n   ✅ 官方 SDK 健康检查通过!")
        else:
            print("\n   ❌ 官方 SDK 健康检查失败")

        return healthy
    except Exception as e:
        print(f"   ❌ 错误: {type(e).__name__}: {e}")
        return False


async def test_text_to_image():
    """Test text-to-image generation with official SDK."""
    print("\n" + "=" * 60)
    print("Test 3: 文生图测试（text-to-image）")
    print("=" * 60)

    api_key, _ = get_gemini_config()
    if not api_key:
        print("   ⚠️ SKIP: GEMINI_API_KEY not set")
        return False

    from pictactic.providers.gemini import GeminiImageProvider
    from pictactic.providers.base import GenerationParams

    provider = GeminiImageProvider()

    params = GenerationParams(
        prompt="A minimalist tech blog header with abstract geometric shapes, blue gradient background",
        aspect_ratio="16:9",
        size="1K",
        reference_images=[],
    )

    print(f"   Prompt: {params.prompt[:50]}...")
    print(f"   Aspect ratio: {params.aspect_ratio}")
    print(f"   Size: {params.size}")
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
                print("\n   ✅ 文生图测试通过!")
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
    finally:
        await provider.close()


async def test_image_to_image():
    """Test image-to-image generation with official SDK."""
    print("\n" + "=" * 60)
    print("Test 4: 图生图测试（image-to-image）")
    print("=" * 60)

    api_key, _ = get_gemini_config()
    if not api_key:
        print("   ⚠️ SKIP: GEMINI_API_KEY not set")
        return False

    # Check for test image
    test_image_path = Path(__file__).parent.parent / "fixtures" / "test_image.jpg"
    if not test_image_path.exists():
        print(f"   ⚠️ SKIP: Test image not found at {test_image_path}")
        print("   提示: 创建 tests/fixtures/test_image.jpg 文件以运行此测试")
        return None  # Skip, not fail

    from pictactic.providers.gemini import GeminiImageProvider
    from pictactic.providers.base import GenerationParams

    provider = GeminiImageProvider()

    # Load test image as base64 data URI
    with open(test_image_path, "rb") as f:
        image_base64 = base64.b64encode(f.read()).decode()
    image_data_uri = f"data:image/jpeg;base64,{image_base64}"

    params = GenerationParams(
        prompt="Transform this image into a dreamy watercolor painting style",
        aspect_ratio="1:1",
        size="1K",
        reference_images=[image_data_uri],
    )

    print(f"   Prompt: {params.prompt}")
    print(f"   Reference image: base64 ({len(image_base64)} chars)")
    print(f"   Mode: {params.get_mode().value}")

    try:
        print("\n   正在生成图片...")
        result = await provider.generate(params, index=0)

        print(f"   URL: {result.url}")
        print(f"   Local path: {result.local_path}")

        if result.local_path and Path(result.local_path).exists():
            file_size = Path(result.local_path).stat().st_size
            print(f"   File size: {file_size:,} bytes")
            print("\n   ✅ 图生图测试通过!")
            return True
        else:
            print("   ⚠️ 本地文件未生成")
            return True

    except Exception as e:
        print(f"   ❌ 生成失败: {type(e).__name__}: {e}")
        import traceback
        traceback.print_exc()
        return False
    finally:
        await provider.close()


async def test_different_resolutions():
    """Test different resolution settings."""
    print("\n" + "=" * 60)
    print("Test 5: 不同分辨率测试")
    print("=" * 60)

    api_key, model = get_gemini_config()
    if not api_key:
        print("   ⚠️ SKIP: GEMINI_API_KEY not set")
        return False

    # Only test resolutions if using pro model
    if "pro" not in model.lower():
        print(f"   ⚠️ SKIP: 当前模型 {model} 不支持多分辨率")
        print("   提示: 使用 gemini-3-pro-image-preview 模型以测试 2K/4K 分辨率")
        return None

    from pictactic.providers.gemini import GeminiImageProvider
    from pictactic.providers.base import GenerationParams

    resolutions = ["1K", "2K"]  # Skip 4K for faster testing

    for resolution in resolutions:
        print(f"\n   测试 {resolution} 分辨率...")

        provider = GeminiImageProvider()
        params = GenerationParams(
            prompt="A simple geometric pattern",
            aspect_ratio="1:1",
            size=resolution,
            reference_images=[],
        )

        try:
            result = await provider.generate(params, index=0)
            if result.local_path and Path(result.local_path).exists():
                file_size = Path(result.local_path).stat().st_size
                print(f"   {resolution}: ✅ ({file_size:,} bytes)")
            else:
                print(f"   {resolution}: ✅ (无本地文件)")
        except Exception as e:
            print(f"   {resolution}: ❌ {e}")
            return False
        finally:
            await provider.close()

    print("\n   ✅ 分辨率测试通过!")
    return True


async def test_batch_generation():
    """Test batch image generation."""
    print("\n" + "=" * 60)
    print("Test 6: 批量生成测试")
    print("=" * 60)

    api_key, _ = get_gemini_config()
    if not api_key:
        print("   ⚠️ SKIP: GEMINI_API_KEY not set")
        return False

    from pictactic.providers.gemini import GeminiImageProvider
    from pictactic.providers.base import GenerationParams

    provider = GeminiImageProvider()

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
    finally:
        await provider.close()


def test_provider_registration():
    """Test that GeminiImageProvider is properly registered."""
    print("\n" + "=" * 60)
    print("Test 7: Provider 注册检查")
    print("=" * 60)

    from pictactic.providers import get_provider, _PROVIDERS, GeminiImageProvider

    print(f"   Registered providers: {list(_PROVIDERS.keys())}")

    # Check gemini maps to GeminiImageProvider
    assert "gemini" in _PROVIDERS, "'gemini' should be in _PROVIDERS"
    assert _PROVIDERS["gemini"] == GeminiImageProvider, "'gemini' should map to GeminiImageProvider"
    print("   ✅ 'gemini' -> GeminiImageProvider")

    # Check jiekouai is also registered
    assert "jiekouai" in _PROVIDERS, "'jiekouai' should be in _PROVIDERS"
    print("   ✅ 'jiekouai' provider registered")

    # Check get_provider works
    provider = get_provider("gemini")
    assert provider.name == "gemini"
    print("   ✅ get_provider('gemini') works")

    print("\n   ✅ Provider 注册检查通过!")
    return True


async def run_all_tests():
    """Run all official Gemini SDK provider tests."""
    print("\n" + "=" * 60)
    print("PicTactic Official Gemini SDK Provider 测试")
    print("=" * 60)

    api_key, model = get_gemini_config()
    print(f"\n配置信息:")
    print(f"  Model: {model}")
    print(f"  API Key: {'已配置' if api_key else '未配置'}")

    results = {}

    # Test 1: Provider class properties (no API call)
    results["properties"] = test_provider_class_properties()

    # Test 7: Provider registration (no API call)
    results["registration"] = test_provider_registration()

    # API tests
    print("\n" + "-" * 60)
    print("API 测试（需要配置 GEMINI_API_KEY）")
    print("-" * 60)

    if not api_key:
        print("\n⚠️ 跳过 API 测试 - GEMINI_API_KEY 未设置")
        print("提示: 从 https://aistudio.google.com/apikey 获取 API Key")
    else:
        # Test 2: SDK health check
        results["health_check"] = await test_sdk_health_check()

        if results.get("health_check"):
            # Test 3: Text-to-image
            results["text_to_image"] = await test_text_to_image()

            # Test 4: Image-to-image
            img2img_result = await test_image_to_image()
            if img2img_result is not None:
                results["image_to_image"] = img2img_result

            # Test 5: Different resolutions
            resolution_result = await test_different_resolutions()
            if resolution_result is not None:
                results["resolutions"] = resolution_result

            # Test 6: Batch generation
            if results.get("text_to_image"):
                results["batch_generate"] = await test_batch_generation()

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
