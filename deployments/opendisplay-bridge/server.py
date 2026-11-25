"""Small authenticated local HTTP companion for the upstream py-opendisplay SDK.

Only this process talks to BlueZ / BLE. Domus calls the narrow HTTP surface below;
it never receives direct Bluetooth privileges or arbitrary command execution.
"""
from __future__ import annotations

import asyncio
import base64
import hmac
import logging
import os
import re
from dataclasses import asdict
from io import BytesIO
from typing import Any

from aiohttp import web
from PIL import Image, UnidentifiedImageError
from opendisplay import FitMode, OpenDisplayDevice, RefreshMode, discover_devices_with_adv
from opendisplay.discovery_ip import discover_ip_devices

_LOG = logging.getLogger("opendisplay.bridge")
_MAC_RE = re.compile(r"^(?:[0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$")
_MAX_IMAGE_BYTES = 8 * 1024 * 1024
_MAX_BASE64_CHARS = ((_MAX_IMAGE_BYTES + 2) // 3) * 4
_ALLOWED_FITS = {"contain", "cover", "stretch", "crop"}
_ALLOWED_REFRESH = {"full", "fast"}


def _required_token() -> str:
    token = os.environ.get("OPENDISPLAY_BRIDGE_TOKEN", "").strip()
    if len(token) < 32:
        raise RuntimeError("OPENDISPLAY_BRIDGE_TOKEN must contain at least 32 characters")
    if "\r" in token or "\n" in token:
        raise RuntimeError("OPENDISPLAY_BRIDGE_TOKEN must not contain newlines")
    return token


def _authorised(request: web.Request) -> bool:
    expected = request.app["token"]
    supplied = request.headers.get("Authorization", "")
    if not supplied.startswith("Bearer "):
        return False
    token = supplied[7:]
    return hmac.compare_digest(token.encode("utf-8"), expected.encode("utf-8"))


@web.middleware
async def auth_middleware(request: web.Request, handler):
    if request.path == "/health":
        return await handler(request)
    if not _authorised(request):
        raise web.HTTPUnauthorized(text="missing or invalid bridge token")
    return await handler(request)


def _serialise_advertisement(adv: Any) -> dict[str, Any] | None:
    if adv is None:
        return None
    payload = asdict(adv)
    payload.pop("dynamic_data", None)
    payload.pop("raw_data", None)
    return payload


async def list_devices(request: web.Request) -> web.Response:
    timeout = float(request.app["ble_scan_seconds"])
    found: dict[str, dict[str, Any]] = {}
    try:
        ble_devices = await discover_devices_with_adv(timeout=timeout)
        for name, (address, advertisement) in ble_devices.items():
            mac = address.upper()
            found[mac] = {
                "name": name,
                "address": mac,
                "transport": "ble",
                "online": True,
                "advertisement": _serialise_advertisement(advertisement),
            }
    except Exception as exc:  # discovery errors should not prevent a LAN scan
        _LOG.warning("BLE discovery failed: %s", exc)

    if request.app["enable_wifi"]:
        try:
            lan_devices = await discover_ip_devices(scan_seconds=min(timeout, 5.0))
            for name, info in lan_devices.items():
                mac = (info.mac or "").upper()
                key = mac if _MAC_RE.fullmatch(mac) else f"lan:{info.host}:{info.port}"
                previous = found.get(key, {})
                found[key] = {
                    **previous,
                    "name": name,
                    "address": key,
                    "mac": mac or None,
                    "transport": "wifi",
                    "host": info.host,
                    "port": info.port,
                    "tls": info.tls,
                    "firmware": info.fw,
                    "online": True,
                }
        except Exception as exc:
            _LOG.warning("OpenDisplay mDNS discovery failed: %s", exc)

    devices = sorted(found.values(), key=lambda item: item["name"].lower())
    return web.json_response({"devices": devices})


def _decode_image(value: Any) -> Image.Image:
    if not isinstance(value, str) or len(value) > _MAX_BASE64_CHARS:
        raise web.HTTPRequestEntityTooLarge(max_size=_MAX_IMAGE_BYTES, actual_size=0)
    try:
        raw = base64.b64decode(value, validate=True)
    except (ValueError, TypeError) as exc:
        raise web.HTTPBadRequest(text="image_base64 must be valid Base64") from exc
    if not raw or len(raw) > _MAX_IMAGE_BYTES:
        raise web.HTTPRequestEntityTooLarge(max_size=_MAX_IMAGE_BYTES, actual_size=len(raw))
    try:
        with Image.open(BytesIO(raw)) as source:
            if source.format not in {"PNG", "JPEG", "WEBP", "BMP"}:
                raise web.HTTPUnsupportedMediaType(text="only PNG, JPEG, WEBP and BMP images are supported")
            source.verify()
        with Image.open(BytesIO(raw)) as source:
            source.load()
            return source.convert("RGB")
    except (UnidentifiedImageError, OSError, Image.DecompressionBombError) as exc:
        raise web.HTTPBadRequest(text="invalid or unsafe image") from exc


async def upload_image(request: web.Request) -> web.Response:
    try:
        payload = await request.json(loads=None)
    except Exception as exc:
        raise web.HTTPBadRequest(text="invalid JSON body") from exc

    transport = payload.get("transport", "ble")
    address = request.match_info["address"]
    fit = str(payload.get("fit", "contain")).lower()
    refresh_mode = str(payload.get("refresh_mode", "full")).lower()
    if fit not in _ALLOWED_FITS:
        raise web.HTTPBadRequest(text="unsupported fit mode")
    if refresh_mode not in _ALLOWED_REFRESH:
        raise web.HTTPBadRequest(text="unsupported refresh mode")

    image = _decode_image(payload.get("image_base64"))
    if transport == "ble":
        if not _MAC_RE.fullmatch(address):
            raise web.HTTPBadRequest(text="BLE address must be a MAC address")
        try:
            async with OpenDisplayDevice(mac_address=address.upper()) as device:
                await device.upload_image(
                    image,
                    fit=FitMode[fit.upper()],
                    refresh_mode=RefreshMode[refresh_mode.upper()],
                )
        except Exception as exc:
            _LOG.warning("BLE upload failed for %s: %s", address, exc)
            raise web.HTTPBadGateway(text="OpenDisplay BLE upload failed") from exc
    elif transport == "wifi":
        host = payload.get("host")
        port = payload.get("port")
        tls = payload.get("tls", False)
        if not isinstance(host, str) or not host or len(host) > 253:
            raise web.HTTPBadRequest(text="Wi-Fi host is required")
        if not isinstance(port, int) or not 1 <= port <= 65535:
            raise web.HTTPBadRequest(text="Wi-Fi port is invalid")
        if not isinstance(tls, bool):
            raise web.HTTPBadRequest(text="Wi-Fi TLS flag is invalid")
        # The SDK's LAN transport derives the device identity and uses the same
        # upload protocol. No arbitrary URL or remote image fetch is accepted.
        try:
            async with OpenDisplayDevice(host=host, port=port, tls=tls) as device:
                await device.upload_image(
                    image,
                    fit=FitMode[fit.upper()],
                    refresh_mode=RefreshMode[refresh_mode.upper()],
                )
        except Exception as exc:
            _LOG.warning("Wi-Fi upload failed for %s:%s: %s", host, port, exc)
            raise web.HTTPBadGateway(text="OpenDisplay Wi-Fi upload failed") from exc
    else:
        raise web.HTTPBadRequest(text="transport must be ble or wifi")

    return web.json_response({"ok": True, "address": address, "transport": transport})


async def health(_: web.Request) -> web.Response:
    return web.json_response({"status": "ok", "component": "opendisplay-bridge"})


def create_app() -> web.Application:
    token = _required_token()
    app = web.Application(middlewares=[auth_middleware], client_max_size=12 * 1024 * 1024)
    app["token"] = token
    app["ble_scan_seconds"] = max(2.0, min(float(os.environ.get("OPENDISPLAY_BRIDGE_BLE_SCAN_SECONDS", "6")), 20.0))
    app["enable_wifi"] = os.environ.get("OPENDISPLAY_BRIDGE_ENABLE_WIFI", "true").lower() in {"1", "true", "yes"}
    app.router.add_get("/health", health)
    app.router.add_get("/api/v1/devices", list_devices)
    app.router.add_post("/api/v1/devices/{address}/image", upload_image)
    return app


if __name__ == "__main__":
    logging.basicConfig(level=os.environ.get("LOG_LEVEL", "INFO").upper())
    host = os.environ.get("OPENDISPLAY_BRIDGE_BIND", "127.0.0.1")
    port = int(os.environ.get("OPENDISPLAY_BRIDGE_PORT", "10320"))
    web.run_app(create_app(), host=host, port=port, access_log=None)
