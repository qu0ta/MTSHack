import os
import csv
import glob
import re
import json
import zipfile
import tempfile
import numpy as np
from openai import OpenAI

DATA_DIR = os.getenv("DATA_DIR", "data")
LLM_MODEL = os.getenv("LLM_MODEL") or os.getenv("GEN_MODEL") or "Qwen3.8-27B"
EMB_MODEL = os.getenv("EMB_MODEL", "BAAI/bge-m3")
TEMPERATURE = float(os.getenv("TEMPERATURE", "0.1"))
MAX_TOKENS = int(os.getenv("MAX_TOKENS", "500"))
TOP_K = int(os.getenv("TOP_K", "3"))

LLM_BASE_URL = (
    os.getenv("LLM_BASE_URL")
    or os.getenv("OPENAI_BASE_URL")
    or "http://localhost:9999/v1"
)
LLM_API_KEY = (
    os.getenv("LLM_API_KEY")
    or os.getenv("OPENAI_API_KEY")
    or "sk-placeholder-for-local-dev"
)

client = OpenAI(base_url=LLM_BASE_URL, api_key=LLM_API_KEY)

LOCAL_MODE = LLM_BASE_URL == "http://localhost:9999/v1"
if LOCAL_MODE:
    print("[solution] LOCAL MODE: no API key — using BM25-only, LLM calls disabled")

RU_STOP = {
    "и", "в", "во", "не", "что", "на", "я", "с", "как", "а", "то", "все",
    "она", "так", "его", "но", "да", "к", "у", "же", "вы", "за", "бы", "по",
    "мне", "от", "меня", "о", "из", "ему", "когда", "если", "уже", "или",
    "может", "они", "тут", "где", "есть", "надо", "для", "мы", "их", "чем",
    "без", "чего", "раз", "тоже", "под", "будет", "тогда", "кто", "этот",
    "потому", "какой", "здесь", "один", "почти", "мой", "тем", "чтобы",
    "сейчас", "были", "куда", "зачем", "всех", "можно", "при", "два", "об",
    "другой", "после", "над", "больше", "тот", "через", "эти", "нас", "про",
    "всего", "них", "какая", "много", "три", "эту", "моя", "перед", "лучше",
    "том", "нельзя", "такой", "им", "более", "всегда", "между", "нам",
    "хочу", "хотим", "хотите", "хочет", "ваш", "ваша", "ваше", "наш", "наша",
}

_RU_ENDINGS = (
    "иями", "ями", "ами", "ов", "ев", "ей", "ах", "ях", "ий", "ый",
    "ая", "ое", "ые", "ой", "а", "я", "ы", "и", "у", "ю", "е", "о", "ь",
)


def _stem(word: str) -> str:
    if len(word) <= 4:
        return word
    for end in _RU_ENDINGS:
        if word.endswith(end) and len(word) - len(end) >= 3:
            return word[:-len(end)]
    return word


def _tokenize(text: str) -> list:
    words = re.findall(r"[а-яёa-z0-9]+", text.lower())
    toks = []
    for w in words:
        if w in RU_STOP:
            continue
        toks.extend([w] * 3)
        toks.extend([_stem(w)] * 2)
        if len(w) >= 5:
            for i in range(len(w) - 2):
                toks.append(w[i:i + 3])
    return toks


SOURCE_PENALTY = {
    "00-product-map.md": 0.55,
    "20-portfolio.md": 0.60,
    "01-numbers-and-indicators.md": 0.85,
    "26-general.md": 0.80,
}


def extract_kb() -> str:
    kb_zip = os.path.join(DATA_DIR, "knowledge-base.zip")
    if os.path.exists(kb_zip):
        extract_path = tempfile.mkdtemp(prefix="kb_")
        with zipfile.ZipFile(kb_zip) as z:
            z.extractall(extract_path)
        return extract_path
    
    fallback = "rag_module/data/raw"
    if os.path.exists(fallback):
        return fallback
    raise FileNotFoundError(f"No KB at {kb_zip}")


