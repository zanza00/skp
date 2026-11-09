#!/usr/bin/env python3
"""GoReleaser publisher for the skp macOS/Linux Homebrew formula."""

import argparse
import base64
import json
from pathlib import Path
import re
import subprocess


ENDPOINT = "repos/zanza00/homebrew-tap/contents/Formula/skp.rb"


def render(tag, checksums):
    if not re.fullmatch(r"v\d+\.\d+\.\d+", tag):
        raise ValueError("Expected a stable vX.Y.Z tag")
    version = tag[1:]
    hashes = {}
    for line in checksums.splitlines():
        digest, name = line.split()
        if not re.fullmatch(r"[0-9a-f]{64}", digest):
            raise ValueError(f"Invalid SHA-256 for {name}")
        if name in hashes:
            raise ValueError(f"Duplicate checksum for {name}")
        hashes[name] = digest
    lines = [
        "class Skp < Formula",
        '  desc "Git-native Agent Skill stack switcher"',
        '  homepage "https://github.com/zanza00/skp"',
        f'  version "{version}"',
        '  license "MIT"',
    ]
    for platform, os_name in (("macos", "darwin"), ("linux", "linux")):
        lines.extend(["", f"  on_{platform} do"])
        for arch, go_arch in (("arm", "arm64"), ("intel", "amd64")):
            name = f"skp_{version}_{os_name}_{go_arch}.tar.gz"
            url = f"https://github.com/zanza00/skp/releases/download/{tag}/{name}"
            lines.extend([
                f"    on_{arch} do",
                f'      url "{url}"',
                f'      sha256 "{hashes[name]}"',
                "    end",
            ])
        if platform == "linux":
            lines.extend(["", '    depends_on "git"'])
        lines.append("  end")
    lines.extend([
        "",
        "  def install", '    bin.install "skp"', "  end", "",
        "  test do",
        '    assert_equal "skp", shell_output("#{bin}/skp --version").lines.first.strip',
        '    assert_match "use", shell_output("#{bin}/skp --help 2>&1")',
        "  end", "end", "",
    ])
    return "\n".join(lines)


def api_failure(result):
    return subprocess.CalledProcessError(
        result.returncode or 1,
        ["gh", "api", ENDPOINT],
        output=result.stdout,
        stderr=result.stderr,
    )


def existing_formula():
    result = subprocess.run(
        ["gh", "api", ENDPOINT, "-f", "ref=main", "--method", "GET", "--include"],
        text=True,
        capture_output=True,
    )
    first_line = result.stdout.splitlines()[0] if result.stdout else ""
    match = re.match(r"HTTP/\S+\s+(\d{3})\b", first_line)
    if not match:
        if result.returncode:
            raise api_failure(result)
        raise ValueError("GitHub API response has no HTTP status")
    status = int(match.group(1))
    if status == 404:
        return None
    if result.returncode or not 200 <= status < 300:
        raise api_failure(result)
    normalized = result.stdout.replace("\r\n", "\n")
    _, separator, body = normalized.partition("\n\n")
    if not separator:
        raise ValueError("GitHub API response has no JSON body")
    return json.loads(body)


def publish(tag, formula):
    existing = existing_formula()
    if existing is not None:
        current = base64.b64decode(existing["content"]).decode()
        if current == formula:
            print(f"Homebrew formula already matches {tag}")
            return
    payload = {
        "branch": "main",
        "message": f"chore: update skp to {tag}",
        "content": base64.b64encode(formula.encode()).decode(),
    }
    if existing is not None:
        payload["sha"] = existing["sha"]
    subprocess.run(
        ["gh", "api", ENDPOINT, "--method", "PUT", "--input", "-", "--silent"],
        input=json.dumps(payload), text=True, check=True,
    )
    print(f"Published Homebrew formula for {tag}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--artifact", type=Path, required=True)
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()
    # GoReleaser invokes publishers for each archive and the checksum artifact.
    if args.artifact.name != "checksums.txt":
        return
    formula = render(args.tag, args.artifact.read_text())
    destination = args.artifact.parent / "homebrew" / "Formula" / "skp.rb"
    destination.parent.mkdir(parents=True, exist_ok=True)
    destination.write_text(formula)
    if args.dry_run:
        print(destination)
    else:
        publish(args.tag, formula)


if __name__ == "__main__":
    main()
