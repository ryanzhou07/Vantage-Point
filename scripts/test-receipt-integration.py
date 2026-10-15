#!/usr/bin/env python3
"""Synthetic two-owner Receipt integration; NEVER targets the owner's live stack.
Use a dedicated Supabase workdir/project vantage-point-receipt-tests, or CI's
fresh vantage-point-local stack with CI=true. No database reset in this script.
"""
import concurrent.futures
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time
import urllib.error
import urllib.request
import uuid

root = Path(__file__).resolve().parent.parent
workdir = Path(os.environ.get('RECEIPT_TEST_WORKDIR', str(root))).resolve()
configuration = (workdir / 'supabase/config.toml').read_text()
assert 'project_id = "vantage-point-receipt-tests"' in configuration or (
    os.environ.get('CI') == 'true' and os.environ.get('GITHUB_ACTIONS') == 'true' and workdir == root and 'project_id = "vantage-point-local"' in configuration
), 'Use a dedicated Receipt test stack; owner preview is not an integration target'
status = subprocess.run(['pnpm', 'exec', 'supabase', '--workdir', str(workdir), 'status', '-o', 'json'], capture_output=True, check=True, text=True)
config = json.loads(status.stdout)
base = config['API_URL']
assert base in ('http://127.0.0.1:59321', 'http://127.0.0.1:56321')
public, admin = config['ANON_KEY'], config['SERVICE_ROLE_KEY']
api = 'http://127.0.0.1:59080'
users, processes = [], []

def request(url, token=None, payload=None, method=None, key=public, mime='application/json', raw=False):
    data = payload if isinstance(payload, bytes) else json.dumps(payload).encode() if payload is not None else None
    req = urllib.request.Request(url, method=method, headers={'apikey': key, **({'Authorization': 'Bearer ' + token} if token else {}), 'Content-Type': mime}, data=data)
    try:
        with urllib.request.urlopen(req, timeout=12) as response:
            body = response.read()
            return response.status, body if raw else json.loads(body) if body else None, response.headers
    except urllib.error.HTTPError as failure:
        body = failure.read()
        try: body = json.loads(body)
        except (ValueError, UnicodeDecodeError): pass
        return failure.code, body, failure.headers

