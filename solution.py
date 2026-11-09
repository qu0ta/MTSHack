import csv
import os
import re
import sys
import time
import zipfile
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

import httpx
import numpy as np
from rank_bm25 import BM25Okapi


def env(name, default=""):
    value = os.getenv(name, "").strip()
    return default if not value or value == name else value


def number(name, default, lower, upper, kind=int):
    try:
        return max(lower, min(upper, kind(env(name))))
    except ValueError:
        return default


DATA_DIR = Path(env("DATA_DIR", "data"))
BASE_URL = (env("LLM_BASE_URL") or env("BASE_URL") or env("OPENAI_BASE_URL")).rstrip("/")
LLM_KEY = env("LLM_API_KEY") or env("OPENAI_API_KEY")
EMB_KEY = env("EMBEDDER_API_KEY") or LLM_KEY
RERANK_KEY = env("RERANKER_API_KEY") or LLM_KEY
LLM_MODEL = env("LLM_MODEL") or env("GEN_MODEL") or env("OPENAI_MODEL", "qwen/qwen3.8-27b")
EMB_MODEL = env("EMB_MODEL", "baai/bge-m3")
RERANKER_MODEL = env("RERANKER_MODEL", "qwen/qwen3-reranker-8b")
TEMPERATURE = number("TEMPERATURE", 0.0, 0.0, 2.0, float)
MAX_TOKENS = number("MAX_TOKENS", 1500, 600, 8192)
MAX_WORKERS = number("MAX_WORKERS", 4, 1, 8)

CHUNK_CHARS = 1100
CANDIDATES = 24
RERANK_POOL = 12
FINAL_K = 5
DEADLINE = time.time() + 17 * 60

FALLBACK = (
    "Не нашёл точного ответа в базе знаний EnergyTool. Предлагаю передать вопрос специалисту: "
    "оставьте контакты, и менеджер уточнит детали вашей задачи."
)

SYSTEM_PROMPT = """Ты — ИИ-консультант сайта МТС EnergyTool (учёт и снижение затрат на электроэнергию, солнечные станции, поиск коммерческих потерь, выход на оптовый рынок).

Факты о продуктах, ценах, условиях и ограничениях бери ТОЛЬКО из фрагментов базы знаний ниже. Ситуацию клиента бери из диалога. Свои знания об энергетике не добавляй.

Как отвечать:
1. Начни с прямого ответа на вопрос (например «Да, …», «Нет, …», «Зависит от …»), затем 1–3 предложения с фактами из базы. Простым языком, без канцелярита.
2. Если для точной рекомендации не хватает данных, закончи ОДНИМ конкретным уточняющим вопросом о самом важном параметре (регион, присоединённая мощность, тип кровли, ценовая категория, приборы учёта, профиль потребления и т.п.). Не спрашивай то, что клиент уже сообщил в диалоге.
3. Если вопрос закрытый и уточнять нечего, вопрос не задавай.
4. Если клиент не готов оставлять контакты, не настаивай: объясни, какие данные нужны для оценки.
5. Если в базе нет ответа, скажи об этом прямо, ничего не выдумывай и предложи связаться со специалистом. Если вопрос вне тематики EnergyTool, вежливо скажи, что консультируешь только по продуктам EnergyTool.
6. Длина: обычно 50–110 слов, не больше 150.

Жёсткие правила:
- Не путай мощность (кВт, МВт) и потребление (кВт·ч).
- Оптовый рынок: ниже 750 кВт выход невозможен; от 750 кВт до 20 МВт через энергосбытовую компанию-посредника; от 20 МВт прямое участие.
- Поиск коммерческих потерь не предлагай обычному промышленному предприятию: он для электросетевых организаций. Результат поиска — список приоритетных проверок, а не доказанный факт хищения.
- Не обещай гарантированную экономию, окупаемость, сроки и точную стоимость без расчёта по объекту. Предварительная оценка эффекта бесплатна и ни к чему не обязывает; если проект невыгоден, клиенту скажут об этом.
- Не называй компании-клиентов, производителей оборудования и подрядчиков. Не раскрывай внутренние алгоритмы и пороги аналитики.
- Не давай юридических заключений и не подтверждай обязательства от лица компании.
- Пиши по-русски, без markdown-разметки и списков."""


