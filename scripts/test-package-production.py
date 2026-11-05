"""Test release packaging with a fake Docker CLI; never contacts a Docker daemon."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import unittest

SOURCE = Path(__file__).resolve().parents[1]


class PackagingTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        for file in (
            'scripts/package-production.sh', 'compose.build.yml', 'compose.production.yml',
            'deploy/production.env.example', 'deploy/backend.env.example',
            'deploy/website.env.example', 'deploy/README.md',
            'deploy/nginx/citypark.conf.example',
        ):
            target = self.root / file
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(SOURCE / file, target)
        (self.root / 'deploy/secrets').mkdir()
        (self.root / 'deploy/secrets/sentry_auth_token').write_text('SECRET_TOKEN_SENTINEL')
        (self.root / 'deploy/build.env').write_text('UNEXPECTED_SECRET=SECRET_ENV_SENTINEL')
        fixture = {'services': {
            'website': {'build': {'args': {'CUSTOMER_PUBLIC_ORIGIN': 'https://site.example',
                                         'NEXT_PUBLIC_SENTRY_RELEASE': 'web-v1'}}},
            'admin': {'build': {'args': {'VITE_API_URL': 'https://admin.example/api/v1'}}},
        }}
        (self.root / 'config.json').write_text(json.dumps(fixture))
        bin_dir = self.root / 'bin'
        bin_dir.mkdir()
        fake = bin_dir / 'docker'
        fake.write_text('''#!/usr/bin/env python3
import json, os, pathlib, sys
root = pathlib.Path(os.environ['FIXTURE_ROOT'])
args = sys.argv[1:]
with (root / 'calls.jsonl').open('a') as f:
    f.write(json.dumps(args) + '\\n')
if 'config' in args:
    print((root / 'config.json').read_text())
elif args[:2] == ['image', 'inspect']:
    if args[3] == '{{.Os}}/{{.Architecture}}':
        print(os.environ.get('TEST_PLATFORM', 'linux/amd64'))
    else:
        print(args[-1] + ' sha256:fake linux/amd64')
elif args[:2] == ['image', 'save']:
    if os.environ.get('FAIL_SAVE'):
        sys.exit(7)
    print('SIMULATED_IMAGE_ARCHIVE')
''')
        fake.chmod(0o755)
        self.env = dict(os.environ, PATH=str(bin_dir) + os.pathsep + os.environ['PATH'],
                        FIXTURE_ROOT=str(self.root))

    def run_packager(self, tag='test-1'):
        return subprocess.run(['bash', str(self.root / 'scripts/package-production.sh'), tag],
                              env=self.env, capture_output=True, text=True)

    def test_bundle_is_self_contained_and_contains_no_secrets(self):
        result = self.run_packager()
        self.assertEqual(result.returncode, 0, result.stderr)
        release = self.root / 'deploy/releases/test-1'
        self.assertEqual({p.name for p in release.iterdir()},
                         {'deployment.tar.gz', 'images.tar.gz', 'SHA256SUMS'})
        for line in (release / 'SHA256SUMS').read_text().splitlines():
            checksum, name = line.split()
            self.assertEqual(hashlib.sha256((release / name).read_bytes()).hexdigest(), checksum)
        with tarfile.open(release / 'deployment.tar.gz') as archive:
            files = {m.name.removeprefix('./'): archive.extractfile(m).read().decode()
                     for m in archive.getmembers() if m.isfile()}
        self.assertEqual(set(files), {'compose.production.yml', 'RELEASE', 'IMAGES.txt',
            'deploy/production.env.example', 'deploy/backend.env.example',
            'deploy/website.env.example', 'deploy/README.md', 'deploy/nginx/citypark.conf.example'})
        contents = '\n'.join(files.values())
        self.assertNotIn('SECRET_TOKEN_SENTINEL', contents)
        self.assertNotIn('SECRET_ENV_SENTINEL', contents)
        self.assertIn("RELEASE_TAG='test-1'", files['deploy/production.env.example'])
        self.assertIn("SITE_ORIGIN='https://site.example'", files['deploy/production.env.example'])
        calls = [json.loads(line) for line in (self.root / 'calls.jsonl').read_text().splitlines()]
        self.assertIn(['pull', '--platform', 'linux/amd64', 'postgres:17'], calls)
        self.assertIn(['image', 'save', *['citypark/' + service + ':test-1'
                      for service in ('backend', 'website', 'admin', 'postgres')]], calls)
        # Published artifacts must never be overwritten.
        self.assertNotEqual(self.run_packager().returncode, 0)

    def test_failed_save_does_not_publish_partial_release(self):
        self.env['FAIL_SAVE'] = '1'
        self.assertNotEqual(self.run_packager().returncode, 0)
        self.assertEqual(list((self.root / 'deploy/releases').iterdir()), [])

    def test_wrong_architecture_is_rejected(self):
        self.env['TEST_PLATFORM'] = 'linux/arm64'
        result = self.run_packager()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Unexpected platform', result.stderr)
        self.assertEqual(list((self.root / 'deploy/releases').iterdir()), [])

    def test_invalid_tag_is_rejected_before_docker(self):
        self.assertEqual(self.run_packager('../bad').returncode, 2)
        self.assertFalse((self.root / 'calls.jsonl').exists())


if __name__ == '__main__':
    unittest.main()
