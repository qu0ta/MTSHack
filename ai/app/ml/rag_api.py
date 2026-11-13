import os
import re
import threading
import time
from pathlib import Path

try:
    from . import solution as S
except ImportError:
    import solution as S

VERSION = "2026-11-13-site1"
if getattr(S, "VERSION", None) != VERSION:
    raise ImportError(f"rag_api {VERSION} не совпадает с solution {getattr(S, 'VERSION', None)}: обновите оба файла вместе")

BUDGET_SECONDS = float(os.getenv("RAG_BUDGET_SECONDS", "45"))
MAX_HISTORY_MESSAGES = int(os.getenv("RAG_MAX_HISTORY", "12"))
RETRIEVAL_TURNS = int(os.getenv("RAG_RETRIEVAL_TURNS", "6"))
PARAMS_MARK = re.compile(r"^\s*(справочные параметры|параметры объекта)", re.IGNORECASE)

_lock = threading.Lock()
_state = {}


class _Budget:
    def __enter__(self):
        S._local.deadline = time.time() + BUDGET_SECONDS

    def __exit__(self, *exc):
        S._local.deadline = None


def init(kb_path=None, site_mode=True):
    with _lock:
        if "index" in _state:
            return
        path = Path(kb_path or os.getenv("KB_PATH") or S.find_kb())
        chunks = S.load_chunks(path)
        docs = dict(S.read_documents(path))
        with S.httpx.Client(timeout=S.TIMEOUT) as client:
            _state["index"] = S.Index(chunks, client, docs)
        if site_mode:
            S.enable_site_mode()


def health():
    index = _state.get("index")
    return {"ready": index is not None, "degraded": list(index.degraded) if index else []}


def _index():
    if "index" not in _state:
        init()
    return _state["index"]


def _split(query, history):
    msgs = [("assistant" if m.get("role") == "assistant" else "user", str(m.get("content", ""))) for m in (history or [])]
    msgs = [m for m in msgs if m[1].strip()]
    params = []
    if msgs and msgs[0][0] == "user" and PARAMS_MARK.match(msgs[0][1]):
        params, msgs = [msgs[0]], msgs[1:]
    dialog = msgs[-MAX_HISTORY_MESSAGES:] + [("user", query)]
    return params + dialog, dialog[-(RETRIEVAL_TURNS + 1):]


def _raw(turns):
    if len(turns) == 1:
        return turns[0][1]
    return "\n".join(("Пользователь: " if r == "user" else "Ассистент: ") + t for r, t in turns)


def _item(chunk, rank):
    return {"source": chunk["source"], "title": chunk["title"], "text": chunk["text"], "score": round(1.0 / (rank + 1), 4)}


def search(query, history=None, top_k=5):
    index = _index()
    _, retrieval = _split(query, history)
    bm25_query, dense_query = S.build_queries(retrieval)
    with _Budget(), S.httpx.Client(timeout=S.TIMEOUT) as client:
        found = index.search(client, bm25_query, dense_query)
    return [_item(c, i) for i, c in enumerate(found[:top_k])]


def build_context(query, history=None):
    index = _index()
    _, retrieval = _split(query, history)
    flags = []
    bm25_query, dense_query = S.build_queries(retrieval)
    with _Budget(), S.httpx.Client(timeout=S.TIMEOUT) as client:
        found = index.search(client, bm25_query, dense_query, flags)
        full, rest = S.expand(index, found)
        rules, needs = S.extras(index, found, {name for name, _ in full})
        examples = S.pick_examples(index, client, _raw(retrieval), flags=flags)
    return {
        "chunks": [_item(c, i) for i, c in enumerate(found)],
        "documents": [{"source": name, "text": text} for name, text in full],
        "rules": [_item(c, i) for i, c in enumerate(rules)],
        "needs": [_item(c, i) for i, c in enumerate(needs)],
        "examples": examples,
        "degraded": sorted(set(index.degraded + flags)),
    }


def generate_answer(query, history=None):
    index = _index()
    turns, retrieval = _split(query, history)
    flags = []
    started = time.perf_counter()
    with _Budget(), S.httpx.Client(timeout=S.TIMEOUT) as client:
        bm25_query, dense_query = S.build_queries(retrieval)
        found = index.search(client, bm25_query, dense_query, flags)
        full, rest = S.expand(index, found)
        rules, needs = S.extras(index, found, {name for name, _ in full})
        raw = _raw(retrieval)
        examples = S.pick_examples(index, client, raw, flags=flags)
        searched = time.perf_counter()
        answer = S.generate(client, turns, rest, rules, needs, full, examples, _raw(turns))
    finished = time.perf_counter()
    sources = []
    for source in [name for name, _ in full] + [c["source"] for c in found]:
        if source not in sources:
            sources.append(source)
    return {
        "answer": answer,
        "sources": sources,
        "documents": [_item(c, i) for i, c in enumerate(found)],
        "degraded": sorted(set(index.degraded + flags)),
        "timings_ms": {
            "retrieval": round((searched - started) * 1000),
            "generation": round((finished - searched) * 1000),
            "total": round((finished - started) * 1000),
        },
    }
