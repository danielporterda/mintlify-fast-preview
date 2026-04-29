#!/usr/bin/env python3
import os
import shutil
import socket
import subprocess
import sys
import tempfile
import time
from pathlib import Path
from urllib.request import urlopen

from playwright.sync_api import sync_playwright


REPO = Path(__file__).resolve().parents[2]


def free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


def wait_for_server(url: str) -> None:
    deadline = time.time() + 20
    last_error = None
    while time.time() < deadline:
        try:
            with urlopen(url, timeout=1) as response:
                if response.status == 200:
                    return
        except Exception as error:
            last_error = error
            time.sleep(0.2)
    raise RuntimeError(f"server did not become ready: {last_error}")


def chromium_path() -> str | None:
    configured = os.environ.get("MINTFAST_CHROMIUM")
    if configured:
        return configured
    for candidate in ("chromium", "google-chrome", "google-chrome-stable"):
        path = shutil.which(candidate)
        if path:
            return path
    return None


def main() -> int:
    with tempfile.TemporaryDirectory(prefix="mintfast-browser-") as tmp:
        root = Path(tmp) / "docs-main"
        shutil.copytree(REPO / "testdata" / "docs-main", root)
        port = free_port()
        base = f"http://127.0.0.1:{port}"
        server = subprocess.Popen(
            [
                "go",
                "run",
                "./cmd/mintfast",
                "dev",
                "--root",
                str(root),
                "--host",
                "127.0.0.1",
                "--port",
                str(port),
                "--no-open",
            ],
            cwd=REPO,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
        )
        try:
            wait_for_server(base + "/api-reference")
            executable = chromium_path()
            with sync_playwright() as playwright:
                launch_options = {"headless": True}
                if executable:
                    launch_options = {
                        "headless": False,
                        "executable_path": executable,
                        "args": ["--headless=new"],
                    }
                browser = playwright.chromium.launch(**launch_options)
                context = browser.new_context(viewport={"width": 1280, "height": 900})
                context.grant_permissions(["clipboard-read", "clipboard-write"], origin=base)
                page = context.new_page()

                response = page.goto(base + "/api-reference")
                assert response is not None and response.status == 200
                assert page.locator(".nav-tree").is_visible()

                page.locator(".mintfast-theme-toggle").click()
                page.wait_for_function("document.documentElement.dataset.theme === 'dark'")
                page.reload()
                page.wait_for_function("document.documentElement.dataset.theme === 'dark'")

                page.goto(base + "/reference/protobuf/operations/com-digitalasset-canton-admin-mediator-v30/mediatorstatusservice/mediatorstatus")
                page.locator(".mintfast-copy").first.click()
                page.locator(".mintfast-copy").first.wait_for(state="visible")
                page.wait_for_function("document.querySelector('.mintfast-copy').textContent === 'Copied'")

                mobile = browser.new_page(viewport={"width": 390, "height": 844})
                mobile.goto(base + "/reference/protobuf/operations/com-digitalasset-canton-admin-mediator-v30/mediatorstatusservice/mediatorstatus")
                assert mobile.locator(".nav").evaluate("el => getComputedStyle(el).display") == "none"
                assert mobile.locator(".toc").evaluate("el => getComputedStyle(el).display") == "none"

                page.goto(base + "/api-reference")
                api_page = root / "api-reference.mdx"
                time.sleep(0.5)
                api_page.write_text(api_page.read_text() + "\n\n## Browser Reload Smoke\n")
                page.get_by_role("heading", name="Browser Reload Smoke").wait_for(timeout=7000)

                browser.close()
        finally:
            server.terminate()
            try:
                server.wait(timeout=5)
            except subprocess.TimeoutExpired:
                server.kill()
                server.wait(timeout=5)
            if server.returncode not in (0, -15):
                output = server.stdout.read() if server.stdout else ""
                raise RuntimeError(f"server exited unexpectedly with {server.returncode}\n{output}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
