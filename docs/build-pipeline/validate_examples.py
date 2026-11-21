#!/usr/bin/env python3
"""Validate the M0 YAML examples against their structural and plan contracts.

Requires PyYAML and jsonschema. Catalog import must additionally check the
installed executor and command registries before accepting a spec.
"""

from __future__ import annotations

import hashlib
import json
import re
from pathlib import Path

import jsonschema
import yaml
from referencing import Registry, Resource


ROOT = Path(__file__).resolve().parent
SCHEMAS = ROOT / "schemas"
EXAMPLES = ROOT / "examples"
SCHEMA_FILES = {
    "CoreSpec": "core-spec.schema.json",
    "TransformSpec": "transform-spec.schema.json",
    "BuildProfile": "build-profile.schema.json",
}
JSON_TAGS = {
    "tag:yaml.org,2002:map",
    "tag:yaml.org,2002:seq",
    "tag:yaml.org,2002:str",
    "tag:yaml.org,2002:bool",
    "tag:yaml.org,2002:int",
    "tag:yaml.org,2002:float",
    "tag:yaml.org,2002:null",
}
MAX_SAFE_INTEGER = 9007199254740991


class StrictLoader(yaml.SafeLoader):
    def compose_node(self, parent, index):
        if self.check_event(yaml.AliasEvent):
            raise ValueError("YAML aliases are not permitted")
        return super().compose_node(parent, index)

    def construct_object(self, node, deep=False):
        if node.tag not in JSON_TAGS:
            raise ValueError(f"YAML tag {node.tag!r} is not permitted")
        if isinstance(node, yaml.ScalarNode):
            value = node.value
            if node.tag.endswith(":bool") and value not in ("true", "false"):
                raise ValueError(f"ambiguous YAML boolean {value!r}; quote it")
            if node.tag.endswith(":null") and value != "null":
                raise ValueError(f"ambiguous YAML null {value!r}; quote it")
            if node.tag.endswith(":int") and not re.fullmatch(r"-?(?:0|[1-9][0-9]*)", value):
                raise ValueError(f"ambiguous YAML integer {value!r}; quote it")
            if node.tag.endswith(":float") and not re.fullmatch(
                r"-?(?:0|[1-9][0-9]*)\.[0-9]+(?:[eE][+-]?[0-9]+)?", value
            ):
                raise ValueError(f"ambiguous YAML number {value!r}; quote it")
        result = super().construct_object(node, deep=deep)
        if type(result) is int and abs(result) > MAX_SAFE_INTEGER:
            raise ValueError("YAML integer exceeds I-JSON safe range")
        return result

    def construct_mapping(self, node, deep=False):
        result = {}
        folded_keys = {}
        for key_node, value_node in node.value:
            if key_node.tag != "tag:yaml.org,2002:str":
                raise ValueError("YAML mapping keys must be strings")
            key = self.construct_object(key_node, deep=deep)
            if key == "<<":
                raise ValueError("YAML merge keys are not permitted")
            if key in result:
                raise ValueError(f"duplicate YAML key {key!r}")
            folded = key.casefold()
            if folded in folded_keys:
                raise ValueError(f"case-equivalent YAML keys {folded_keys[folded]!r} and {key!r}")
            folded_keys[folded] = key
            result[key] = self.construct_object(value_node, deep=deep)
        return result


def load_yaml(path: Path) -> dict:
    with path.open(encoding="utf-8") as stream:
        documents = list(yaml.load_all(stream, Loader=StrictLoader))
    if len(documents) != 1 or not isinstance(documents[0], dict):
        raise ValueError(f"{path}: expected one YAML mapping document")
    return documents[0]


def canonical_example_bytes(value: object) -> bytes:
    """The examples contain only JSON's integer, string, bool, list, and map types.

    For those types this encoding is RFC 8785 compatible. The production
    catalog must use full RFC 8785 for specs with finite non-integer numbers.
    """
    def reject_float(item: object) -> None:
        if isinstance(item, float):
            raise ValueError("example digest checker needs full RFC 8785 for non-integer numbers")
        if isinstance(item, list):
            for child in item:
                reject_float(child)
        if isinstance(item, dict):
            for child in item.values():
                reject_float(child)

    reject_float(value)
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode("utf-8")


def check_digest(spec: dict) -> None:
    digest = spec["metadata"]["digest"]
    unsigned = dict(spec)
    unsigned["metadata"] = dict(spec["metadata"])
    del unsigned["metadata"]["digest"]
    calculated = "sha256:" + hashlib.sha256(canonical_example_bytes(unsigned)).hexdigest()
    if digest != calculated:
        raise ValueError(f"{spec['metadata']['name']}: digest {digest} != {calculated}")


def check_option_value(definition: dict, value: object, label: str) -> None:
    name = definition["name"]
    option_type = definition["type"]
    matches = {
        "boolean": lambda: type(value) is bool,
        "string": lambda: isinstance(value, str),
        "integer": lambda: type(value) is int,
        "number": lambda: type(value) in (int, float),
        "enum": lambda: isinstance(value, str) and value in definition["values"],
        "string-list": lambda: isinstance(value, list) and all(isinstance(item, str) for item in value),
        "secret-ref": lambda: isinstance(value, dict) and value.get("kind") == "secret",
        "input-artifact-ref": lambda: isinstance(value, dict) and value.get("kind") == "input-artifact",
    }[option_type]()
    if not matches:
        raise ValueError(f"{label}: invalid value type for {name}")
    if "minimum" in definition and value < definition["minimum"]:
        raise ValueError(f"{label}: {name} below minimum")
    if "maximum" in definition and value > definition["maximum"]:
        raise ValueError(f"{label}: {name} above maximum")


