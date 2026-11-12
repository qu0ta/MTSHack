import threading
import time
from pathlib import Path

try:
    from . import solution as S
except ImportError:
    import solution as S

_lock = threading.Lock()
_state = {}


def init(kb_path=None):
    with _lock:
        if "index" in _state:
            return
        S.EXPAND_UNTIL = float("inf")
        path = Path(kb_path) if kb_path else S.find_kb()
        chunks = S.load_chunks(path)
        docs = dict(S.read_documents(path))
        with S.httpx.Client(timeout=S.TIMEOUT) as client:
            _state["index"] = S.Index(chunks, client, docs)


def _index():
    if "index" not in _state:
        init()
    return _state["index"]


def _turns(query, history):
    turns = [("assistant" if m["role"] == "assistant" else "user", m["content"]) for m in (history or [])]
    turns.append(("user", query))
    return turns


def _raw(turns):
    if len(turns) == 1:
        return turns[0][1]
    return "\n".join(("Пользователь: " if r == "user" else "Ассистент: ") + t for r, t in turns)


def _item(chunk, rank):
    return {"source": chunk["source"], "title": chunk["title"], "text": chunk["text"], "score": round(1.0 / (rank + 1), 4)}


def search(query, history=None, top_k=5):
    index = _index()
    turns = _turns(query, history)
    bm25_query, dense_query = S.build_queries(turns)
    with S.httpx.Client(timeout=S.TIMEOUT) as client:
        found = index.search(client, bm25_query, dense_query)
    return [_item(c, i) for i, c in enumerate(found[:top_k])]


def build_context(query, history=None):
    index = _index()
    turns = _turns(query, history)
    bm25_query, dense_query = S.build_queries(turns)
    with S.httpx.Client(timeout=S.TIMEOUT) as client:
        found = index.search(client, bm25_query, dense_query)
        full, rest = S.expand(index, found)
        rules, needs = S.extras(index, found, {name for name, _ in full})
        examples = S.pick_examples(index, client, _raw(turns))
    return {
        "chunks": [_item(c, i) for i, c in enumerate(found)],
        "documents": [{"source": name, "text": text} for name, text in full],
        "rules": [_item(c, i) for i, c in enumerate(rules)],
        "needs": [_item(c, i) for i, c in enumerate(needs)],
        "examples": examples,
    }


def generate_answer(query, history=None):
    index = _index()
    turns = _turns(query, history)
    started = time.perf_counter()
    with S.httpx.Client(timeout=S.TIMEOUT) as client:
        bm25_query, dense_query = S.build_queries(turns)
        found = index.search(client, bm25_query, dense_query)
        full, rest = S.expand(index, found)
        rules, needs = S.extras(index, found, {name for name, _ in full})
        examples = S.pick_examples(index, client, _raw(turns))
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
        "timings_ms": {
            "retrieval": round((searched - started) * 1000),
            "generation": round((finished - searched) * 1000),
            "total": round((finished - started) * 1000),
        },
    }