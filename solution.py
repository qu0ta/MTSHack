import os
import csv
import re
import glob
import zipfile
import tempfile
import numpy as np
from openai import OpenAI

DATA_DIR = os.getenv("DATA_DIR", "data")
LLM_BASE_URL = os.getenv("LLM_BASE_URL") or os.getenv("OPENAI_BASE_URL") or "http://localhost:9999/v1"
LLM_API_KEY = os.getenv("LLM_API_KEY") or os.getenv("OPENAI_API_KEY") or "sk-placeholder"
LLM_MODEL = os.getenv("LLM_MODEL") or os.getenv("GEN_MODEL") or "Qwen3.8-27B"
EMB_MODEL = os.getenv("EMB_MODEL", "BAAI/bge-m3")
TEMPERATURE = float(os.getenv("TEMPERATURE", "0.1"))
MAX_TOKENS = int(os.getenv("MAX_TOKENS", "500"))
TOP_K = int(os.getenv("TOP_K", "3"))
LOCAL_MODE = (LLM_BASE_URL == "http://localhost:9999/v1")
client = OpenAI(base_url=LLM_BASE_URL, api_key=LLM_API_KEY)

RU_STOP = {"и", "в", "во", "не", "что", "он", "на", "я", "с", "со", "как", "а", "то", "все", "она", "так", "его", "но", "да", "ты", "к", "у", "же", "вы", "за", "бы", "по", "только", "ее", "мне", "было", "вот", "от", "меня", "еще", "нет", "о", "из", "ему", "теперь", "когда", "даже", "ну", "вдруг", "ли", "если", "уже", "или", "ни", "быть", "был", "него", "до", "вас", "нибудь", "опять", "уж", "вам", "ведь", "там", "потом", "себя", "ничего", "ей", "может", "они", "тут", "где", "есть", "надо", "ней", "для", "мы", "тебя", "их", "чем", "была", "сам", "чтоб", "без", "будто", "чего", "раз", "тоже", "себе", "под", "будет", "ж", "тогда", "кто", "этот", "того", "потому", "этого", "какой", "совсем", "ним", "здесь", "этом", "один", "почти", "мой", "тем", "чтобы", "нее", "сейчас", "были", "куда", "зачем", "всех", "никогда", "можно", "при", "наконец", "два", "об", "другой", "хоть", "после", "над", "больше", "тот", "через", "эти", "нас", "про", "всего", "них", "какая", "много", "разве", "три", "эту", "моя", "впрочем", "хорошо", "свою", "этой", "перед", "иногда", "лучше", "чуть", "том", "нельзя", "такой", "им", "более", "всегда", "конечно", "всю", "между", "нам", "хочу", "хотим", "хотите", "хочет", "ваш", "ваша", "ваше", "вашего", "вашем", "ваших", "наш", "наша", "наше", "нашего", "ребят", "пожалуйста"}
_RU_ENDINGS = ("иями", "ями", "ами", "ов", "ев", "ей", "ах", "ях", "ий", "ый", "ая", "ое", "ые", "ой", "а", "я", "ы", "и", "у", "ю", "е", "о", "ь")

def _stem(word):
    if len(word) <= 4: return word
    for end in _RU_ENDINGS:
        if word.endswith(end) and len(word) - len(end) >= 3:
            return word[:-len(end)]
    return word

def _tokenize(text):
    words = re.findall(r"[а-яёa-z0-9]+", text.lower())
    toks = []
    for w in words:
        if w in RU_STOP: continue
        toks.extend([w] * 3)
        toks.extend([_stem(w)] * 2)
        if len(w) >= 5:
            for i in range(len(w) - 2):
                toks.append(w[i:i + 3])
    return toks

SOURCE_PENALTY = {"00-product-map.md": 0.55, "20-portfolio.md": 0.60, "01-numbers-and-indicators.md": 0.85, "26-general.md": 0.80}

