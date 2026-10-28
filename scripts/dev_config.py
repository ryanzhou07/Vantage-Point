"""Private local-development configuration, parsed as data and never executed."""
from pathlib import Path
import os
import stat
import subprocess

PUBLIC_KEYS = frozenset({"NEXT_PUBLIC_API_URL", "NEXT_PUBLIC_SUPABASE_URL", "NEXT_PUBLIC_SUPABASE_ANON_KEY"})
PRIVATE_KEYS = frozenset({"GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET", "SUPABASE_URL", "SUPABASE_ANON_KEY", "SUPABASE_SERVICE_ROLE_KEY", "WEB_ORIGIN", "GEMINI_API_KEY", "RECEIPT_VISION_PROVIDER", "RECEIPT_ALLOW_FAKE"})
ALLOWED_KEYS = PUBLIC_KEYS | PRIVATE_KEYS
CONFIG_NAME = ".env.development.local"


def primary_root(checkout: Path) -> Path:
    common = subprocess.check_output(
        ["git", "-C", str(checkout), "rev-parse", "--path-format=absolute", "--git-common-dir"], text=True
    ).strip()
    return Path(common).resolve().parent


def read_private(path: Path) -> dict[str, str]:
    try:
        descriptor = os.open(path, os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0))
    except FileNotFoundError:
        return {}
    except OSError as error:
        raise ValueError("Configuration must be a private regular file (chmod 600) under 16 KiB.") from error
    with os.fdopen(descriptor) as handle:
        details = os.fstat(handle.fileno())
        if not stat.S_ISREG(details.st_mode) or stat.S_IMODE(details.st_mode) & 0o077 or details.st_size > 16384:
            raise ValueError("Configuration must be a private regular file (chmod 600) under 16 KiB.")
        lines = handle.read().splitlines()
    values: dict[str, str] = {}
    for raw in lines:
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        key, separator, value = line.partition("=")
        key, value = key.removeprefix("export ").strip(), value.strip()
        if not separator or key not in ALLOWED_KEYS or key in values:
            raise ValueError("Configuration contains an unsupported or duplicate variable name.")
        if value[:1] in {'"', "'"}:
            if len(value) < 2 or value[-1] != value[0]:
                raise ValueError("Configuration contains an unmatched quote.")
            value = value[1:-1]
        if "\x00" in value:
            raise ValueError("Configuration contains invalid data.")
        values[key] = value
    return values


def import_private(source: Path, target: Path) -> None:
    values = read_private(source)
    if not values:
        raise ValueError("Import source contains no supported configuration.")
    target.parent.mkdir(parents=True, exist_ok=True)
    descriptor = os.open(target, os.O_WRONLY | os.O_CREAT | os.O_EXCL | getattr(os, "O_NOFOLLOW", 0), 0o600)
    with os.fdopen(descriptor, "w") as handle:
        handle.writelines(f"{key}={value}\n" for key, value in values.items())


def resolve_config(saved: dict[str, str], environ: dict[str, str]) -> dict[str, str]:
    return {key: environ[key] if key in environ else value for key, value in saved.items()} | {
        key: value for key, value in environ.items() if key in ALLOWED_KEYS and key not in saved
    }


def public_environment(values: dict[str, str]) -> dict[str, str]:
    return {key: values[key] for key in PUBLIC_KEYS if key in values}