def chunk_kb(raw_folder: str) -> list:
    from langchain_text_splitters import (
        MarkdownHeaderTextSplitter,
        RecursiveCharacterTextSplitter,
    )
    md_splitter = MarkdownHeaderTextSplitter(
        [("#", "Header1"), ("##", "Header2"), ("###", "Header3")]
    )
    txt_splitter = RecursiveCharacterTextSplitter(chunk_size=450, chunk_overlap=60)

    chunks = []
    files = (
        glob.glob(f"{raw_folder}/**/*.md", recursive=True)
        + glob.glob(f"{raw_folder}/**/*.txt", recursive=True)
    )
    for fp in files:
        fname = os.path.basename(fp)
        with open(fp, "r", encoding="utf-8") as f:
            text = f.read()
        for ch in txt_splitter.split_documents(md_splitter.split_text(text)):
            chunks.append({
                "text": ch.page_content,
                "metadata": {**ch.metadata, "source": fname},
            })
    return chunks


def get_embeddings(texts: list) -> np.ndarray:
    if LOCAL_MODE:
        raise RuntimeError("Local mode: no embeddings API available")
    batch = 32
    out = []
    for i in range(0, len(texts), batch):
        r = client.embeddings.create(input=texts[i:i+batch], model=EMB_MODEL)
        out.extend([d.embedding for d in r.data])
    return np.array(out, dtype=np.float32)


def build_index(chunks: list, use_faiss: bool = True) -> dict:
    from rank_bm25 import BM25Okapi
    indexed = []
    for c in chunks:
        m = c["metadata"]
        headers = [m.get(k) for k in ("Header1", "Header2", "Header3") if m.get(k)]
        prefix = " ".join(headers)
        indexed.append(f"{prefix}\n{c['text']}" if prefix else c["text"])

    bm25 = BM25Okapi([_tokenize(t) for t in indexed])

    faiss_index = None
    if use_faiss:
        try:
            import faiss
            embs = get_embeddings([c["text"] for c in chunks])
            faiss.normalize_L2(embs)
            faiss_index = faiss.IndexFlatIP(embs.shape[1])
            faiss_index.add(embs)
        except Exception as e:
            print(f"[solution] FAISS failed: {e}")
            faiss_index = None

    return {"bm25": bm25, "chunks": chunks, "faiss": faiss_index}


def retrieve(query: str, index: dict, top_k: int = 3) -> list:
    bm25_scores = index["bm25"].get_scores(_tokenize(query))
    penalties = np.array([
        SOURCE_PENALTY.get(c["metadata"].get("source", ""), 1.0)
        for c in index["chunks"]
    ])
    bm25_scores = bm25_scores * penalties
    bm25_top = np.argsort(bm25_scores)[::-1][:top_k * 4]

    faiss_top = []
    if index["faiss"] is not None:
        try:
            import faiss
            q = get_embeddings([query])
            faiss.normalize_L2(q)
            _, idx = index["faiss"].search(q, top_k * 4)
            faiss_top = idx[0].tolist()
        except Exception:
            pass

    k = 60
    rrf = {}
    for rank, i in enumerate(bm25_top):
        rrf[i] = rrf.get(i, 0) + 1.0 / (k + rank + 1)
    for rank, i in enumerate(faiss_top):
        rrf[i] = rrf.get(i, 0) + 1.0 / (k + rank + 1)

    sorted_idx = sorted(rrf.keys(), key=lambda x: rrf[x], reverse=True)[:top_k]
    return [index["chunks"][i] for i in sorted_idx]


SYSTEM_PROMPT = """Ты — ИИ-консультант МТС EnergyTool.
Отвечай на вопрос пользователя, используя ТОЛЬКО предоставленный контекст.
Если в контексте нет ответа — скажи: "Я специализируюсь только на продуктах EnergyTool. Давайте передам вопрос специалисту."
Ответ: до 150 слов, простым языком, без технического жаргона.

ПРАВИЛА:
- НЕ говори, что МТС продаёт электроэнергию (МТС — не энергосбыт, а учёт и аналитика).
- НЕ называй итоговую стоимость — только индикативные ориентиры из контекста.
- НЕ обещай окупаемость/экономию без расчёта.
- Если вопрос требует точного расчёта или юридической оценки — предложи бесплатную оценку или специалиста.
- В конце задай ОДИН уточняющий вопрос (если он поможет лучше понять задачу клиента).

КОНТЕКСТ:
{context}"""


