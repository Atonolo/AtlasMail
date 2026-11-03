#!/usr/bin/env python3
"""Identify bundled component functions in an official Electron symbol archive."""

import argparse
import json
import re
import sys
import zipfile


FFT_1D = re.compile(rb"[/\\]fft2d[/\\]src[/\\]fftsg\.c$", re.IGNORECASE)
FFT_2D = re.compile(rb"[/\\]fft2d[/\\]src[/\\]fftsg2d\.c$", re.IGNORECASE)
JPEG_MIN = re.compile(rb"[/\\]libjpeg_turbo[/\\]src[/\\]jdapimin\.c$", re.IGNORECASE)
JPEG_STD = re.compile(rb"[/\\]libjpeg_turbo[/\\]src[/\\]jdapistd\.c$", re.IGNORECASE)
CHROMIUM_SQLITE = re.compile(
    rb"[/\\]third_party[/\\]sqlite[/\\]src[/\\]amalgamation[/\\]sqlite3\.c$", re.IGNORECASE
)
NODE_SQLITE = re.compile(
    rb"[/\\]third_party[/\\]electron_node[/\\]deps[/\\]sqlite[/\\]sqlite3\.c$", re.IGNORECASE
)
COMPONENTS = {
    "fft": {b"cdft": FFT_1D, b"rdft": FFT_1D, b"rdft2d": FFT_2D},
    "jpeg": {
        b"chromium_jpeg_CreateDecompress": JPEG_MIN,
        b"chromium_jpeg_read_header": JPEG_MIN,
        b"chromium_jpeg_read_scanlines": JPEG_STD,
    },
    "sqlite": {b"chrome_sqlite3_step": CHROMIUM_SQLITE, b"sqlite3_step": NODE_SQLITE},
}
SYMBOL_NAMES = ("/electron.sym", "/electron.exe.sym", "/Electron Framework.sym")


def inspect(archive_path, expected_id, component="fft"):
    expected_functions = COMPONENTS[component]
    with zipfile.ZipFile(archive_path) as archive:
        entries = [entry for entry in archive.infolist() if entry.filename.endswith(SYMBOL_NAMES)]
        if len(entries) != 1:
            raise ValueError("Expected exactly one Electron executable or framework symbol file")
        with archive.open(entries[0]) as symbols:
            module = symbols.readline().decode("utf-8").strip().split(" ", 4)
            if len(module) != 5 or module[0] != "MODULE":
                raise ValueError("Unrecognized Breakpad module header")
            if expected_id and module[3] != expected_id:
                raise ValueError("Symbol module ID does not match the expected executable ID")

            sources = {}
            functions = {}
            current = None
            for line in symbols:
                if line.startswith(b"FILE "):
                    fields = line.rstrip().split(b" ", 2)
                    if len(fields) == 3 and any(
                        pattern.search(fields[2]) for pattern in expected_functions.values()
                    ):
                        sources[fields[1]] = fields[2]
                elif line.startswith(b"FUNC "):
                    fields = line.rstrip().split(b" ", 4)
                    current = fields[4] if len(fields) == 5 and fields[4] in expected_functions else None
                elif current:
                    fields = line.split()
                    source = sources.get(fields[3]) if len(fields) == 4 else None
                    if source is not None and expected_functions[current].search(source):
                        line_number = int(fields[2])
                        if line_number <= 0:
                            continue
                        functions[current.decode()] = {
                            "source": source.decode("utf-8", "replace"),
                            "line": line_number,
                            "address": fields[0].decode("ascii"),
                        }
                        current = None
                        if len(functions) == len(expected_functions):
                            break
            if set(functions) != {name.decode() for name in expected_functions}:
                raise ValueError(f"The expected {component.upper()} function-to-source mappings were not all found")
            return {"module": " ".join(module[1:]), "functions": functions}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("archive", help="Electron platform symbols ZIP")
    parser.add_argument("--expected-id", help="Breakpad module ID matched to the shipped executable")
    parser.add_argument("--component", choices=sorted(COMPONENTS), default="fft")
    args = parser.parse_args()
    try:
        result = inspect(args.archive, args.expected_id, args.component)
    except (OSError, ValueError, UnicodeError, zipfile.BadZipFile) as error:
        print(f"Symbol inspection failed: {error}", file=sys.stderr)
        return 1
    print(json.dumps(result, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
