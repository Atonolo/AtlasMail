import importlib.util
import pathlib
import tempfile
import unittest
import zipfile


MODULE_PATH = pathlib.Path(__file__).resolve().parents[1] / "scripts" / "inspect-electron-symbols.py"
SPEC = importlib.util.spec_from_file_location("inspect_electron_symbols", MODULE_PATH)
INSPECTOR = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(INSPECTOR)

MODULE_ID = "TESTMODULE1"
HEADER = f"MODULE windows x86_64 {MODULE_ID} electron.exe\n"
FFT = (
    "FILE 1 C:\\src\\third_party\\fft2d\\src\\fftsg.c\n"
    "FILE 2 C:\\src\\third_party\\fft2d\\src\\fftsg2d.c\n"
    "FUNC 100 8 0 cdft\n100 8 11 1\n"
    "FUNC 200 8 0 rdft\n200 8 22 1\n"
    "FUNC 300 8 0 rdft2d\n300 8 33 2\n"
)
JPEG = (
    "FILE 3 ../../third_party/libjpeg_turbo/src/jdapimin.c\n"
    "FILE 4 ../../third_party/libjpeg_turbo/src/jdapistd.c\n"
    "FUNC 400 8 0 chromium_jpeg_CreateDecompress\n400 1 0 3\n401 7 44 3\n"
    "FUNC 500 8 0 chromium_jpeg_read_header\n500 7 55 3\n507 1 0 3\n"
    "FUNC 600 8 0 chromium_jpeg_read_scanlines\n600 8 66 4\n"
)
SQLITE = (
    "FILE 5 ../../third_party/sqlite/src/amalgamation/sqlite3.c\n"
    "FILE 6 ../../third_party/electron_node/deps/sqlite/sqlite3.c\n"
    "FUNC 700 8 0 chrome_sqlite3_step\n700 8 77 5\n"
    "FUNC 800 8 0 sqlite3_step\n800 8 88 6\n"
)


class InspectorTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.archive = pathlib.Path(self.directory.name) / "symbols.zip"

    def write_archive(self, symbols, duplicate=False):
        with zipfile.ZipFile(self.archive, "w") as archive:
            archive.writestr("module/electron.exe.sym", symbols)
            if duplicate:
                archive.writestr("other/electron.sym", symbols)

    def test_fft_jpeg_and_sqlite_function_sources(self):
        self.write_archive(HEADER + FFT + JPEG + SQLITE)
        fft = INSPECTOR.inspect(self.archive, MODULE_ID)
        jpeg = INSPECTOR.inspect(self.archive, MODULE_ID, "jpeg")
        sqlite = INSPECTOR.inspect(self.archive, MODULE_ID, "sqlite")
        self.assertEqual(fft["functions"]["rdft2d"]["line"], 33)
        self.assertTrue(fft["functions"]["rdft2d"]["source"].endswith("fftsg2d.c"))
        self.assertEqual(jpeg["functions"]["chromium_jpeg_CreateDecompress"]["line"], 44)
        self.assertEqual(jpeg["functions"]["chromium_jpeg_read_header"]["line"], 55)
        self.assertEqual(jpeg["functions"]["chromium_jpeg_read_scanlines"]["line"], 66)
        self.assertTrue(jpeg["functions"]["chromium_jpeg_read_scanlines"]["source"].endswith("jdapistd.c"))
        self.assertEqual(sqlite["functions"]["chrome_sqlite3_step"]["line"], 77)
        self.assertEqual(sqlite["functions"]["sqlite3_step"]["line"], 88)
        self.assertIn("electron_node", sqlite["functions"]["sqlite3_step"]["source"])

    def test_mismatched_module_is_rejected(self):
        self.write_archive(HEADER + FFT)
        with self.assertRaisesRegex(ValueError, "module ID does not match"):
            INSPECTOR.inspect(self.archive, "OTHER")

    def test_missing_mapping_is_rejected(self):
        self.write_archive(HEADER + FFT.replace("300 8 33 2", "300 8 33 99"))
        with self.assertRaisesRegex(ValueError, "FFT function-to-source mappings"):
            INSPECTOR.inspect(self.archive, MODULE_ID)
        with self.assertRaisesRegex(ValueError, "JPEG function-to-source mappings"):
            INSPECTOR.inspect(self.archive, MODULE_ID, "jpeg")
        with self.assertRaisesRegex(ValueError, "SQLITE function-to-source mappings"):
            INSPECTOR.inspect(self.archive, MODULE_ID, "sqlite")

    def test_function_must_match_its_specific_source(self):
        self.write_archive(HEADER + JPEG.replace("600 8 66 4", "600 8 66 3"))
        with self.assertRaisesRegex(ValueError, "JPEG function-to-source mappings"):
            INSPECTOR.inspect(self.archive, MODULE_ID, "jpeg")

    def test_duplicate_symbol_files_are_rejected(self):
        self.write_archive(HEADER + FFT, duplicate=True)
        with self.assertRaisesRegex(ValueError, "exactly one"):
            INSPECTOR.inspect(self.archive, MODULE_ID)

    def test_unrecognized_module_header_is_rejected(self):
        self.write_archive("NOT_A_MODULE\n" + FFT)
        with self.assertRaisesRegex(ValueError, "module header"):
            INSPECTOR.inspect(self.archive, MODULE_ID)


if __name__ == "__main__":
    unittest.main()
