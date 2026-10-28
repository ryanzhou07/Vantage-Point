#!/usr/bin/env python3
"""Run Receipt-only commands using a stable private file parsed as data, not shell.
No networking is performed by this loader. Smoke requires explicit live opt-in.
"""
import argparse
import os
from pathlib import Path
import shutil
import stat
import subprocess
import sys

ALLOWED = {'GEMINI_API_KEY', 'RECEIPT_VISION_PROVIDER', 'RECEIPT_ALLOW_FAKE', 'SUPABASE_URL', 'SUPABASE_SERVICE_ROLE_KEY'}

def read_private(path):
    if not path.exists():
        return {}
    if not path.is_file() or stat.S_IMODE(path.stat().st_mode) & 0o077:
        raise ValueError('Receipt configuration must be a private regular file (chmod 600).')
    if path.stat().st_size > 16384:
        raise ValueError('Receipt configuration file is too large.')
    values = {}
    for line in path.read_text().splitlines():
        line = line.strip()
        if not line or line.startswith('#'):
            continue
        key, separator, value = line.partition('=')
        key, value = key.strip(), value.strip()
        if not separator or key not in ALLOWED or key in values:
            raise ValueError('Receipt file contains unsupported or duplicate variable names; values were not executed.')
        if value.startswith(('"', "'")):
            if len(value) < 2 or value[-1] != value[0]:
                raise ValueError('Receipt file contains an unmatched quote.')
            value = value[1:-1]
        if '\x00' in value:
            raise ValueError('Receipt file contains invalid data.')
        values[key] = value
    return values

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=('smoke', 'worker'))
    parser.add_argument('--live-synthetic', action='store_true', help='Allow one live Gemini request with the embedded synthetic image')
    args = parser.parse_args()
    if args.command == 'smoke' and not args.live_synthetic:
        parser.error('Smoke sends one real provider request; add --live-synthetic to opt in.')
    root = Path(__file__).resolve().parent.parent
    common = Path(subprocess.check_output(['git', '-C', str(root), 'rev-parse', '--path-format=absolute', '--git-common-dir'], text=True).strip())
    primary = common.parent
    try:
        configuration = read_private(primary / '.env.receipts.local')
    except (ValueError, OSError):
        print('Receipt configuration could not be loaded safely. Check file format and chmod 600; no values were printed.', file=sys.stderr)
        return 2
    env = {**os.environ, **configuration}
    if args.command == 'smoke':
        if not env.get('GEMINI_API_KEY'):
            print('GEMINI_API_KEY is not configured in the stable root .env.receipts.local. No request sent.', file=sys.stderr)
            return 2
        env['RECEIPT_LIVE_SMOKE'] = 'synthetic-only'
    executable = shutil.which('go')
    # Reuse the repository's pinned local Go toolchain when already installed;
    # do not provision tools or change the system-wide environment here.
    cached = sorted((primary / '.local-tools/gopath/pkg/mod/golang.org').glob('toolchain@v0.0.1-go1.26.10.*/bin/go'))
    if cached:
        executable = str(cached[0])
        env['GOPATH'] = str(primary / '.local-tools/gopath')
        env['GOCACHE'] = str(primary / '.local-tools/gocache')
    if not executable:
        print('Install the pinned Go version from go.mod first.', file=sys.stderr)
        return 2
    command = 'receipt-smoke' if args.command == 'smoke' else 'receipt-worker'
    os.chdir(root / 'services/backend')
    os.execve(executable, [executable, 'run', './cmd/' + command], env)
    return 0

if __name__ == '__main__':
    sys.exit(main())
