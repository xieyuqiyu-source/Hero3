#!/usr/bin/env python3
"""Bundle local OpenAPI YAML refs into one Apifox-importable file."""

from __future__ import annotations

import argparse
from pathlib import Path
from typing import Any

import yaml


def load_yaml(path: Path) -> Any:
    with path.open("r", encoding="utf-8") as file:
        return yaml.safe_load(file)


def lookup_fragment(document: Any, fragment: str) -> Any:
    if fragment in ("", "#"):
        return document

    current = document
    for part in fragment.removeprefix("#/").split("/"):
        key = part.replace("~1", "/").replace("~0", "~")
        current = current[key]
    return current


def resolve_refs(value: Any, current_file: Path, root_file: Path) -> Any:
    if isinstance(value, list):
        return [resolve_refs(item, current_file, root_file) for item in value]

    if not isinstance(value, dict):
        return value

    ref = value.get("$ref")
    if isinstance(ref, str) and not ref.startswith("#/"):
        ref_file, _, fragment = ref.partition("#")
        target_file = (current_file.parent / ref_file).resolve()
        target_doc = load_yaml(target_file)
        target_value = lookup_fragment(target_doc, f"#{fragment}" if fragment else "")
        resolved = resolve_refs(target_value, target_file, root_file)
        siblings = {key: item for key, item in value.items() if key != "$ref"}
        if siblings and isinstance(resolved, dict):
            return {**resolved, **resolve_refs(siblings, current_file, root_file)}
        return resolved

    return {
        key: resolve_refs(item, current_file, root_file)
        for key, item in value.items()
    }


def validate_openapi(document: dict[str, Any]) -> None:
    required = ("openapi", "info", "paths")
    missing = [key for key in required if key not in document]
    if missing:
        raise ValueError(f"OpenAPI 缺少必需字段: {', '.join(missing)}")

    paths = document.get("paths", {})
    if not isinstance(paths, dict) or not paths:
        raise ValueError("OpenAPI paths 不能为空")


def main() -> None:
    parser = argparse.ArgumentParser(description="Bundle Hero3 OpenAPI files.")
    parser.add_argument("--input", default="docs/openapi/openapi.yaml")
    parser.add_argument("--output", default="docs/openapi.bundle.yaml")
    args = parser.parse_args()

    input_path = Path(args.input).resolve()
    output_path = Path(args.output).resolve()

    document = load_yaml(input_path)
    bundled = resolve_refs(document, input_path, input_path)
    validate_openapi(bundled)

    output_path.write_text(
        yaml.safe_dump(bundled, allow_unicode=True, sort_keys=False),
        encoding="utf-8",
    )
    print(f"Bundled {len(bundled['paths'])} paths -> {output_path}")


if __name__ == "__main__":
    main()
