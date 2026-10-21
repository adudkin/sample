"""Functional API checks for a running stack; creates a few test cards."""
import json
import os
import ssl
import unittest
import urllib.error
import urllib.request
import uuid

BASE = os.environ.get("BASE_URL", "https://localhost").rstrip("/")
# For a local self-signed certificate, set CA_CERT to its path; TLS stays verified.
TLS = ssl.create_default_context(cafile=os.environ.get("CA_CERT"))


class ApplicationTest(unittest.TestCase):
    def request(self, path, data=None, raw=None):
        body = json.dumps(data).encode() if data is not None else raw
        request = urllib.request.Request(
            BASE + path, data=body, headers={"Content-Type": "application/json"}
        )
        return urllib.request.urlopen(request, timeout=10, context=TLS)

    def test_frontend_routes_and_health(self):
        for path in ["/", "/form", "/admin", "/view&id=1", "/view?id=1"]:
            with self.request(path) as response:
                self.assertEqual(response.status, 200)
                self.assertIn('<div id="app">', response.read().decode())
        with self.request("/healthz") as response:
            self.assertEqual(response.read().decode().strip(), "ok")

    def test_save_read_and_list(self):
        fields = {
            "request": "Нужна помощь " + str(uuid.uuid4()),
            "offer": "Время\n<script>alert(1)</script>",
            "details": "Москва, в выходные",
            "consideration": "Уточнить время",
            "contact": "Иван, ivan@example.org",
        }
        with self.request("/api/cards", fields) as response:
            self.assertEqual(response.status, 201)
            saved = json.load(response)
        self.assertEqual(set(saved), {"id", "timestamp", *fields})
        self.assertIsInstance(saved["id"], int)
        self.assertGreaterEqual(saved["id"], 1)
        for key, value in fields.items():
            self.assertEqual(saved[key], value)
        self.assertTrue(saved["timestamp"])
        with self.request("/api/cards/" + str(saved["id"])) as response:
            self.assertEqual(json.load(response), saved)
        with self.request("/api/cards") as response:
            entries = json.load(response)
        self.assertIn(saved, entries)
        self.assertEqual(
            [card["id"] for card in entries],
            sorted([card["id"] for card in entries], reverse=True),
        )

    def test_optional_fields(self):
        with self.request("/api/cards", {"request": "Только запрос"}) as response:
            saved = json.load(response)
        for field in ["offer", "details", "consideration", "contact"]:
            self.assertEqual(saved[field], "")

    def test_invalid_input(self):
        invalid = [
            {}, {"request": ""}, {"request": "   "},
            {"request": "я" * 10001}, {"request": 42},
            {"request": "Нужно", "unknown": "value"},
            {"request": "Нужно", "offer": "я" * 10001},
        ]
        for data in invalid:
            with self.subTest(data_type=list(data)):
                with self.assertRaises(urllib.error.HTTPError) as error:
                    self.request("/api/cards", data)
                self.assertEqual(error.exception.code, 400)
        for body in [b'{', b'{"request":"ok"} {}']:
            with self.assertRaises(urllib.error.HTTPError) as error:
                self.request("/api/cards", raw=body)
            self.assertEqual(error.exception.code, 400)

    def test_missing_card(self):
        with self.assertRaises(urllib.error.HTTPError) as error:
            self.request("/api/cards/9223372036854775807")
        self.assertEqual(error.exception.code, 404)
        self.assertEqual(json.load(error.exception)["error"], "Не найдено")


if __name__ == "__main__":
    unittest.main()
