#!/usr/bin/env python3
"""Local-only Auth + RLS + /me integration. Supabase keys remain in subprocess memory."""
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
status = subprocess.run(['pnpm', 'exec', 'supabase', '--workdir', str(root), 'status', '-o', 'json'], capture_output=True, check=True, text=True)
config = json.loads(status.stdout)
base = config['API_URL']
assert base == 'http://127.0.0.1:56321', 'Only the documented local stack is allowed'
public = config['ANON_KEY']
admin = config['SERVICE_ROLE_KEY']
def request(url, token=None, payload=None, key=public):
    req = urllib.request.Request(url, headers={'apikey': key, **({'Authorization': 'Bearer '+token} if token else {}), 'Content-Type': 'application/json'}, data=json.dumps(payload).encode() if payload is not None else None)
    try:
        with urllib.request.urlopen(req, timeout=5) as resp:
            return resp.status, json.loads(resp.read()), resp.headers
    except urllib.error.HTTPError as err:
        return err.code, json.loads(err.read()), err.headers

users = []
server = None
try:
    with tempfile.TemporaryDirectory(prefix='vp-003-auth-') as task_temp:
        binary = str(Path(task_temp)/'api')
        subprocess.run(['go', 'build', '-o', binary, './cmd/api'], cwd=root/'services/backend', check=True)
        env = {**os.environ, 'API_PORT':'58080', 'SUPABASE_URL':base, 'SUPABASE_ANON_KEY':public, 'WEB_ORIGIN':'http://localhost:3000'}
        server = subprocess.Popen([binary], env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        for _ in range(50):
            try:
                urllib.request.urlopen('http://localhost:58080/health', timeout=1).close()
                break
            except OSError:
                time.sleep(.1)
        for i in range(2):
            email = f'vp003-{uuid.uuid4()}@example.invalid'
            password = uuid.uuid4().hex + '-SyntheticFixture!'
            code, user, _ = request(base+'/auth/v1/admin/users', admin, {'email':email,'password':password,'email_confirm':True,'user_metadata':{'full_name':f'Synthetic user {i}'}}, admin)
            assert code == 200, 'Local test identity provisioning failed'
            users.append(user['id'])
            code, session, _ = request(base+'/auth/v1/token?grant_type=password', payload={'email':email,'password':password})
            assert code == 200, 'Local fixture session creation failed'
            users[i] = (user['id'], session['access_token'])
        def verify(identity):
            uid, bearer = identity
            code, body, headers = request('http://localhost:58080/api/v1/me', bearer)
            assert code == 200 and body['profile']['id']==uid, 'Canonical ownership or response failed'
            assert headers['Cache-Control']=='no-store', 'Private response cache policy failed'
            other = users[1][0] if uid==users[0][0] else users[0][0]
            code, rows, _ = request(base+'/rest/v1/profiles?select=id&id=eq.'+other, bearer)
            assert code == 200 and rows == [], 'Real bearer cross-user RLS failed'
        for identity in users: verify(identity)
        with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
            list(pool.map(verify, users*8))
        code, body, _ = request('http://localhost:58080/api/v1/me')
        assert code == 401 and body['error']['request_id'], 'Missing bearer was not rejected'
        code, _, _ = request('http://localhost:58080/api/v1/me', users[0][1][:-10]+'invalidsig')
        assert code == 401, 'Forged bearer accepted'
        code, _, _ = request('http://localhost:58080/api/v1/me?user_id='+users[1][0], users[0][1])
        assert code == 400, 'Forged user identifier accepted'
        print('Real local Supabase Auth + two-user RLS + concurrent /me + forged-input integration passed.')
finally:
    if server:
        server.terminate()
        server.wait(timeout=5)
    for user in users:
        uid = user[0] if isinstance(user, tuple) else user
        req = urllib.request.Request(base+'/auth/v1/admin/users/'+uid, method='DELETE', headers={'apikey':admin,'Authorization':'Bearer '+admin})
        with urllib.request.urlopen(req,timeout=5) as resp:
            assert resp.status == 200
