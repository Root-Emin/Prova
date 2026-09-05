#!/usr/bin/env python3
"""Validate the generated corpus and its split artifacts."""

import argparse
import json
import re
import sys
import unicodedata
from collections import Counter, defaultdict
from difflib import SequenceMatcher
from pathlib import Path


ROOT = Path(__file__).resolve().parent
DEFAULT_DATA_DIR = ROOT / "generated"
DEFAULT_LABELS = ROOT / "generation" / "labels.json"
ALLOWED_TYPES = {
    "positive", "negative", "mixed", "neutral", "ambiguous",
    "hard_negative", "multi_label",
}
ALLOWED_SPLITS = {"train", "validation", "test"}
ID_PATTERN = re.compile(r"^tr_[0-9]{6}$")
DIFFICULTIES = {"kolay", "orta", "zor"}
AMBIGUITIES = {"düşük", "orta", "yüksek"}
NATURALNESS = {"yüksek", "orta"}
CONTROL_CHARS = re.compile(r"[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]")


def fail(message: str) -> None:
    print(f"HATA: {message}", file=sys.stderr)
    raise SystemExit(1)


def normalize(text: str) -> str:
    text = unicodedata.normalize("NFKC", text).lower()
    text = re.sub(r"[^\wçğıöşüâîû]+", " ", text, flags=re.UNICODE)
    return re.sub(r"\s+", " ", text).strip()


