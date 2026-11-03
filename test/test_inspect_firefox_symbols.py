import gzip
import hashlib
import importlib.util
import pathlib
import tempfile
import unittest


MODULE_PATH = pathlib.Path(__file__).resolve().parents[1] / "scripts" / "inspect-firefox-symbols.py"
SPEC = importlib.util.spec_from_file_location("inspect_firefox_symbols", MODULE_PATH)
INSPECTOR = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(INSPECTOR)

MODULE_ID = "TESTMODULE1"
JPEG = (
    "FILE 1 git:github.com/mozilla-firefox/firefox:media/libjpeg/src/jdapimin.c:revision\n"
    "FUNC 100 8 0 jpeg_CreateDecompress(jpeg_decompress_struct*, int)\n100 8 37 1\n"
    "FUNC 200 8 0 jpeg_read_header(jpeg_decompress_struct*, int)\n200 8 268 1\n"
)
SQLITE = (
    "FILE 2 git:github.com/mozilla-firefox/firefox:third_party/sqlite3/src/sqlite3.c:revision\n"
    "FUNC 300 8 0 sqlite3_open_v2(char const*, sqlite3**)\n300 8 11 2\n"
    "FUNC 400 8 0 sqlite3_prepare_v2(sqlite3*, char const*)\n400 8 22 2\n"
    "FUNC 500 8 0 sqlite3_step(sqlite3_stmt*)\n500 8 33 2\n"
)


class InspectorTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.symbols = pathlib.Path(self.directory.name) / "symbols.gz"

    def write_symbols(self, module, content, platform="windows"):
        with gzip.open(self.symbols, "wb") as target:
            target.write(f"MODULE {platform} x86_64 {MODULE_ID} {module}\n{content}".encode())
        return hashlib.sha256(self.symbols.read_bytes()).hexdigest()

    def test_jpeg_and_sqlite_function_sources(self):
        sha = self.write_symbols("xul.pdb", JPEG)
        jpeg = INSPECTOR.inspect(self.symbols, MODULE_ID, sha, "jpeg")
        self.assertEqual(jpeg["functions"]["jpeg_CreateDecompress"]["line"], 37)
        self.assertEqual(jpeg["functions"]["jpeg_read_header"]["functionAddress"], "200")
        sha = self.write_symbols("nss3.pdb", SQLITE)
        sqlite = INSPECTOR.inspect(self.symbols, MODULE_ID, sha, "sqlite")
        self.assertEqual(sqlite["functions"]["sqlite3_step"]["line"], 33)

    def test_linux_function_sources_and_platform_module_pair(self):
        sha = self.write_symbols("libxul.so", JPEG, "Linux")
        self.assertEqual(INSPECTOR.inspect(self.symbols, MODULE_ID, sha, "jpeg")["functions"]["jpeg_read_header"]["line"], 268)
        sha = self.write_symbols("libmozsqlite3.so", SQLITE, "Linux")
        self.assertEqual(INSPECTOR.inspect(self.symbols, MODULE_ID, sha, "sqlite")["functions"]["sqlite3_step"]["line"], 33)
        sha = self.write_symbols("libmozsqlite3.so", SQLITE)
        with self.assertRaisesRegex(ValueError, "module ID or name"):
            INSPECTOR.inspect(self.symbols, MODULE_ID, sha, "sqlite")

    def test_mismatched_hash_or_module_is_rejected(self):
        sha = self.write_symbols("xul.pdb", JPEG)
        with self.assertRaisesRegex(ValueError, "SHA-256"):
            INSPECTOR.inspect(self.symbols, MODULE_ID, "0" * 64, "jpeg")
        with self.assertRaisesRegex(ValueError, "module ID or name"):
            INSPECTOR.inspect(self.symbols, "OTHER", sha, "jpeg")
        with self.assertRaisesRegex(ValueError, "module ID or name"):
            INSPECTOR.inspect(self.symbols, MODULE_ID, sha, "sqlite")

    def test_missing_or_unrelated_source_is_rejected(self):
        sha = self.write_symbols("xul.pdb", JPEG.replace("200 8 268 1", "200 8 0 1"))
        with self.assertRaisesRegex(ValueError, "JPEG function-to-source"):
            INSPECTOR.inspect(self.symbols, MODULE_ID, sha, "jpeg")
        sha = self.write_symbols("xul.pdb", JPEG.replace("200 8 268 1", "300 8 268 1"))
        with self.assertRaisesRegex(ValueError, "JPEG function-to-source"):
            INSPECTOR.inspect(self.symbols, MODULE_ID, sha, "jpeg")
        sha = self.write_symbols("nss3.pdb", SQLITE.replace(":third_party/sqlite3/", ":unrelated/sqlite3/"))
        with self.assertRaisesRegex(ValueError, "SQLITE function-to-source"):
            INSPECTOR.inspect(self.symbols, MODULE_ID, sha, "sqlite")


if __name__ == "__main__":
    unittest.main()
