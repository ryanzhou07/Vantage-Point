#!/usr/bin/env python3
import importlib.util
from pathlib import Path
import stat
import sys
import tempfile
import unittest

sys.dont_write_bytecode = True
SPEC = importlib.util.spec_from_file_location("dev_config", Path(__file__).with_name("dev_config.py"))
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class DevConfigurationTests(unittest.TestCase):
    def private_file(self, directory: str, text: str = "GOOGLE_CLIENT_ID=synthetic\n") -> Path:
        path = Path(directory) / "source.env"
        path.write_text(text)
        path.chmod(0o600)
        return path

    def test_primary_root_is_shared_across_worktrees(self):
        checkout = Path(__file__).resolve().parent.parent
        self.assertEqual(MODULE.primary_root(checkout), MODULE.primary_root(checkout.parent.parent))
        self.assertEqual(MODULE.primary_root(checkout) / MODULE.CONFIG_NAME, checkout.parent.parent / MODULE.CONFIG_NAME)

    def test_parser_treats_values_as_literal_data_and_allowlists_names(self):
        with tempfile.TemporaryDirectory() as directory:
            source = self.private_file(directory, 'export GOOGLE_CLIENT_SECRET="$(do-not-execute)"\nNEXT_PUBLIC_API_URL=http://localhost:8080\n')
            values = MODULE.read_private(source)
            self.assertEqual(values["GOOGLE_CLIENT_SECRET"], "$(do-not-execute)")
            source.write_text("UNSUPPORTED_SECRET=synthetic\n")
            with self.assertRaises(ValueError):
                MODULE.read_private(source)

    def test_parser_rejects_unsafe_files_duplicates_and_bad_quotes(self):
        with tempfile.TemporaryDirectory() as directory:
            source = self.private_file(directory)
            source.chmod(0o644)
            with self.assertRaises(ValueError):
                MODULE.read_private(source)
            source.chmod(0o600)
            for text in ('GOOGLE_CLIENT_ID=a\nGOOGLE_CLIENT_ID=b\n', 'GOOGLE_CLIENT_ID="open\n'):
                source.write_text(text)
                with self.assertRaises(ValueError):
                    MODULE.read_private(source)
            link = Path(directory) / "link.env"
            link.symlink_to(source)
            with self.assertRaises(ValueError):
                MODULE.read_private(link)
            source.unlink()
            with self.assertRaises(ValueError):
                MODULE.read_private(link)

    def test_import_is_private_exclusive_and_preserves_source(self):
        with tempfile.TemporaryDirectory() as directory:
            source = self.private_file(directory)
            target = Path(directory) / "stable" / MODULE.CONFIG_NAME
            MODULE.import_private(source, target)
            self.assertEqual(stat.S_IMODE(target.stat().st_mode), 0o600)
            self.assertEqual(source.read_text(), target.read_text())
            target.write_text("GOOGLE_CLIENT_ID=owner-value\n")
            with self.assertRaises(FileExistsError):
                MODULE.import_private(source, target)
            self.assertEqual(target.read_text(), "GOOGLE_CLIENT_ID=owner-value\n")
            with self.assertRaises(ValueError):
                MODULE.import_private(Path(directory) / "missing.env", Path(directory) / "unused.env")

    def test_environment_precedence_and_public_projection(self):
        saved = {"GOOGLE_CLIENT_SECRET": "saved", "NEXT_PUBLIC_API_URL": "saved-public"}
        resolved = MODULE.resolve_config(saved, {"GOOGLE_CLIENT_SECRET": "environment", "NEXT_PUBLIC_SUPABASE_URL": "public", "PATH": "ignored"})
        self.assertEqual(resolved["GOOGLE_CLIENT_SECRET"], "environment")
        self.assertEqual(MODULE.public_environment(resolved), {"NEXT_PUBLIC_API_URL": "saved-public", "NEXT_PUBLIC_SUPABASE_URL": "public"})
        self.assertNotIn("GOOGLE_CLIENT_SECRET", MODULE.public_environment(resolved))


if __name__ == "__main__":
    unittest.main()