def extract_kb():
    for path in [os.path.join(DATA_DIR, "knowledge-base.zip"), "knowledge-base.zip"]:
        if os.path.exists(path):
            extract_path = tempfile.mkdtemp(prefix="kb_")
            with zipfile.ZipFile(path) as z:
                z.extractall(extract_path)
            return extract_path
    if os.path.exists("rag_module/data/raw"):
        return "rag_module/data/raw"
    raise FileNotFoundError("knowledge-base.zip not found")

def chunk_kb(raw_folder):
    from langchain_text_splitters import MarkdownHeaderTextSplitter, RecursiveCharacterTextSplitter
    md_splitter = MarkdownHeaderTextSplitter([("#", "Header1"), ("##", "Header2"), ("###", "Header3")])
    txt_splitter = RecursiveCharacterTextSplitter(chunk_size=450, chunk_overlap=60)
    chunks = []
    files = glob.glob(f"{raw_folder}/**/*.md", recursive=True) + glob.glob(f"{raw_folder}/**/*.txt", recursive=True)
    for fp in files:
        fname = os.path.basename(fp)
        with open(fp, "r", encoding="utf-8") as f:
            text = f.read()
        for ch in txt_splitter.split_documents(md_splitter.split_text(text)):
            chunks.append({"text": ch.page_content, "metadata": {**ch.metadata, "source": fname}})
    return chunks

def get_embeddings(texts):
    if LOCAL_MODE: raise RuntimeError("Local mode")
    batch = 32
    out = []
    for i in range(0, len(texts), batch):
        r = client.embeddings.create(input=texts[i:i+batch], model=EMB_MODEL)
        out.extend([d.embedding for d in r.data])
    return np.array(out, dtype=np.float32)

def build_index(chunks):
    from rank_bm25 import BM25Okapi
    indexed = []
    for c in chunks:
        headers = [c["metadata"].get(k) for k in ("Header1", "Header2", "Header3") if c["metadata"].get(k)]
        prefix = " ".join(headers)
        indexed.append(f"{prefix}\n{c['text']}" if prefix else c["text"])
    bm25 = BM25Okapi([_tokenize(t) for t in indexed])
    faiss_index = None
    if not LOCAL_MODE:
        try:
            import faiss
            embs = get_embeddings([c["text"] for c in chunks])
            faiss.normalize_L2(embs)
            faiss_index = faiss.IndexFlatIP(embs.shape[1])
            faiss_index.add(embs)
        except Exception as e:
            print(f"[solution] FAISS failed: {e}")
    return {"bm25": bm25, "chunks": chunks, "faiss": faiss_index}

def retrieve(query, index, top_k=3):
    bm25_scores = index["bm25"].get_scores(_tokenize(query))
    penalties = np.array([SOURCE_PENALTY.get(c["metadata"].get("source", ""), 1.0) for c in index["chunks"]])
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
        except Exception: pass
    k = 60
    rrf = {}
    for rank, i in enumerate(bm25_top): rrf[i] = rrf.get(i, 0) + 1.0 / (k + rank + 1)
    for rank, i in enumerate(faiss_top): rrf[i] = rrf.get(i, 0) + 1.0 / (k + rank + 1)
    sorted_idx = sorted(rrf.keys(), key=lambda x: rrf[x], reverse=True)[:top_k]
    return [index["chunks"][i] for i in sorted_idx]

SYSTEM_PROMPT = """Ты — ИИ-консультант МТС EnergyTool.
Отвечай на вопрос, используя ТОЛЬКО контекст ниже. Если ответа нет — скажи: "Я специализируюсь только на продуктах EnergyTool. Давайте передам вопрос специалисту." До 150 слов. Без выдумок, без итоговых цен, без обещаний окупаемости.

КОНТЕКСТ:
{context}"""