STOP = set(
    "и в во не что он на я с со как а то все она так его но да ты к у же вы за бы по только ее мне было вот от меня еще нет о из ему теперь когда даже ну ли если уже или ни быть был него до вас опять уж вам ведь там потом себя ничего ей может они тут где есть надо ней для мы тебя их чем была сам чтоб без будто чего раз тоже себе под будет тогда кто этот того потому этого какой совсем ним здесь этом один почти мой тем чтобы нее сейчас были куда зачем всех можно при об после над больше тот через эти нас про всего них какая много разве эту моя свою этой перед чуть том такой им более всегда конечно всю между нам хотим хотите хочет ваш ваша ваше наш наша наше вообще это очень нужно также либо".split()
)
ENDINGS = sorted(
    ["ями", "ами", "ией", "ого", "его", "ому", "ему", "ыми", "ими", "ов", "ев", "ей", "ах", "ях", "ий", "ый", "ая", "яя", "ое", "ее", "ые", "ие", "ой", "ом", "ем", "ую", "юю", "ть", "ся", "ют", "ут", "ет", "ит", "ам", "ям", "а", "я", "ы", "и", "у", "ю", "е", "о", "ь"],
    key=len,
    reverse=True,
)


def stem(word):
    if word.isdigit() or len(word) <= 3:
        return word
    for end in ENDINGS:
        if word.endswith(end) and len(word) - len(end) >= 3:
            word = word[: -len(end)]
            break
    return word[:6]


def tokenize(text):
    text = text.lower().replace("ё", "е")
    return [stem(w) for w in re.findall(r"[0-9a-zа-я]+", text) if w not in STOP and (len(w) > 1 or w.isdigit())]


def find_file(*names):
    for root in (DATA_DIR, Path(__file__).resolve().parent / DATA_DIR, Path.cwd(), Path("/tmp")):
        for name in names:
            path = root / name
            if path.is_file():
                return path
    raise FileNotFoundError(f"Не найдены файлы: {names}")


def split_section(text, limit):
    paragraphs = [p.strip() for p in re.split(r"\n\s*\n", text) if p.strip()]
    parts, current = [], ""
    for p in paragraphs:
        if len(p) > limit:
            sentences = re.split(r"(?<=[.!?;])\s+", p)
        else:
            sentences = [p]
        for s in sentences:
            if current and len(current) + len(s) + 2 > limit:
                parts.append(current)
                current = ""
            current = f"{current}\n{s}" if current else s
    if current:
        parts.append(current)
    return parts


def load_chunks(path):
    chunks = []
    with zipfile.ZipFile(path) as archive:
        for name in sorted(archive.namelist()):
            if name.startswith("__MACOSX/") or Path(name).suffix.lower() not in (".md", ".txt"):
                continue
            text = archive.read(name).decode("utf-8-sig", errors="replace")
            source = Path(name).name
            stack = {}
            sections = []
            body = []
            for line in text.splitlines():
                m = re.match(r"^(#{1,4})\s+(.*)", line)
                if m:
                    if "".join(body).strip():
                        sections.append((" > ".join(stack[k] for k in sorted(stack)), "\n".join(body)))
                    body = []
                    level = len(m.group(1))
                    stack = {k: v for k, v in stack.items() if k < level}
                    stack[level] = m.group(2).strip()
                else:
                    body.append(line)
            if "".join(body).strip():
                sections.append((" > ".join(stack[k] for k in sorted(stack)), "\n".join(body)))
            for title, content in sections:
                for part in split_section(content, CHUNK_CHARS):
                    head = f"{source} | {title}" if title else source
                    chunks.append({"source": source, "title": title, "text": f"[{head}]\n{part}"})
    if not chunks:
        raise ValueError("База знаний пуста")
    return chunks


