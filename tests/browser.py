"""Optional end-to-end UI test: pip install playwright; use a trusted browser.

The default target is the explicitly enabled loopback debug endpoint. Production
HTTPS/API is verified separately by smoke.py with its actual certificate chain.
The test creates two cards and leaves them in the selected database.
"""
import os
import uuid

from playwright.sync_api import expect, sync_playwright

BASE = os.environ.get("BROWSER_BASE_URL", "http://localhost:8080").rstrip("/")


def assert_mobile_layout(page):
    assert page.evaluate("document.documentElement.scrollWidth <= innerWidth"), "Page overflows phone viewport"
    for textarea in page.locator("textarea").all():
        assert float(textarea.evaluate("element => getComputedStyle(element).fontSize").removesuffix("px")) >= 16
    for button in page.locator("button, a.button").all():
        if button.is_visible():
            assert button.bounding_box()["height"] >= 44


with sync_playwright() as playwright:
    options = {"args": ["--no-sandbox"]}
    if os.environ.get("CHROMIUM_PATH"):
        options["executable_path"] = os.environ["CHROMIUM_PATH"]
    browser = playwright.chromium.launch(**options)
    page = browser.new_page(viewport={"width": 390, "height": 844})
    errors = []
    page.on("pageerror", lambda error: errors.append(str(error)))
    page.goto(BASE)
    expect(page.get_by_role("heading", level=1)).to_have_text("Все вместе")
    expect(page.locator("textarea, input")).to_have_count(0)
    page.locator("main").get_by_role("link", name="Заполнить", exact=True).click()
    expect(page).to_have_url(BASE + "/form")
    assert_mobile_layout(page)

    page.get_by_role("button", name="Отправить", exact=True).click()
    expect(page.locator("#request-error")).to_be_visible()
    expect(page.locator("#request")).to_be_focused()
    marker = "Нужна помощь «Вместе» " + str(uuid.uuid4())
    page.locator("#request").fill(marker)
    page.locator("#offer").fill("Могу помочь\n<script>window.injected=true</script>")
    page.locator("#details").fill("Казань, выходные")
    page.locator("#consideration").fill("Сроки уточняются")
    page.locator("#contact").fill("Анна, anna@example.org")

    # A failed request must leave entered data available for retry.
    page.route("**/api/cards", lambda route: route.fulfill(
        status=503, content_type="application/json", body='{"error":"Тест: сервер временно недоступен"}'
    ))
    page.get_by_role("button", name="Отправить", exact=True).click()
    expect(page.get_by_role("alert")).to_contain_text("Тест: сервер временно недоступен")
    expect(page.locator("#request")).to_have_value(marker)
    page.unroute("**/api/cards")
    page.get_by_role("button", name="Отправить", exact=True).click()
    page.wait_for_url("**/view&id=*")
    expect(page.locator(".card-table tbody tr")).to_have_count(5)
    expect(page.locator(".card-table")).to_contain_text(marker)
    assert page.evaluate("window.injected") is None
    saved_url = page.url
    page.reload()
    expect(page.locator(".card-table")).to_contain_text(marker)
    assert_mobile_layout(page)

    page.goto(BASE + "/admin")
    expect(page.get_by_role("heading", level=1)).to_have_text("Панель администрирования")
    expect(page.locator(".admin-table")).to_contain_text(marker)
    expect(page.locator(".admin-table thead th")).to_have_count(7)
    assert_mobile_layout(page)
    page.set_viewport_size({"width": 1440, "height": 1000})
    page.goto(BASE + "/form")
    assert page.locator(".form-panel").bounding_box()["width"] <= 800
    page.locator("#request").fill("Только одно заполненное поле")
    page.get_by_role("button", name="Отправить", exact=True).click()
    page.wait_for_url("**/view&id=*")
    expect(page.locator(".card-table tbody tr")).to_have_count(1)
    expect(page.locator(".card-table")).not_to_contain_text("timestamp")
    page.goto(saved_url.replace("/view&id=", "/view?id="))
    expect(page.locator(".card-table")).to_contain_text(marker)
    for path in ["/view&id=9223372036854775807", "/view&id=invalid"]:
        page.goto(BASE + path)
        expect(page.get_by_role("heading", level=1)).to_have_text("Не найдено")
    assert not errors, errors
    browser.close()
    print("Browser checks passed: welcome, form, retry, save, direct URLs, tables, missing card, mobile layout.")