try:
    with tempfile.TemporaryDirectory(prefix='vp008-receipt-') as directory:
        for command in ('api', 'receipt-worker'):
            subprocess.run(['go', 'build', '-o', str(Path(directory) / command), './cmd/' + command], cwd=root / 'services/backend', check=True)
        env = {**os.environ, 'API_PORT': '59080', 'SUPABASE_URL': base, 'SUPABASE_ANON_KEY': public,
               'SUPABASE_SERVICE_ROLE_KEY': admin, 'WEB_ORIGIN': 'http://localhost:3000',
               'RECEIPT_VISION_PROVIDER': 'fake', 'RECEIPT_ALLOW_FAKE': 'development-test-only'}
        # Never inherit a live provider key into deterministic test children.
        env.pop('GEMINI_API_KEY', None)
        for command in ('api', 'receipt-worker'):
            processes.append(subprocess.Popen([str(Path(directory) / command)], env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL))
        for _ in range(50):
            try:
                urllib.request.urlopen(api + '/health', timeout=1).close(); break
            except OSError: time.sleep(.1)
        for index in range(2):
            email, password = f'vp008-{uuid.uuid4()}@example.invalid', uuid.uuid4().hex + '-Synthetic!'
            code, user, _ = request(base + '/auth/v1/admin/users', admin, {'email': email, 'password': password, 'email_confirm': True}, key=admin)
            assert code == 200, 'fixture identity failed'
            users.append((user['id'], None))
            code, session, _ = request(base + '/auth/v1/token?grant_type=password', payload={'email': email, 'password': password})
            assert code == 200
            users[-1] = (user['id'], session['access_token'])
        owner, other = users[0][1], users[1][1]
        image = (root / 'services/backend/cmd/receipt-smoke/testdata/synthetic.png').read_bytes()
        code, first, headers = request(api + '/api/v1/receipts', owner, image, mime='image/png')
        assert code == 202 and headers['Cache-Control'] == 'no-store', 'upload failed'
        rid = first['id']
        code, duplicate, _ = request(api + '/api/v1/receipts', owner, image, mime='image/png')
        assert code == 202 and duplicate['id'] == rid, 'duplicate upload not coalesced'
        code, second, _ = request(api + '/api/v1/receipts', other, image, mime='image/png')
        assert code == 202 and second['id'] != rid, 'cross-owner hash collision'
        assert request(api + '/api/v1/receipts/' + rid, other)[0] == 404, 'API owner isolation'
        assert request(api + '/api/v1/receipts/' + rid + '/image', other, raw=True)[0] == 404, 'API image isolation'
        code, rows, _ = request(base + '/rest/v1/receipts?id=eq.' + rid, other)
        assert code == 200 and rows == [], 'receipt RLS isolation'
        assert request(base + '/rest/v1/rpc/receipt_create', other, {'p_owner': users[0][0], 'p_hash': 'a' * 64, 'p_mime': 'image/png'})[0] in (401, 403, 404), 'browser privileged RPC accessible'
        path = first['image_path']
        assert request(base + '/storage/v1/object/authenticated/receipt-images/' + path, other, raw=True)[0] != 200, 'private image RLS isolation'
        assert request(base + '/storage/v1/object/public/receipt-images/' + path, raw=True)[0] != 200, 'public image accessible'
        assert request(api + '/api/v1/receipts/' + rid + '/image', owner, raw=True)[1] == image, 'owner image unavailable'
        assert request(api + '/api/v1/receipts', owner, b'<svg/>', mime='image/png')[0] == 422, 'file spoof accepted'
        assert request(api + '/api/v1/receipts')[0] == 401, 'anonymous read accepted'
        for _ in range(20):
            code, record, _ = request(api + '/api/v1/receipts/' + rid, owner)
            if record['status'] == 'NEEDS_REVIEW': break
            time.sleep(.3)
        assert record['status'] == 'NEEDS_REVIEW', 'worker did not persist proposal'
        assert record['draft'] is None and record['preview'] is None, 'model output treated as verified'
        draft = {**record['proposal'], 'participants': [{'id': 'a', 'name': 'Student A', 'included': True}, {'id': 'b', 'name': 'Student B', 'included': True}], 'tax_method': 'equal', 'tip_method': 'proportional'}
        draft['items'][0]['participant_id'] = 'a'
        version = record['version']
        payload = {'version': version, 'draft': draft, 'calculate': True}
        def save(): return request(api + '/api/v1/receipts/' + rid + '/review', owner, payload)
        with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
            results = list(pool.map(lambda _: save(), range(2)))
        assert sorted(item[0] for item in results) == [200, 409], 'stale concurrent updates not rejected'
        record = next(item[1] for item in results if item[0] == 200)
        assert sum(item['total_minor'] for item in record['preview']['shares']) == 1280, 'money conservation failed'
        assert record['status'] == 'NEEDS_REVIEW' and record['preview']['draft'] is True, 'draft finalized accidentally'
        assert request(api + '/api/v1/receipts/' + rid, owner)[1]['preview'] == record['preview'], 'preview not persisted'
        bad = {**draft, 'total_minor': 1}
        assert request(api + '/api/v1/receipts/' + rid + '/review', owner, {'version': record['version'], 'draft': bad, 'calculate': True})[0] == 422, 'unreconciled preview accepted'
        code, saved_draft, _ = request(api + '/api/v1/receipts/' + rid + '/review', owner, {'version': record['version'], 'draft': draft, 'calculate': False})
        assert code == 200 and saved_draft['preview'] is None, 'Save-draft null preview did not persist'
        print('Receipt integration passed: private upload/image, real two-user Auth/RLS, durable fake worker, hash deduplication, concurrent 200/409, integer preview persistence and invalid-input rejection.')
finally:
    for process in processes:
        process.terminate()
    for process in processes:
        try: process.wait(timeout=6)
        except subprocess.TimeoutExpired: process.kill(); process.wait()
    for uid, _ in users:
        # Auth deletion cascades receipt records; explicitly delete fixture image objects first.
        request(base + '/storage/v1/object/receipt-images', admin, {'prefixes': [f'{uid}/{row["id"]}.png' for row in request(base + '/rest/v1/receipts?owner_id=eq.' + uid, admin, key=admin)[1]]}, method='DELETE', key=admin)
        request(base + '/auth/v1/admin/users/' + uid, admin, method='DELETE', key=admin)
