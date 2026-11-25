"""Focused tests for the OpenDisplay companion's security boundary."""
import base64
import os
import unittest
from unittest.mock import AsyncMock, patch

from aiohttp.test_utils import AioHTTPTestCase, unittest_run_loop
from PIL import Image
from io import BytesIO

os.environ.setdefault("OPENDISPLAY_BRIDGE_TOKEN", "test-token-" + "x" * 40)

import server


def png_b64() -> str:
    out = BytesIO()
    Image.new("RGB", (16, 16), "white").save(out, format="PNG")
    return base64.b64encode(out.getvalue()).decode("ascii")


class BridgeTests(AioHTTPTestCase):
    async def get_application(self):
        return server.create_app()

    @unittest_run_loop
    async def test_health_is_public(self):
        response = await self.client.request("GET", "/health")
        self.assertEqual(response.status, 200)

    @unittest_run_loop
    async def test_api_requires_bearer_token(self):
        response = await self.client.request("GET", "/api/v1/devices")
        self.assertEqual(response.status, 401)

    @unittest_run_loop
    async def test_discovery_returns_empty_list_when_no_devices(self):
        with patch.object(server, "discover_devices_with_adv", new=AsyncMock(return_value={})):
            with patch.object(server, "discover_ip_devices", new=AsyncMock(return_value={})):
                response = await self.client.request(
                    "GET", "/api/v1/devices", headers={"Authorization": "Bearer " + os.environ["OPENDISPLAY_BRIDGE_TOKEN"]}
                )
        self.assertEqual(response.status, 200)
        self.assertEqual((await response.json())["devices"], [])

    @unittest_run_loop
    async def test_image_rejects_unknown_image_type(self):
        response = await self.client.request(
            "POST",
            "/api/v1/devices/AA:BB:CC:DD:EE:FF/image",
            json={"image_base64": base64.b64encode(b"not-image").decode("ascii")},
            headers={"Authorization": "Bearer " + os.environ["OPENDISPLAY_BRIDGE_TOKEN"]},
        )
        self.assertEqual(response.status, 400)

    @unittest_run_loop
    async def test_image_rejects_invalid_transport(self):
        response = await self.client.request(
            "POST",
            "/api/v1/devices/AA:BB:CC:DD:EE:FF/image",
            json={"transport": "ssh", "image_base64": png_b64()},
            headers={"Authorization": "Bearer " + os.environ["OPENDISPLAY_BRIDGE_TOKEN"]},
        )
        self.assertEqual(response.status, 400)



    @unittest_run_loop
    async def test_json_scalar_is_rejected(self):
        response = await self.client.request(
            "POST",
            "/api/v1/devices/AA:BB:CC:DD:EE:FF/image",
            data='"not-an-object"',
            headers={
                "Authorization": "Bearer " + os.environ["OPENDISPLAY_BRIDGE_TOKEN"],
                "Content-Type": "application/json",
            },
        )
        self.assertEqual(response.status, 400)

    @unittest_run_loop
    async def test_tls_psk_is_rejected_until_device_credentials_exist(self):
        response = await self.client.request(
            "POST",
            "/api/v1/devices/AA:BB:CC:DD:EE:FF/image",
            json={
                "transport": "wifi",
                "host": "192.0.2.10",
                "port": 2446,
                "tls": True,
                "image_base64": png_b64(),
            },
            headers={"Authorization": "Bearer " + os.environ["OPENDISPLAY_BRIDGE_TOKEN"]},
        )
        self.assertEqual(response.status, 501)


    @unittest_run_loop
    async def test_json_scalar_is_rejected(self):
        response = await self.client.request(
            "POST",
            "/api/v1/devices/AA:BB:CC:DD:EE:FF/image",
            data='"not-an-object"',
            headers={
                "Authorization": "Bearer " + os.environ["OPENDISPLAY_BRIDGE_TOKEN"],
                "Content-Type": "application/json",
            },
        )
        self.assertEqual(response.status, 400)

    @unittest_run_loop
    async def test_tls_psk_is_rejected_until_wifi_credentials_are_supported(self):
        response = await self.client.request(
            "POST",
            "/api/v1/devices/AA:BB:CC:DD:EE:FF/image",
            json={
                "transport": "wifi",
                "host": "192.0.2.10",
                "port": 2446,
                "tls": True,
                "image_base64": png_b64(),
            },
            headers={"Authorization": "Bearer " + os.environ["OPENDISPLAY_BRIDGE_TOKEN"]},
        )
        self.assertEqual(response.status, 501)

    @unittest_run_loop
    async def test_unknown_action_is_rejected(self):
        response = await self.client.request(
            "POST",
            "/api/v1/devices/AA:BB:CC:DD:EE:FF/actions/shell",
            json={},
            headers={"Authorization": "Bearer " + os.environ["OPENDISPLAY_BRIDGE_TOKEN"]},
        )
        self.assertEqual(response.status, 404)

    @unittest_run_loop
    async def test_action_requires_object_body(self):
        response = await self.client.request(
            "POST",
            "/api/v1/devices/AA:BB:CC:DD:EE:FF/actions/led",
            data='"not-an-object"',
            headers={
                "Authorization": "Bearer " + os.environ["OPENDISPLAY_BRIDGE_TOKEN"],
                "Content-Type": "application/json",
            },
        )
        self.assertEqual(response.status, 400)

    @unittest_run_loop
    async def test_drawcustom_rejects_remote_image_elements(self):
        from contextlib import asynccontextmanager

        @asynccontextmanager
        async def fake_open(*args, **kwargs):
            class Config:
                displays = []
            class Device:
                config = Config()
            yield Device()

        with patch.object(server, "_open_device", new=fake_open):
            response = await self.client.request(
                "POST",
                "/api/v1/devices/AA:BB:CC:DD:EE:FF/actions/drawcustom",
                json={"elements": [{"type": "dlimg", "url": "http://127.0.0.1:1/secret"}]},
                headers={"Authorization": "Bearer " + os.environ["OPENDISPLAY_BRIDGE_TOKEN"]},
            )
        self.assertEqual(response.status, 400)

    @unittest_run_loop
    async def test_invalid_device_key_map_is_rejected(self):
        with patch.dict(os.environ, {"OPENDISPLAY_BRIDGE_DEVICE_KEYS_JSON": '{"AA:BB:CC:DD:EE:FF":"short"}'}):
            with self.assertRaises(RuntimeError):
                server._load_device_keys()


if __name__ == "__main__":
    unittest.main()
