import base64
import importlib.util
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch


spec = importlib.util.spec_from_file_location("publisher", Path(__file__).with_name("publish-homebrew.py"))
publisher = importlib.util.module_from_spec(spec)
spec.loader.exec_module(publisher)


def response(document, status=200, returncode=0):
    body = json.dumps(document)
    return subprocess.CompletedProcess(
        ["gh", "api"],
        returncode,
        stdout=f"HTTP/2.0 {status} Test\r\nContent-Type: application/json\r\n\r\n{body}",
        stderr="",
    )


class PublisherTest(unittest.TestCase):
    def setUp(self):
        self.entries = {
            f"skp_1.2.3_{system}_{arch}.tar.gz": str(index) * 64
            for index, (system, arch) in enumerate(
                (("darwin", "arm64"), ("darwin", "amd64"), ("linux", "arm64"), ("linux", "amd64")), 1
            )
        }
        self.checksums = "\n".join(f"{digest}  {name}" for name, digest in self.entries.items())

    def test_formula_covers_all_platforms_with_matching_checksums(self):
        formula = publisher.render("v1.2.3", self.checksums)
        self.assertIn("class Skp < Formula", formula)
        self.assertIn('version "1.2.3"', formula)
        self.assertIn('license "MIT"', formula)
        self.assertIn('depends_on "git"', formula)
        self.assertIn('bin.install "skp"', formula)
        self.assertIn('shell_output("#{bin}/skp --version")', formula)
        self.assertEqual(formula.count("      url "), 4)
        for name, digest in self.entries.items():
            self.assertIn(f'/{name}"\n      sha256 "{digest}"', formula)

    def test_git_dependency_is_linux_only(self):
        formula = publisher.render("v1.2.3", self.checksums)
        before_linux, linux_and_after = formula.split("  on_linux do\n", 1)
        linux_block = linux_and_after.split("\n  end\n", 1)[0]
        self.assertNotIn('depends_on "git"', before_linux)
        self.assertIn('    depends_on "git"', linux_block)

    def test_invalid_inputs_fail_before_publication(self):
        for tag in ("v1.2.3-rc1", "1.2.3", 'v1.2.3"'):
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                publisher.render(tag, self.checksums)
        with self.assertRaises(ValueError):
            publisher.render("v1.2.3", self.checksums.replace("1" * 64, "invalid"))
        with self.assertRaises(ValueError):
            publisher.render("v1.2.3", self.checksums + "\n" + self.checksums.splitlines()[0])
        with self.assertRaises(KeyError):
            publisher.render("v1.2.3", "\n".join(self.checksums.splitlines()[:-1]))

    def test_identical_formula_does_not_write(self):
        existing = response({"sha": "old-sha", "content": base64.b64encode(b"formula").decode()})
        with patch.object(publisher.subprocess, "run", return_value=existing) as api:
            publisher.publish("v1.2.3", "formula")
        api.assert_called_once()

    def test_update_uses_existing_sha_and_exact_content(self):
        existing = response({"sha": "old-sha", "content": base64.b64encode(b"old").decode()})
        with patch.object(publisher.subprocess, "run", side_effect=[existing, response({})]) as api:
            publisher.publish("v1.2.3", "new\n")
        payload = json.loads(api.call_args_list[1].kwargs["input"])
        self.assertEqual(payload["sha"], "old-sha")
        self.assertEqual(payload["branch"], "main")
        self.assertEqual(base64.b64decode(payload["content"]), b"new\n")
        self.assertTrue(api.call_args_list[1].kwargs["check"])

    def test_missing_formula_is_created_without_sha(self):
        missing = response({"message": "Not Found"}, status=404, returncode=1)
        with patch.object(publisher.subprocess, "run", side_effect=[missing, response({})]) as api:
            publisher.publish("v1.2.3", "new\n")
        payload = json.loads(api.call_args_list[1].kwargs["input"])
        self.assertNotIn("sha", payload)
        self.assertEqual(payload["branch"], "main")
        self.assertEqual(base64.b64decode(payload["content"]), b"new\n")

    def test_api_failure_propagates(self):
        failure = response({"message": "server error"}, status=500, returncode=1)
        with patch.object(publisher.subprocess, "run", return_value=failure):
            with self.assertRaises(subprocess.CalledProcessError):
                publisher.publish("v1.2.3", "formula")

    def test_archive_invocations_do_not_publish(self):
        with patch("sys.argv", ["publisher", "--tag", "v1.2.3", "--artifact", "missing-archive.tar.gz"]), \
                patch.object(publisher, "publish") as publish:
            publisher.main()
        publish.assert_not_called()

    def test_dry_run_writes_generated_formula(self):
        with tempfile.TemporaryDirectory() as directory:
            artifact = Path(directory) / "checksums.txt"
            artifact.write_text(self.checksums)
            argv = ["publisher", "--tag", "v1.2.3", "--artifact", str(artifact), "--dry-run"]
            with patch("sys.argv", argv), patch.object(publisher, "publish") as publish:
                publisher.main()
            formula = artifact.parent / "homebrew" / "Formula" / "skp.rb"
            self.assertEqual(formula.read_text(), publisher.render("v1.2.3", self.checksums))
        publish.assert_not_called()


if __name__ == "__main__":
    unittest.main()