def generate_answer(question: str, chunks: list) -> str:
    
    if LOCAL_MODE:
        preview = chunks[0]["text"][:150].replace("\n", " ") if chunks else "N/A"
        return f"[LOCAL MOCK] Based on {chunks[0]['metadata'].get('source', '?')}: {preview}..."

    context = "\n\n---\n\n".join(
        f"[{c['metadata'].get('source', '?')}]\n{c['text']}" for c in chunks
    )
    prompt = SYSTEM_PROMPT.format(context=context)

    for attempt in range(3):
        try:
            resp = client.chat.completions.create(
                model=LLM_MODEL,
                messages=[
                    {"role": "system", "content": prompt},
                    {"role": "user", "content": question},
                ],
                temperature=TEMPERATURE,
                max_tokens=MAX_TOKENS,
            )
            return resp.choices[0].message.content.strip()
        except Exception as e:
            if attempt == 2:
                raise
            print(f"[solution] LLM error (attempt {attempt + 1}): {e}")

def find_questions_file():
    """Ищем questions.csv в нескольких местах."""
    candidates = [
        os.path.join(DATA_DIR, "questions.csv"),
        "questions.csv",
        os.path.join("data", "questions.csv"),
        "/tmp/questions.csv",
        "./questions.csv",
    ]
    for p in candidates:
        if os.path.exists(p):
            print(f"[solution] Found questions at: {p}")
            return p
    return None


def main():
    # 1. Ищем вопросы
    qpath = find_questions_file()
    if not qpath:
        print("[solution] ERROR: questions.csv not found anywhere!")
        print(f"[solution] DATA_DIR = {DATA_DIR}")
        if os.path.exists(DATA_DIR):
            print(f"[solution] Files in DATA_DIR: {os.listdir(DATA_DIR)}")
        print(f"[solution] Files in CWD: {os.listdir('.')}")
        # Создаем пустой submission, чтобы платформа не падала с другой ошибкой
        with open("submission.csv", "w", encoding="utf-8", newline="") as f:
            w = csv.writer(f, quoting=csv.QUOTE_ALL)
            w.writerow(["id", "answer"])
        return

    with open(qpath, "r", encoding="utf-8-sig") as f:
        questions = [dict(r) for r in csv.DictReader(f)]
    print(f"[solution] {len(questions)} questions from {qpath}")

    # 2. Строим базу знаний
    raw = extract_kb()
    chunks = chunk_kb(raw)
    print(f"[solution] {len(chunks)} chunks")

    # 3. Индекс
    index = build_index(chunks, use_faiss=True)
    mode = "BM25+FAISS" if index["faiss"] is not None else "BM25 only"
    print(f"[solution] Index ready ({mode})")

    # 4. Обрабатываем вопросы
    answers = []
    for q in questions:
        qid = q["id"]
        qtext = q["question"]
        try:
            ctx = retrieve(qtext, index, top_k=TOP_K)
            ans = generate_answer(qtext, ctx)
        except Exception as e:
            print(f"[solution] Q{qid} failed: {e}")
            ans = ("Извините, произошла техническая ошибка. "
                   "Пожалуйста, свяжитесь со специалистом МТС.")
        answers.append({"id": qid, "answer": ans})
        print(f"[solution] Q{qid}: {ans[:80]}...")

    # 5. Сохраняем
    with open("submission.csv", "w", encoding="utf-8", newline="") as f:
        w = csv.DictWriter(f, fieldnames=["id", "answer"], quoting=csv.QUOTE_ALL)
        w.writeheader()
        w.writerows(answers)
    print(f"[solution] ✓ submission.csv saved ({len(answers)} rows)")

if __name__ == "__main__":
    main()  