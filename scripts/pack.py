#!/usr/bin/env python3
"""One-shot packer: build frontend + Go binary and drop the GM exe where I need it.

Usage:
    python scripts/pack.py                 # full build (frontend + backend)
    python scripts/pack.py --skip-frontend # reuse the assets already on disk
"""

from __future__ import annotations

import argparse
from pathlib import Path
import shutil
import subprocess
import sys
import zipfile

try:
    sys.stdout.reconfigure(encoding="utf-8")
except AttributeError:
    pass

ROOT = Path(__file__).resolve().parents[1]
SCRIPT_DIR = ROOT / "script"
sys.path.insert(0, str(SCRIPT_DIR))

# Final destination for the packaged executable.
DEST_DIR = ROOT.parent / "参考" / "打包"
EXE_NAME = "Pst魔改版-GM.exe"
SAV_CLI_NAME = "sav_cli.exe"
DEFAULT_VERSION = "GM-0.1.0"

def run(command: list[str], *, cwd: Path = ROOT, shell: bool = False) -> None:
    printable = command if shell else subprocess.list2cmdline(command)
    print("+", printable, flush=True)
    subprocess.run(command, cwd=cwd, check=True, shell=shell)


def build_frontend() -> None:
    web = ROOT / "web"
    if not (web / "package.json").is_file():
        raise FileNotFoundError(f"frontend project not found: {web}")
    pnpm = shutil.which("pnpm") or shutil.which("pnpm.cmd")
    if pnpm is None:
        raise RuntimeError("pnpm not found in PATH; install it or pass --skip-frontend")
    run([pnpm, "run", "build"], cwd=web)


def resolve_sav_cli(explicit: Path | None) -> Path:
    """Pick the newest sav_cli.exe among the known locations."""
    candidates: list[Path] = []
    if explicit is not None:
        candidates.append(explicit)
    candidates += [
        ROOT / SAV_CLI_NAME,
        DEST_DIR / SAV_CLI_NAME,
        ROOT / "dist" / "sav-cli" / SAV_CLI_NAME,
    ]

    existing = [p.resolve() for p in candidates if p.is_file()]
    if not existing:
        raise FileNotFoundError(
            "sav_cli.exe not found. Build it with script/build_sav_cli.py "
            "or pass --sav-cli <path>."
        )
    if explicit is not None:
        return existing[0]

    # Deduplicate while keeping the newest build.
    unique: dict[Path, float] = {p: p.stat().st_mtime for p in existing}
    newest = max(unique, key=unique.get)
    for path, mtime in sorted(unique.items(), key=lambda kv: -kv[1]):
        size_mb = path.stat().st_size / 1024 / 1024
        marker = "  <- using" if path == newest else ""
        print(f"  sav_cli candidate: {path} ({size_mb:.2f} MB){marker}", flush=True)
    return newest


def sync_sav_cli(source: Path) -> None:
    """Keep the copy next to the packaged exe in sync with the chosen build."""
    target = DEST_DIR / SAV_CLI_NAME
    if target.resolve() == source.resolve():
        return
    DEST_DIR.mkdir(parents=True, exist_ok=True)
    shutil.copy2(source, target)
    print(f"synced {target} ({target.stat().st_size / 1024 / 1024:.2f} MB)", flush=True)


def extract_exe(archive: Path) -> Path:
    DEST_DIR.mkdir(parents=True, exist_ok=True)
    destination = DEST_DIR / EXE_NAME
    with zipfile.ZipFile(archive) as zf:
        member = next((n for n in zf.namelist() if n.endswith(".exe") and "sav_cli" not in n), None)
        if member is None:
            raise RuntimeError(f"no PST executable inside {archive}")
        with zf.open(member) as src, destination.open("wb") as dst:
            shutil.copyfileobj(src, dst)
    return destination


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--version", default=DEFAULT_VERSION)
    parser.add_argument("--sav-cli", type=Path, default=None)
    parser.add_argument("--skip-frontend", action="store_true")
    args = parser.parse_args()

    if args.skip_frontend:
        print("== frontend: skipped ==", flush=True)
    else:
        print("== frontend build ==", flush=True)
        build_frontend()

    print("== sav_cli ==", flush=True)
    sav_cli = resolve_sav_cli(args.sav_cli)
    sync_sav_cli(sav_cli)

    print("== go build + package ==", flush=True)
    from package_release import package

    artifacts = package(
        args.version,
        "windows",
        "amd64",
        sav_cli,
        (ROOT / "dist" / "release").resolve(),
    )
    archive = next(p for p in artifacts if p.suffix == ".zip")

    print("== deploy ==", flush=True)
    exe = extract_exe(archive)
    print(f"OK {exe} ({exe.stat().st_size / 1024 / 1024:.2f} MB)", flush=True)


if __name__ == "__main__":
    main()
