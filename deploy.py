#!/usr/bin/env python3
"""
  python3 deploy.py \
  --repo /root/work \
  --url git@github.com:maxBezel/ledgerbot.git \
  --branch main \
  --work-dir /root/ \
  --service ledgerbot.service \
  --binary-name ledgerbot
"""

import argparse
import os
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

def which_or_die(cmd: str):
    path = shutil.which(cmd)
    if not path:
        sys.exit(f"ERROR: '{cmd}' not found in PATH.")
    return path

def run(cmd, cwd=None, check=True, capture_output=True, text=True):
    proc = subprocess.run(cmd, cwd=cwd, check=check, capture_output=capture_output, text=text)
    return proc

def get_current_head(repo: Path) -> str:
    out = run(["git", "rev-parse", "HEAD"], cwd=repo).stdout.strip()
    return out

def fetch_origin(repo: Path, remote: str = "origin"):
    run(["git", "fetch", "--prune", remote], cwd=repo)

def get_remote_head(repo: Path, branch: str, remote: str = "origin") -> str:
    out = run(["git", "rev-parse", f"{remote}/{branch}"], cwd=repo).stdout.strip()
    return out

def pull(repo: Path):
    try:
        run(["git", "pull", "--ff-only"], cwd=repo)
    except subprocess.CalledProcessError:
        run(["git", "pull"], cwd=repo)

def clone_repo(repo: Path, url: str, branch: str, remote: str = "origin"):
    repo.parent.mkdir(parents=True, exist_ok=True)
    print(f"Cloning {url} (branch {branch}) into {repo} ...")
    run(["git", "clone", "--branch", branch, url, str(repo)])

def build_go(repo: Path, binary_name: str, ldflags: str = "", extra_build_args=None) -> Path:
    which_or_die("go")
    build_dir = Path(tempfile.mkdtemp(prefix="go-build-"))
    out_path = build_dir / binary_name
    cmd = ["go", "build", "-o", str(out_path)]
    if ldflags:
        cmd.extend(["-ldflags", ldflags])
    if extra_build_args:
        cmd.extend(extra_build_args)
    run(cmd, cwd=repo)
    if not out_path.exists():
        sys.exit("ERROR: Build succeeded but output binary not found.")
    return out_path

def deploy_binary(new_binary: Path, work_dir: Path, binary_name: str) -> Path:
    work_dir.mkdir(parents=True, exist_ok=True)
    dest = work_dir / binary_name
    tmp_dest = work_dir / (binary_name + ".new")
    shutil.copy2(new_binary, tmp_dest)
    os.chmod(tmp_dest, 0o755)
    os.replace(tmp_dest, dest)
    return dest

def restart_service(service: str, user_service: bool, use_sudo: bool):
    if user_service:
        run(["systemctl", "--user", "daemon-reload"])
        run(["systemctl", "--user", "restart", service])
        run(["systemctl", "--user", "status", service, "--no-pager"], check=False)
    else:
        base = ["systemctl"]
        if use_sudo:
            base.insert(0, "sudo")
        run(base + ["daemon-reload"])
        run(base + ["restart", service])
        run(base + ["status", service, "--no-pager"], check=False)

def main():
    parser = argparse.ArgumentParser(description="Clone/pull repo, build Go, deploy to ~/work, restart service if updated.")
    parser.add_argument("--repo", required=True, type=Path, help="Path to the git repository")
    parser.add_argument("--url", help="Git URL (required if repo is missing)")
    parser.add_argument("--branch", default="main", help="Branch to track (default: main)")
    parser.add_argument("--remote", default="origin", help="Remote name (default: origin)")
    parser.add_argument("--service", required=True, help="systemd unit name, e.g., myapp.service")
    parser.add_argument("--binary-name", required=True, help="Name of the output binary (and file in work dir)")
    parser.add_argument("--work-dir", type=Path, default=Path.home() / "work", help="Destination directory (default: ~/work)")
    parser.add_argument("--user-service", action="store_true", help="Use systemctl --user for a user service")
    parser.add_argument("--use-sudo", action="store_true", help="Run systemctl with sudo (for system services)")
    parser.add_argument("--always-build", action="store_true", help="Build/restart even if no git updates detected")
    parser.add_argument("--ldflags", default="", help="Go build -ldflags string")
    parser.add_argument("--extra-build-arg", action="append", dest="extra_build_args", help="Extra args passed to `go build` (repeatable)")
    args = parser.parse_args()

    which_or_die("git")
    which_or_die("systemctl")

    repo = args.repo.resolve()

    if not repo.exists() or not (repo / ".git").exists():
        if not args.url:
            sys.exit("ERROR: Repo not found locally. Provide --url to clone it.")
        clone_repo(repo, args.url, args.branch, args.remote)

    try:
        local_head_before = get_current_head(repo)
    except subprocess.CalledProcessError:
        sys.exit("ERROR: Unable to determine local HEAD. Is this a valid git repo?")

    fetch_origin(repo, args.remote)
    try:
        remote_head = get_remote_head(repo, args.branch, args.remote)
    except subprocess.CalledProcessError:
        sys.exit(f"ERROR: Remote branch {args.remote}/{args.branch} not found.")

    changed = (remote_head != local_head_before)

    if changed:
        print(f"Remote {args.remote}/{args.branch} advanced: {local_head_before[:8]} → {remote_head[:8]}. Pulling...")
        pull(repo)
    else:
        print(f"No upstream changes on {args.remote}/{args.branch}.")
        if not args.always_build:
            print("Nothing to do. Use --always-build to force build/deploy/restart.")
            return

    print("Building Go project...")
    new_bin = build_go(repo, args.binary_name, ldflags=args.ldflags, extra_build_args=args.extra_build_args)

    dest = deploy_binary(new_bin, args.work_dir, args.binary_name)
    print(f"Deployed binary to: {dest}")

    print(f"Restarting service: {args.service} ({'user' if args.user_service else 'system'})")
    restart_service(args.service, user_service=args.user_service, use_sudo=args.use_sudo)

    try:
        local_head_after = get_current_head(repo)
        print(f"Current HEAD: {local_head_after}")
    except Exception:
        pass

if __name__ == "__main__":
    try:
        main()
    except subprocess.CalledProcessError as e:
        sys.stderr.write(f"Command failed with exit code {e.returncode}:\n  {' '.join(e.cmd)}\nSTDOUT:\n{e.stdout}\nSTDERR:\n{e.stderr}\n")
        sys.exit(e.returncode)
    except KeyboardInterrupt:
        sys.exit(130)