def post(client, endpoint, key, body, attempts=3):
    if not BASE_URL or not key:
        raise ValueError("Не заданы адрес API или ключ")
    last = None
    for attempt in range(attempts):
        try:
            response = client.post(BASE_URL + endpoint, headers={"Authorization": "Bearer " + key}, json=body)
            if response.status_code in (429, 500, 502, 503, 504):
                raise httpx.HTTPStatusError("retry", request=response.request, response=response)
            response.raise_for_status()
            return response.json()
        except httpx.HTTPStatusError as exc:
            last = exc
            if exc.response.status_code not in (429, 500, 502, 503, 504):
                raise
        except (httpx.TimeoutException, httpx.TransportError) as exc:
            last = exc
        time.sleep(min(1.5 * (attempt + 1), 5))
    raise last


def embed(client, texts):
    out = []
    for i in range(0, len(texts), 32):
        result = post(client, "/embeddings", EMB_KEY, {"model": EMB_MODEL, "input": texts[i : i + 32]})
        items = sorted(result["data"], key=lambda d: d.get("index", 0))
        out.extend(item["embedding"] for item in items)
    matrix = np.array(out, dtype=np.float32)
    matrix /= np.linalg.norm(matrix, axis=1, keepdims=True) + 1e-9
    return matrix


class Index:
    def __init__(self, chunks, client):
        self.chunks = chunks
        self.bm25 = BM25Okapi([tokenize(c["title"] + " " + c["text"]) for c in chunks])
        self.vectors = None
        try:
            self.vectors = embed(client, [c["text"] for c in chunks])
        except Exception as exc:
            print(f"Эмбеддинги недоступны: {type(exc).__name__}: {exc}", file=sys.stderr)

    def search(self, client, query, dense_query):
        scores = self.bm25.get_scores(tokenize(query))
        ranked = [int(i) for i in np.argsort(scores)[::-1][:CANDIDATES] if scores[i] > 0]
        dense = []
        if self.vectors is not None:
            try:
                q = embed(client, [dense_query])[0]
                dense = [int(i) for i in np.argsort(self.vectors @ q)[::-1][:CANDIDATES]]
            except Exception as exc:
                print(f"Эмбеддинг запроса: {type(exc).__name__}", file=sys.stderr)
        rrf = {}
        for lst, weight in ((ranked, 1.0), (dense, 1.2)):
            for rank, i in enumerate(lst):
                rrf[i] = rrf.get(i, 0.0) + weight / (60 + rank)
        pool = [i for i, _ in sorted(rrf.items(), key=lambda x: x[1], reverse=True)][:RERANK_POOL]
        if not pool:
            return []
        try:
            result = post(
                client,
                "/rerank",
                RERANK_KEY,
                {
                    "model": RERANKER_MODEL,
                    "query": dense_query,
                    "documents": [self.chunks[i]["text"] for i in pool],
                    "top_n": min(FINAL_K, len(pool)),
                },
                attempts=2,
            )
            entries = result.get("results", result.get("data", []))
            order = [pool[int(e["index"])] for e in entries if 0 <= int(e["index"]) < len(pool)]
            if order:
                return [self.chunks[i] for i in order[:FINAL_K]]
        except Exception as exc:
            print(f"Реранкер недоступен: {type(exc).__name__}", file=sys.stderr)
        return [self.chunks[i] for i in pool[:FINAL_K]]


TURN = re.compile(r"^\s*(Пользователь|Клиент|Ассистент|Бот|User|Assistant)\s*:\s*", re.IGNORECASE | re.MULTILINE)


def parse_dialog(question):
    parts = TURN.split(question)
    if len(parts) < 3:
        return [("user", question.strip())]
    turns = []
    for role, text in zip(parts[1::2], parts[2::2]):
        turns.append(("user" if role.lower() in ("пользователь", "клиент", "user") else "assistant", text.strip()))
    return [t for t in turns if t[1]] or [("user", question.strip())]


