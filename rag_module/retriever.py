"""
Retriever Module.
Hybrid BM25 + FAISS with Russian-aware tokenization, header enrichment,
and penalty for summary documents.
"""

import os
import re
import json
import numpy as np
import faiss
from rank_bm25 import BM25Okapi
from openai import OpenAI

client = OpenAI(
    base_url=os.getenv("LLM_BASE_URL", "http://localhost:8000/v1"),
    api_key=os.getenv("LLM_API_KEY", "dummy-key"),
)
EMB_MODEL = os.getenv("EMB_MODEL", "BAAI/bge-m3")

RU_STOP = {
    "и", "в", "во", "не", "что", "он", "на", "я", "с", "со", "как", "а", "то",
    "все", "она", "так", "его", "но", "да", "ты", "к", "у", "же", "вы", "за",
    "бы", "по", "только", "ее", "мне", "было", "вот", "от", "меня", "еще",
    "нет", "о", "из", "ему", "теперь", "когда", "даже", "ну", "вдруг", "ли",
    "если", "уже", "или", "ни", "быть", "был", "него", "до", "вас", "нибудь",
    "опять", "уж", "вам", "ведь", "там", "потом", "себя", "ничего", "ей",
    "может", "они", "тут", "где", "есть", "надо", "ней", "для", "мы", "тебя",
    "их", "чем", "была", "сам", "чтоб", "без", "будто", "чего", "раз", "тоже",
    "себе", "под", "будет", "ж", "тогда", "кто", "этот", "того", "потому",
    "этого", "какой", "совсем", "ним", "здесь", "этом", "один", "почти",
    "мой", "тем", "чтобы", "нее", "сейчас", "были", "куда", "зачем", "всех",
    "никогда", "можно", "при", "наконец", "два", "об", "другой", "хоть",
    "после", "над", "больше", "тот", "через", "эти", "нас", "про", "всего",
    "них", "какая", "много", "разве", "три", "эту", "моя", "впрочем", "хорошо",
    "свою", "этой", "перед", "иногда", "лучше", "чуть", "том", "нельзя",
    "такой", "им", "более", "всегда", "конечно", "всю", "между", "нам",
    "хочу", "хотим", "хотите", "хочет", "ваш", "ваша", "ваше", "вашего",
    "вашем", "ваших", "наш", "наша", "наше", "нашего", "ребят", "пожалуйста",
}

SOURCE_PENALTY = {
    "00-product-map.md": 0.55,
    "20-portfolio.md": 0.60,
    "01-numbers-and-indicators.md": 0.85,
    "26-general.md": 0.80,
}

_RU_ENDINGS = (
    "иями", "ями", "ами", "ов", "ев", "ей", "ах", "ях",
    "ий", "ый", "ая", "ое", "ые", "ой", "ей",
    "а", "я", "ы", "и", "у", "ю", "е", "о", "ь",
)


def stem(word: str) -> str:
    if len(word) <= 4:
        return word
    for end in _RU_ENDINGS:
        if word.endswith(end) and len(word) - len(end) >= 3:
            return word[: -len(end)]
    return word


def tokenize_ru(text: str) -> list:
    """Tokenize with weights: full word ×3, stem ×2, trigrams ×1."""
    text = text.lower()
    words = re.findall(r"[а-яёa-z0-9]+", text)
    tokens = []
    for w in words:
        if w in RU_STOP:
            continue
        tokens.extend([w] * 3)
        tokens.extend([stem(w)] * 2)
        if len(w) >= 5:
            for i in range(len(w) - 2):
                tokens.append(w[i:i + 3])
    return tokens


def load_chunks(chunks_path: str) -> list:
    if not os.path.exists(chunks_path):
        print(f"[retriever] ERROR: {chunks_path} not found.")
        return []
    with open(chunks_path, "r", encoding="utf-8") as f:
        return [json.loads(line) for line in f]