def generate_answer(question, chunks):
    if LOCAL_MODE:
        return f"[MOCK] Based on {chunks[0]['metadata'].get('source', '?')}: {chunks[0]['text'][:100]}..."
    context = "\n\n---\n\n".join(f"[{c['metadata'].get('source', '?')}]\n{c['text']}" for c in chunks)
    prompt = SYSTEM_PROMPT.format(context=context)
    try:
        resp = client.chat.completions.create(
            model=LLM_MODEL,
            messages=[{"role": "system", "content": prompt}, {"role": "user", "content": question}],
            temperature=TEMPERATURE, max_tokens=MAX_TOKENS,
        )
        ans = resp.choices[0].message.content.strip()
    except Exception as e:
        print(f"[solution] LLM error: {e}")
        ans = "Извините, произошла техническая ошибка. Свяжитесь со специалистом МТС."
    if not ans.strip():
        ans = "Не могу ответить на этот вопрос. Свяжитесь со специалистом МТС."
    # Clean dangerous chars
    ans = ans.replace("\r", " ").replace("\x00", "").strip()
    return ans[:9000]


def main():
    # --- DIAGNOSTICS: log what's in CWD and DATA_DIR ---
    print("[solution] === DIAGNOSTICS ===")
    print(f"[solution] CWD: {os.getcwd()}")
    print(f"[solution] CWD files: {sorted(os.listdir('.'))[:30]}")
    if os.path.exists(DATA_DIR):
        print(f"[solution] DATA_DIR files: {sorted(os.listdir(DATA_DIR))[:30]}")
    else:
        print(f"[solution] DATA_DIR ({DATA_DIR}) does not exist")
    print("[solution] ====================")

    # --- Find questions.csv ---
    candidates = [
        os.path.join(DATA_DIR, "questions.csv"),
        "/tmp/questions.csv",
        "questions.csv",
    ]
    qpath = None
    for p in candidates:
        if os.path.exists(p):
            qpath = p
            break
    if not qpath:
        print("[solution] ERROR: no questions.csv found. Writing empty fallback.")
        with open("submission.csv", "w", encoding="utf-8", newline="") as f:
            f.write("id,answer\n")
        return

    print(f"[solution] Using questions from: {qpath}")

    # --- Read questions ---
    questions = []
    with open(qpath, "r", encoding="utf-8-sig", newline="") as f:
        for row in csv.DictReader(f):
            clean = {k.strip().lower(): (v or "").strip() for k, v in row.items() if k}
            if clean.get("id") and clean.get("question"):
                questions.append(clean)
    # Deduplicate by id (keep first)
    seen = set()
    unique_questions = []
    for q in questions:
        if q["id"] not in seen:
            seen.add(q["id"])
            unique_questions.append(q)
    questions = unique_questions
    print(f"[solution] Loaded {len(questions)} unique questions")

    # --- KB + index ---
    raw = extract_kb()
    chunks = chunk_kb(raw)
    print(f"[solution] Chunks: {len(chunks)}")
    index = build_index(chunks)

    # --- Answers ---
    answers = []
    for q in questions:
        try:
            ctx = retrieve(q["question"], index, top_k=TOP_K)
            ans = generate_answer(q["question"], ctx)
        except Exception as e:
            print(f"[solution] Q{q['id']} error: {e}")
            ans = "Извините, произошла техническая ошибка. Свяжитесь со специалистом МТС."
        answers.append({"id": q["id"], "answer": ans})

    # --- Write CSV: MINIMAL quoting, \n line endings, UTF-8 ---
    with open("submission.csv", "w", encoding="utf-8", newline="") as f:
        writer = csv.DictWriter(
            f,
            fieldnames=["id", "answer"],
            quoting=csv.QUOTE_MINIMAL,
            lineterminator="\n",
        )
        writer.writeheader()
        writer.writerows(answers)

    # --- Sanity check output ---
    size = os.path.getsize("submission.csv")
    print(f"[solution] ✓ submission.csv written: {len(answers)} rows, {size} bytes")
    with open("submission.csv", "r", encoding="utf-8") as f:
        print(f"[solution] First 2 lines:\n{f.readline()}{f.readline()}")


if __name__ == "__main__":
    main()