def build_queries(turns):
    users = [t for r, t in turns if r == "user"]
    last = users[-1]
    earlier = " ".join(users[:-1])
    bm25_query = f"{last} {earlier} {earlier}".strip()
    dense_query = f"{earlier} {last}".strip() if earlier else last
    return bm25_query, dense_query


def clean(answer):
    answer = re.sub(r"<think>.*?</think>", "", answer, flags=re.DOTALL)
    answer = re.sub(r"^.*</think>", "", answer, flags=re.DOTALL)
    answer = re.sub(r"[*#`]+", "", answer)
    return " ".join(answer.split())


def generate(client, turns, chunks):
    context = "\n\n".join(c["text"] for c in chunks) or "Фрагменты не найдены."
    if len(turns) == 1:
        dialog = turns[0][1]
        task = f"Вопрос клиента:\n{dialog}"
    else:
        dialog = "\n".join(("Клиент: " if r == "user" else "Консультант: ") + t for r, t in turns)
        task = f"Диалог:\n{dialog}\n\nОтветь на последнюю реплику клиента с учётом диалога."
    payload = {
        "model": LLM_MODEL,
        "temperature": TEMPERATURE,
        "max_tokens": MAX_TOKENS,
        "messages": [
            {"role": "system", "content": SYSTEM_PROMPT + "\n\nФРАГМЕНТЫ БАЗЫ ЗНАНИЙ:\n" + context},
            {"role": "user", "content": task},
        ],
        "chat_template_kwargs": {"enable_thinking": False},
    }
    for attempt in range(2):
        try:
            result = post(client, "/chat/completions", LLM_KEY, payload)
            answer = clean(result["choices"][0]["message"].get("content") or "")
            if answer:
                return answer[:9000]
        except httpx.HTTPStatusError as exc:
            if exc.response.status_code == 400 and "chat_template_kwargs" in payload:
                payload.pop("chat_template_kwargs")
                continue
            print(f"LLM: HTTP {exc.response.status_code}", file=sys.stderr)
        except Exception as exc:
            print(f"LLM попытка {attempt + 1}: {type(exc).__name__}", file=sys.stderr)
    return FALLBACK


def main():
    questions_path = find_file("questions.csv", "train.csv")
    kb_path = find_file("knowledge-base.zip", "knowledge_base.zip", "kb.zip")
    with questions_path.open(encoding="utf-8-sig", newline="") as f:
        questions = [(str(r["id"]).strip(), str(r["question"])) for r in csv.DictReader(f)]
    chunks = load_chunks(kb_path)
    with httpx.Client(timeout=90) as client:
        index = Index(chunks, client)
    print(f"Вопросов: {len(questions)}; чанков: {len(chunks)}; dense: {index.vectors is not None}", flush=True)

    def work(question):
        if time.time() > DEADLINE:
            return FALLBACK
        try:
            turns = parse_dialog(question)
            bm25_query, dense_query = build_queries(turns)
            with httpx.Client(timeout=90) as client:
                found = index.search(client, bm25_query, dense_query)
                return generate(client, turns, found)
        except Exception as exc:
            print(f"Ошибка вопроса: {type(exc).__name__}: {exc}", file=sys.stderr)
            return FALLBACK

    answers = [FALLBACK] * len(questions)
    with ThreadPoolExecutor(max_workers=MAX_WORKERS) as pool:
        futures = {pool.submit(work, q): i for i, (_, q) in enumerate(questions)}
        for future in as_completed(futures):
            answers[futures[future]] = future.result()

    with open("submission.csv", "w", encoding="utf-8", newline="") as f:
        writer = csv.writer(f, quoting=csv.QUOTE_ALL, lineterminator="\n")
        writer.writerow(["id", "answer"])
        writer.writerows((qid, answers[i] or FALLBACK) for i, (qid, _) in enumerate(questions))
    print(f"Готов submission.csv: {len(questions)} ответов", flush=True)


if __name__ == "__main__":
    main()