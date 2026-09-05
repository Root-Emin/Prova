# Prova ML araçları

Bu klasör iki küçük ve bağımsız sorumluluk taşır:

- `generation/`: 46 insan tarafından hazırlanmış seed kaydından deterministik
  Türkçe behavioral dataset üretir.
- `persona_service/`: Go backend’in canlı role-play akışı için kullandığı
  mock/vLLM uyumlu persona HTTP servisini çalıştırır.

## Dataset ve model kaynak gerçekliği

Üretilen 6.000 kayıt Hugging Face dataset reposunda tutulur:

- Dataset: `eminkutlu/prova-dataset`
- BERTurk QLoRA adapter: `eminkutlu/prova-berturk-qlora`

Repository içinde büyük JSONL dataset kopyası veya model checkpoint’i
tutulmaz. Generator çıktıları yerel olarak `backend/ml/generated/` altına
yazılır ve `.gitignore` içindedir.

## Dataset üretimi

```bash
python3 backend/ml/generation/generate_dataset.py
python3 backend/ml/validate_dataset.py
```

Özel çıktı veya seed yolu vermek için:

```bash
python3 backend/ml/generation/generate_dataset.py \
  --seed-dataset backend/ml/generation/seed_dataset.jsonl \
  --output-dir backend/ml/generated
python3 backend/ml/validate_dataset.py \
  --data-dir backend/ml/generated \
  --labels backend/ml/generation/labels.json
```

Generator; split sızıntısını, duplicate’leri, evidence span’lerini, taxonomy
uyumunu ve kalite alanlarını kontrol eden `validate_dataset.py` ile birlikte
kullanılır. Seed kayıtları `generation/seed_dataset.jsonl` dosyasındadır;
üretilmiş datasetin tamamı HF üzerinde kaynak kabul edilir.

## Persona servisi

Mock provider ile GPU gerektirmeden:

```bash
python3 backend/ml/persona_service/server.py --host 127.0.0.1 --port 8090
curl http://127.0.0.1:8090/health
```

Gerçek canlı role-play için OpenAI-compatible vLLM endpoint’i:

```bash
vllm serve Qwen/Qwen3-4B-Instruct-2507 --port 8000
AI_PERSONA_PROVIDER=vllm \
PERSONA_LLM_BASE_URL=http://localhost:8000/v1 \
python3 backend/ml/persona_service/server.py
```

Oturum sonu değerlendirme ayrı backend LLM profiliyle
`Qwen/Qwen3-30B-A3B-Instruct-2507` modeline gider. BERTurk QLoRA eğitimi
Colab’da tamamlanmış ve adapter Hugging Face’te tutulmaktadır; bu klasör
training/checkpoint pipeline’ı içermez.

## Test

```bash
PYTHONPATH=backend/ml python3 -m unittest discover \
  -s backend/ml/persona_service -p 'test_*.py'
python3 backend/ml/validate_dataset.py
```