def enrich_for_bm25(chunk: dict) -> str:
    """
    Prepend header context to chunk text before BM25 indexing.
    This makes every chunk from 22-askue.md 'know' it belongs to the
    'Облачная платформа учёта' product, and match queries like
    'симки операторов' or 'расход по цехам'.
    """
    meta = chunk.get("metadata", {})
    headers = [meta.get(k) for k in ("Header1", "Header2", "Header3") if meta.get(k)]
    header_str = " ".join(headers)
    return f"{header_str}\n{chunk['text']}" if header_str else chunk["text"]


_local_model = None

def get_embeddings(texts: list) -> np.ndarray:
    """Get embeddings either from local model (dev) or API (platform)."""
    global _local_model
    
    api_key = os.getenv("LLM_API_KEY") or os.getenv("OPENAI_API_KEY")
    base_url = os.getenv("LLM_BASE_URL") or os.getenv("OPENAI_BASE_URL")
    
    if api_key and base_url and "localhost:9999" not in base_url:
        batch_size = 32
        all_embeddings = []
        for i in range(0, len(texts), batch_size):
            batch = texts[i:i + batch_size]
            response = client.embeddings.create(input=batch, model=EMB_MODEL)
            all_embeddings.extend([data.embedding for data in response.data])
        return np.array(all_embeddings, dtype=np.float32)
        
    else:
        if _local_model is None:
            print("[retriever] Loading local BAAI/bge-m3 model (this may take a minute)...")
            from sentence_transformers import SentenceTransformer
            _local_model = SentenceTransformer("BAAI/bge-m3")
        
        print(f"[retriever] Encoding {len(texts)} texts locally...")
        embeddings = _local_model.encode(texts, normalize_embeddings=True)
        return np.array(embeddings, dtype=np.float32)


class Retriever:
    def __init__(self, chunks_path: str = "rag_module/data/chunks.jsonl",
                 use_faiss: bool = True):
        self.chunks = load_chunks(chunks_path)
        if not self.chunks:
            raise ValueError("No chunks loaded.")
        self.use_faiss = use_faiss

        print("[retriever] Building BM25 index...")
        indexed_texts = [enrich_for_bm25(c) for c in self.chunks]
        self.bm25 = BM25Okapi([tokenize_ru(t) for t in indexed_texts])

        self.penalties = np.array([
            SOURCE_PENALTY.get(c["metadata"].get("source", ""), 1.0)
            for c in self.chunks
        ])

        if use_faiss:
            print("[retriever] Building FAISS index...")
            emb = get_embeddings([c["text"] for c in self.chunks])
            faiss.normalize_L2(emb)
            self.dim = emb.shape[1]
            self.faiss_index = faiss.IndexFlatIP(self.dim)
            self.faiss_index.add(emb)
            print(f"[retriever] FAISS: {self.faiss_index.ntotal} vectors.")

    def retrieve_documents(self, query: str, top_k: int = 3) -> list:
        
        bm25_scores = self.bm25.get_scores(tokenize_ru(query))
        bm25_scores = bm25_scores * self.penalties
        bm25_top = np.argsort(bm25_scores)[::-1][:top_k * 4]

        faiss_top = []
        if self.use_faiss:
            q_emb = get_embeddings([query])
            faiss.normalize_L2(q_emb)
            _, faiss_idx = self.faiss_index.search(q_emb, top_k * 4)
            faiss_top = faiss_idx[0].tolist()

        k = 60
        rrf = {}
        for rank, idx in enumerate(bm25_top):
            rrf[idx] = rrf.get(idx, 0.0) + 1.0 / (k + rank + 1)
        for rank, idx in enumerate(faiss_top):
            rrf[idx] = rrf.get(idx, 0.0) + 1.0 / (k + rank + 1)

        sorted_idx = sorted(rrf.keys(), key=lambda x: rrf[x], reverse=True)

        results = []
        for idx in sorted_idx[:top_k]:
            c = self.chunks[idx]
            results.append({
                "text": c["text"],
                "source": c["metadata"].get("source", "unknown"),
                "score": float(rrf[idx]),
            })
        return results