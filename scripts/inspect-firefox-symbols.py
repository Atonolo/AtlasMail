#!/usr/bin/env python3
"""Verify function-to-source mappings in a matched Mozilla Breakpad symbol file."""

import argparse
import gzip
import hashlib
import json
import re
import sys
import zlib


COMPONENTS = {
    "jpeg": {
        "modules": {("windows", "xul.pdb"), ("linux", "libxul.so")},
        "source": re.compile(rb":media/libjpeg/src/jdapimin\.c:"),
        "functions": ("jpeg_CreateDecompress", "jpeg_read_header"),
    },
    "sqlite": {
        "modules": {("windows", "nss3.pdb"), ("linux", "libmozsqlite3.so")},
        "source": re.compile(rb":third_party/sqlite3/src/sqlite3\.c:"),
        "functions": ("sqlite3_open_v2", "sqlite3_prepare_v2", "sqlite3_step"),
    },
}


def inspect(filename, expected_id, expected_sha256, component):
    settings = COMPONENTS[component]
    hasher = hashlib.sha256()
    with open(filename, "rb") as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b""):
            hasher.update(chunk)
    digest = hasher.hexdigest()
    if digest != expected_sha256.lower():
        raise ValueError("Symbol file SHA-256 does not match")
    expected = {name.encode() for name in settings["functions"]}
    with gzip.open(filename, "rb") as symbols:
        module = symbols.readline().decode("utf-8").strip().split(" ", 4)
        if len(module) != 5 or module[0] != "MODULE":
            raise ValueError("Unrecognized Breakpad module header")
        if module[3] != expected_id or (module[1].lower(), module[4].lower()) not in settings["modules"]:
            raise ValueError("Symbol module ID or name does not match the binary")
        sources = {}
        functions = {}
        current = None
        start = None
        end = None
        for line in symbols:
            if line.startswith(b"FILE "):
                fields = line.rstrip().split(b" ", 2)
                if len(fields) == 3 and settings["source"].search(fields[2]):
                    sources[fields[1]] = fields[2]
            elif line.startswith(b"FUNC "):
                fields = line.rstrip().split(b" ", 4)
                name = fields[4].split(b"(", 1)[0] if len(fields) == 5 else None
                current = name if name in expected else None
                start = fields[1] if current else None
                end = int(fields[1], 16) + int(fields[2], 16) if current else None
            elif current:
                fields = line.split()
                if len(fields) != 4:
                    continue
                source = sources.get(fields[3])
                address = int(fields[0], 16) if re.fullmatch(rb"[0-9a-fA-F]+", fields[0]) else None
                if (source is not None and address is not None and int(start, 16) <= address < end
                        and fields[2].isdigit() and int(fields[2]) > 0):
                    functions[current.decode()] = {
                        "source": source.decode("utf-8", "replace"),
                        "line": int(fields[2]),
                        "functionAddress": start.decode("ascii"),
                    }
                    current = None
                    if len(functions) == len(expected):
                        break
    if set(functions) != {name.decode() for name in expected}:
        raise ValueError(f"The expected {component.upper()} function-to-source mappings were not all found")
    return {"module": " ".join(module[1:]), "sha256": digest, "functions": functions}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("symbols", help="gzip-compressed Mozilla Breakpad symbol file")
    parser.add_argument("--expected-id", required=True, help="Module ID matched to the shipped binary")
    parser.add_argument("--sha256", required=True, help="Expected SHA-256 of the compressed symbol file")
    parser.add_argument("--component", choices=sorted(COMPONENTS), required=True)
    args = parser.parse_args()
    try:
        result = inspect(args.symbols, args.expected_id, args.sha256, args.component)
    except (OSError, EOFError, ValueError, UnicodeError, zlib.error) as error:
        print(f"Symbol inspection failed: {error}", file=sys.stderr)
        return 1
    print(json.dumps(result, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