def check_definitions(definitions: list[dict], label: str) -> None:
    names = [definition["name"] for definition in definitions]
    if len(set(names)) != len(names):
        raise ValueError(f"{label}: duplicate option definition")
    for definition in definitions:
        if "minimum" in definition and "maximum" in definition and definition["minimum"] > definition["maximum"]:
            raise ValueError(f"{label}: invalid range for {definition['name']}")
        if "default" in definition:
            check_option_value(definition, definition["default"], label)


def check_options(assignments: list[dict], definitions: list[dict], label: str) -> None:
    by_name = {definition["name"]: definition for definition in definitions}
    seen = set()
    for assignment in assignments:
        name = assignment["name"]
        if name in seen:
            raise ValueError(f"{label}: duplicate option assignment {name}")
        seen.add(name)
        if name not in by_name:
            raise ValueError(f"{label}: unknown option {name}")
        definition = by_name[name]
        value = assignment["value"]
        check_option_value(definition, value, label)
    missing = [item["name"] for item in definitions if item["required"] and item["name"] not in seen]
    if missing:
        raise ValueError(f"{label}: missing required options {missing}")


def spec_key(spec: dict) -> tuple[str, str, str]:
    metadata = spec["metadata"]
    return metadata["name"], metadata["version"], metadata["digest"]


def resolve_spec(reference: dict, specs: dict[tuple[str, str, str], dict]) -> dict:
    key = reference["name"], reference["version"], reference["digest"]
    try:
        return specs[key]
    except KeyError as error:
        raise ValueError(f"unknown spec reference {key}") from error


def match_target(target: dict, capability: dict) -> bool:
    return target in capability["targets"]


def check_profile(profile: dict, specs: dict[tuple[str, str, str], dict]) -> None:
    name = profile["metadata"]["name"]
    slots = {slot["name"]: slot for slot in profile.get("perBuildInputs", [])}
    if len(slots) != len(profile.get("perBuildInputs", [])):
        raise ValueError(f"{name}: duplicate per-build input slot")
    def check_references(assignments: list[dict]) -> None:
        for assignment in assignments:
            value = assignment["value"]
            if isinstance(value, dict):
                slot = slots.get(value["name"])
                if slot is None or slot["kind"] != value["kind"]:
                    raise ValueError(f"{name}: unbound per-build input {value}")

    core = resolve_spec(profile["core"], specs)
    if core["kind"] != "CoreSpec":
        raise ValueError(f"{name}: core reference is not a CoreSpec")
    check_options(profile["coreOptions"], core["options"], name)
    check_references(profile["coreOptions"])
    protocols = {endpoint["protocol"] for endpoint in profile["c2"]["endpoints"]}
    protocols.update(profile["c2"].get("includeProtocols", []))
    matches = [
        capability for capability in core["capabilities"]
        if match_target(profile["target"], capability)
        and capability["mode"] == profile["mode"]
        and protocols.issubset(set(capability["protocols"]))
        and capability["output"]["artifact"] == profile["coreOutput"]
    ]
    if len(matches) != 1:
        raise ValueError(f"{name}: core capability matching returned {len(matches)} rows")
    current = profile["coreOutput"]
    for step in profile["transforms"]:
        transform = resolve_spec(step["spec"], specs)
        if transform["kind"] != "TransformSpec":
            raise ValueError(f"{name}: transform reference is not a TransformSpec")
        check_options(step["options"], transform["options"], name)
        check_references(step["options"])
        matching_edges = [
            capability for capability in transform["capabilities"]
            if match_target(profile["target"], capability) and capability["input"] == current
        ]
        if len(matching_edges) != 1:
            raise ValueError(f"{name}: transform {transform['metadata']['name']} has no unique matching edge")
        current = matching_edges[0]["output"]
    if current != profile["finalOutput"]["artifact"]:
        raise ValueError(f"{name}: final artifact differs from pipeline output")
    if not profile["transforms"] and profile["finalOutput"]["format"] not in matches[0]["output"]["operatorFormats"]:
        raise ValueError(f"{name}: direct output format is not advertised by core")


def main() -> None:
    schema_docs = {path.name: json.loads(path.read_text(encoding="utf-8")) for path in SCHEMAS.glob("*.schema.json")}
    for schema in schema_docs.values():
        jsonschema.Draft202012Validator.check_schema(schema)
    registry = Registry()
    for name, schema in schema_docs.items():
        resource = Resource.from_contents(schema)
        registry = registry.with_resource(name, resource)
        registry = registry.with_resource((SCHEMAS / name).as_uri(), resource)
    specs = {}
    name_versions = {}
    profiles = []
    for path in sorted(EXAMPLES.glob("*.yaml")):
        document = load_yaml(path)
        schema_name = SCHEMA_FILES[document["kind"]]
        schema = schema_docs[schema_name]
        jsonschema.Draft202012Validator(schema, registry=registry).validate(document)
        if document["kind"] == "BuildProfile":
            profiles.append(document)
        else:
            check_digest(document)
            check_definitions(document["options"], document["metadata"]["name"])
            key = spec_key(document)
            if key in specs:
                raise ValueError(f"duplicate spec {key}")
            name_version = key[:2]
            if name_version in name_versions and name_versions[name_version] != key[2]:
                raise ValueError(f"conflicting spec content for {name_version}")
            name_versions[name_version] = key[2]
            specs[key] = document
    for profile in profiles:
        check_profile(profile, specs)
    print(f"validated {len(specs)} immutable specs and {len(profiles)} profiles")


if __name__ == "__main__":
    main()
