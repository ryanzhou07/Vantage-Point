#!/usr/bin/env python3
import importlib.util
from pathlib import Path
import tempfile
import sys
sys.dont_write_bytecode = True
import unittest

spec = importlib.util.spec_from_file_location('receipt_command', Path(__file__).with_name('receipt-command.py'))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

class PrivateConfigurationTests(unittest.TestCase):
    def test_parse_as_literal_data_and_deny_public_keys(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'private.env'
            path.write_text('GEMINI_API_KEY="$(do-not-execute)"\nRECEIPT_VISION_PROVIDER=gemini\n')
            path.chmod(0o600)
            self.assertEqual(module.read_private(path)['GEMINI_API_KEY'], '$(do-not-execute)')
            path.write_text('NEXT_PUBLIC_GEMINI_API_KEY=synthetic\n')
            with self.assertRaises(ValueError): module.read_private(path)
    def test_permissions_duplicate_and_unmatched_quote(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'private.env'
            path.write_text('GEMINI_API_KEY=synthetic\n');path.chmod(0o644)
            with self.assertRaises(ValueError): module.read_private(path)
            path.chmod(0o600)
            for value in ('GEMINI_API_KEY=a\nGEMINI_API_KEY=b\n', 'GEMINI_API_KEY="unclosed\n'):
                path.write_text(value)
                with self.assertRaises(ValueError): module.read_private(path)

if __name__ == '__main__': unittest.main()