def near_duplicate_count(records: list[dict]) -> int:
    buckets: defaultdict[tuple[int, str, str], list[str]] = defaultdict(list)
    count = 0
    for record in records:
        key = normalize(record["context"] + " [UTTERANCE] " + record["utterance"])
        bucket = (len(key) // 40, key[:2], key[-2:])
        for previous in buckets[bucket]:
            if previous != key and SequenceMatcher(None, previous, key).ratio() >= 0.92:
                count += 1
                break
        buckets[bucket].append(key)
    return count


def read_records(path: Path) -> list[dict]:
    try:
        lines = path.read_text(encoding="utf-8").splitlines()
    except UnicodeDecodeError as exc:
        fail(f"UTF-8 okunamadı: {path}: {exc}")
    records = []
    for line_number, line in enumerate(lines, 1):
        if not line.strip():
            continue
        try:
            record = json.loads(line)
        except json.JSONDecodeError as exc:
            fail(f"{path.name} satır {line_number} JSON değil: {exc}")
        if not isinstance(record, dict):
            fail(f"{path.name} satır {line_number}: kayıt nesne olmalı")
        records.append(record)
    return records


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--min-samples", type=int, default=5000)
    parser.add_argument("--data-dir", type=Path, default=DEFAULT_DATA_DIR)
    parser.add_argument("--labels", type=Path, default=DEFAULT_LABELS)
    args = parser.parse_args()
    data_dir = args.data_dir.resolve()
    dataset = data_dir / "dataset.jsonl"
    labels = args.labels.resolve()
    taxonomy = json.loads(labels.read_text(encoding="utf-8"))
    allowed_labels = {item["key"] for item in taxonomy["labels"]}
    if len(allowed_labels) != 8:
        fail(f"taxonomy sekiz kriter içermiyor: {len(allowed_labels)}")
    records = read_records(dataset)
    seen_ids = set()
    seen_inputs = set()
    split_ids: defaultdict[str, set[str]] = defaultdict(set)
    families: defaultdict[str, set[str]] = defaultdict(set)
    groups: defaultdict[str, list[dict]] = defaultdict(list)

    for item in records:
        record_id = item.get("id")
        if record_id in seen_ids:
            fail(f"tekrarlı id: {record_id}")
        seen_ids.add(record_id)

        required = {"id", "scenario", "context", "utterance", "labels",
                    "negative_labels", "evidence_spans", "quality",
                    "example_type", "split"}
        missing = required - item.keys()
        if missing:
            fail(f"{record_id}: eksik alanlar: {sorted(missing)}")
        if not isinstance(record_id, str) or not ID_PATTERN.fullmatch(record_id):
            fail(f"{record_id}: id tr_###### biçiminde olmalı")
        if item["example_type"] not in ALLOWED_TYPES:
            fail(f"{record_id}: geçersiz example_type")
        if item["split"] not in ALLOWED_SPLITS:
            fail(f"{record_id}: geçersiz split")
        scenario = item["scenario"]
        if not isinstance(scenario, dict):
            fail(f"{record_id}: scenario nesne olmalı")
        required_scenario = {"industry", "scenario_type", "persona", "difficulty"}
        if set(scenario) != required_scenario:
            fail(f"{record_id}: scenario alanları hatalı")
        if any(not isinstance(scenario[key], str) or not scenario[key].strip()
               for key in required_scenario - {"difficulty"}):
            fail(f"{record_id}: scenario metin alanları boş olamaz")
        if scenario["difficulty"] not in DIFFICULTIES:
            fail(f"{record_id}: geçersiz difficulty")
        if not isinstance(item["context"], str) or not isinstance(item["utterance"], str):
            fail(f"{record_id}: context ve utterance metin olmalı")
        if not item["context"].strip() or not item["utterance"].strip():
            fail(f"{record_id}: context ve utterance boş olamaz")
        if CONTROL_CHARS.search(item["context"] + item["utterance"]):
            fail(f"{record_id}: kontrol karakteri içeriyor")
        if "�" in item["context"] + item["utterance"]:
            fail(f"{record_id}: bozuk Unicode replacement karakteri içeriyor")

        if not isinstance(item["labels"], list) or not all(isinstance(value, str) for value in item["labels"]):
            fail(f"{record_id}: labels string dizisi olmalı")
        if not isinstance(item["negative_labels"], list) or not all(isinstance(value, str) for value in item["negative_labels"]):
            fail(f"{record_id}: negative_labels string dizisi olmalı")
        positive = set(item["labels"])
        negative = set(item["negative_labels"])
        if len(positive) != len(item["labels"]) or len(negative) != len(item["negative_labels"]):
            fail(f"{record_id}: labels tekrar içeriyor")
        unknown = (positive | negative) - allowed_labels
        if unknown:
            fail(f"{record_id}: taxonomy dışı etiketler: {sorted(unknown)}")
        if item["example_type"] == "neutral" and (positive or negative):
            fail(f"{record_id}: neutral örnekte etiket olamaz")
        quality = item["quality"]
        if not isinstance(quality, dict) or set(quality) != {"ambiguity", "naturalness", "hard_negative", "context_dependent"}:
            fail(f"{record_id}: quality alanları hatalı")
        if quality["ambiguity"] not in AMBIGUITIES or quality["naturalness"] not in NATURALNESS:
            fail(f"{record_id}: quality seviyesi hatalı")
        if not isinstance(quality["hard_negative"], bool) or not isinstance(quality["context_dependent"], bool):
            fail(f"{record_id}: quality boolean alanları hatalı")
        if item["example_type"] == "hard_negative" and not quality["hard_negative"]:
            fail(f"{record_id}: hard_negative kalite işareti true olmalı")
        if not isinstance(item["evidence_spans"], list):
            fail(f"{record_id}: evidence_spans dizi olmalı")
        for span in item["evidence_spans"]:
            if not isinstance(span, dict) or set(span) != {"label", "text"}:
                fail(f"{record_id}: evidence span alanları hatalı")
            if span["label"] not in allowed_labels:
                fail(f"{record_id}: span etiketi taxonomy dışı")
            if not isinstance(span["text"], str) or not span["text"].strip():
                fail(f"{record_id}: evidence span metni boş olamaz")
            if span["text"] not in item["utterance"]:
                fail(f"{record_id}: span utterance içinde bulunmuyor: {span['text']!r}")
            if span["label"] not in positive | negative:
                fail(f"{record_id}: span etiketi labels/negative_labels içinde yok")
        if "contrastive_group" in item and not isinstance(item["contrastive_group"], str):
            fail(f"{record_id}: contrastive_group metin olmalı")
        for optional in ("focus_label", "template_family"):
            if optional in item and (not isinstance(item[optional], str) or not item[optional].strip()):
                fail(f"{record_id}: {optional} boş olamaz")

        input_key = normalize(item["context"] + " [UTTERANCE] " + item["utterance"])
        if input_key in seen_inputs:
            fail(f"exact duplicate: {record_id}")
        seen_inputs.add(input_key)
        split_ids[item["split"]].add(record_id)
        if item.get("template_family"):
            families[item["template_family"]].add(item["split"])
        if item.get("contrastive_group"):
            groups[item["contrastive_group"]].append(item)

    if not records:
        fail("dataset boş")
    if len(records) < args.min_samples:
        fail(f"TOTAL_SAMPLES={len(records)} < {args.min_samples}")
    all_types = {item["example_type"] for item in records}
    missing_types = ALLOWED_TYPES - all_types
    if missing_types:
        fail(f"örnek kategorileri eksik: {sorted(missing_types)}")
    splits = Counter(item["split"] for item in records)
    if len(splits) < 3:
        fail("train, validation ve test split'lerinin üçü de bulunmalı")

    leaked = {group: sorted({item["split"] for item in group_records}) for group, group_records in groups.items()
              if len({item["split"] for item in group_records}) > 1}
    if leaked:
        fail(f"contrastive grup split sızıntısı: {leaked}")
    malformed_groups = {group: len(items) for group, items in groups.items() if len(items) != 2}
    if malformed_groups:
        fail(f"contrastive gruplar tam çift değil: {malformed_groups}")
    family_leaks = {family: sorted(split_set) for family, split_set in families.items() if len(split_set) > 1}
    if family_leaks:
        fail(f"template-family split sızıntısı: {family_leaks}")

    for split in ALLOWED_SPLITS:
        split_path = data_dir / f"{split}.jsonl"
        if not split_path.exists():
            fail(f"split dosyası eksik: {split_path.name}")
        split_records = read_records(split_path)
        if any(record["split"] != split for record in split_records):
            fail(f"{split_path.name}: split alanı ile dosya adı uyuşmuyor")
        if {record["id"] for record in split_records} != split_ids[split]:
            fail(f"{split_path.name}: dataset.jsonl ile kayıt kümesi uyuşmuyor")

    if not (0.78 <= splits["train"] / len(records) <= 0.82 and
            0.08 <= splits["validation"] / len(records) <= 0.12 and
            0.08 <= splits["test"] / len(records) <= 0.12):
        fail(f"split oranları 80/10/10 dışında: {dict(splits)}")

    signal_counts = Counter(label for record in records for label in set(record["labels"]) | set(record["negative_labels"]))
    low = {label: signal_counts[label] for label in allowed_labels if signal_counts[label] < len(records) * 0.05}
    high = {label: signal_counts[label] for label in allowed_labels if signal_counts[label] > len(records) * 0.45}
    if low or high:
        fail(f"class imbalance: düşük={low}, yüksek={high}")

    near = near_duplicate_count(records)
    if near:
        fail(f"near duplicate bulundu (eşik .92): {near}")

    type_counts = Counter(item["example_type"] for item in records)
    if any(type_counts[k] < len(records) * 0.05 for k in ALLOWED_TYPES):
        fail(f"örnek kategorisi %5'in altında: {dict(type_counts)}")

    manifest = data_dir / "generation_manifest.json"
    if not manifest.exists():
        fail("generation_manifest.json eksik")
    manifest_data = json.loads(manifest.read_text(encoding="utf-8"))
    if manifest_data.get("generated_sample_count", 0) + manifest_data.get("retained_seed_samples", 0) != len(records):
        fail("manifest sample sayısı dataset ile uyuşmuyor")
    if manifest_data.get("validation_status") not in {"pending_validator", "passed"}:
        fail("manifest validation_status geçersiz")

    # Marking the manifest only after every check above succeeds keeps the
    # status factual even if validation is interrupted halfway through.
    manifest_data["validation_status"] = "passed"
    manifest.write_text(json.dumps(manifest_data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    print(f"OK: {len(records)} kayıt doğrulandı")
    print("Türler:", dict(type_counts))
    print("Splitler:", dict(splits))
    print("Etiketler:", dict(sorted(signal_counts.items())))
    print("Exact duplicates: 0")
    print(f"Near duplicates: {near}")
    print(f"Contrastive pairs: {len(groups)}")


if __name__ == "__main__":
    main()
