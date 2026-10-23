"""Read-only site checks plus an optional temporary ACME webroot probe.

DOMAIN defaults to vsevmeste.tech. CA_CERT can trust a local test CA; certificate
verification is always enabled. ACME_WEBROOT enables an on-disk probe, which is
removed when the test finishes. No application or database records are changed.
"""
import os
from pathlib import Path
import ssl
import tempfile
import unittest
import urllib.error
import urllib.request
import uuid

DOMAIN = os.environ.get('DOMAIN', 'vsevmeste.tech')
HTTP_BASE = os.environ.get('HTTP_BASE_URL', f'http://{DOMAIN}').rstrip('/')
HTTPS_BASE = os.environ.get('HTTPS_BASE_URL', f'https://{DOMAIN}').rstrip('/')
TLS = ssl.create_default_context(cafile=os.environ.get('CA_CERT'))


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, request, fp, code, message, headers, new_url):
        return None


HTTP = urllib.request.build_opener(NoRedirect())


class DeploymentTest(unittest.TestCase):
    def test_http_redirect(self):
        path = '/view&id=1?source=http'
        # The redirect uses the configured domain, even with an unrelated Host.
        for host in [DOMAIN, 'untrusted.invalid']:
            request = urllib.request.Request(HTTP_BASE + path, headers={'Host': host})
            with self.assertRaises(urllib.error.HTTPError) as failure:
                HTTP.open(request, timeout=10)
            self.assertEqual(failure.exception.code, 308)
            self.assertEqual(failure.exception.headers['Location'], f'https://{DOMAIN}{path}')

    def test_https_application(self):
        with urllib.request.urlopen(HTTPS_BASE + '/healthz', timeout=10, context=TLS) as response:
            self.assertEqual(response.status, 200)
            self.assertEqual(response.read().strip(), b'ok')
        with urllib.request.urlopen(HTTPS_BASE + '/', timeout=10, context=TLS) as response:
            self.assertIn(b'<div id="app">', response.read())

    def test_missing_challenge_is_404(self):
        url = HTTP_BASE + '/.well-known/acme-challenge/missing-' + uuid.uuid4().hex
        with self.assertRaises(urllib.error.HTTPError) as failure:
            HTTP.open(url, timeout=10)
        self.assertEqual(failure.exception.code, 404)

    @unittest.skipUnless(os.environ.get('ACME_WEBROOT'), 'Set ACME_WEBROOT to probe a local shared webroot')
    def test_acme_webroot(self):
        webroot = Path(os.environ['ACME_WEBROOT'])
        for directory in [webroot / '.well-known', webroot / '.well-known' / 'acme-challenge']:
            if not directory.exists():
                directory.mkdir()
                directory.chmod(0o755)
        contents = 'sample-acme-probe-' + uuid.uuid4().hex
        with tempfile.NamedTemporaryFile(mode='w', prefix='sample-probe-', dir=directory, delete=False) as token:
            token.write(contents)
            path = Path(token.name)
        try:
            path.chmod(0o644)
            with HTTP.open(HTTP_BASE + '/.well-known/acme-challenge/' + path.name, timeout=10) as response:
                self.assertEqual(response.status, 200)
                self.assertEqual(response.read().decode(), contents)
        finally:
            path.unlink(missing_ok=True)


if __name__ == '__main__':
    unittest.main()
