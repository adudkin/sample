"""Integration smoke test against the running Compose application."""
import json
import os
import unittest
import urllib.error
import urllib.request
import uuid

BASE = os.environ.get('BASE_URL', 'http://localhost:8080')

class ApplicationTest(unittest.TestCase):
    def request(self, path, data=None):
        body = None if data is None else json.dumps(data).encode()
        return urllib.request.urlopen(urllib.request.Request(BASE + path, data=body, headers={'Content-Type': 'application/json'}), timeout=10)

    def test_pages_and_health(self):
        for path, expected in [('/', 'Сохранить'), ('/admin', 'Все записи'), ('/healthz', 'ok')]:
            with self.request(path) as response:
                self.assertEqual(response.status, 200)
                self.assertIn(expected, response.read().decode())

    def test_save_and_list(self):
        text = 'Проверка\n<script>alert(1)</script> ' + str(uuid.uuid4())
        with self.request('/api/texts', {'text': text}) as response:
            self.assertEqual(response.status, 201)
            saved = json.load(response)
        with self.request('/api/texts') as response:
            entries = json.load(response)
        self.assertIn(saved, entries)
        self.assertEqual(saved['text'], text)

    def test_invalid_input(self):
        for data in [{'text': ''}, {'text': '   '}, {'text': 'x' * 70000}, {'text': 42}, {'text': 'valid', 'unknown': 1}]:
            with self.assertRaises(urllib.error.HTTPError) as error:
                self.request('/api/texts', data)
            self.assertEqual(error.exception.code, 400)

if __name__ == '__main__':
    unittest.main()
