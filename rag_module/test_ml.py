import hashlib
import json
import os
import re
import sys
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

HERE = Path(__file__).resolve().parent
PORT = int(os.getenv("TEST_PORT", "9990"))
KB = os.getenv("KB_PATH")
if not KB or not Path(KB).exists():
    sys.exit("Укажите KB_PATH: путь к knowledge-base.zip или папке с документами")


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def reply(self, code, payload):
        data = json.dumps(payload).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(data)

    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        mode = self.server.mode
        if self.path.endswith("/embeddings"):
            if mode == "emb500":
                return self.reply(500, {"error": "x"})

            def vector(text):
                v = [0.0] * 128
                for word in re.findall(r"\w+", text.lower()):
                    v[int(hashlib.md5(word[:5].encode()).hexdigest(), 16) % 128] += 1
                return v

            return self.reply(200, {"data": [{"index": i, "embedding": vector(t)} for i, t in enumerate(body["input"])]})
        if self.path.endswith("/rerank"):
            if mode == "norerank":
                return self.reply(404, {"error": "not found"})
            return self.reply(200, {"results": [{"index": i, "relevance_score": 1} for i in range(body["top_n"])]})
        self.server.last = body
        if mode == "chat500":
            return self.reply(500, {"error": "boom"})
        if mode == "slow":
            time.sleep(12)
        user = body["messages"][-1]["content"]
        return self.reply(200, {"choices": [{"message": {"content": "ECHO[" + user.replace("\n", " | ")[-400:] + "]"}, "finish_reason": "stop"}]})


server = ThreadingHTTPServer(("127.0.0.1", PORT), Handler)
server.mode, server.last = "ok", None
threading.Thread(target=server.serve_forever, daemon=True).start()
os.environ.update(LLM_BASE_URL=f"http://127.0.0.1:{PORT}/v1", LLM_API_KEY="test", KB_PATH=KB)
sys.path.insert(0, str(HERE))
import rag_api  # noqa: E402
import solution as S  # noqa: E402

failed = []


def check(condition, name):
    print(("PASS " if condition else "FAIL ") + name)
    if not condition:
        failed.append(name)


rag_api.init()
history = [{"role": "user", "content": "У нас магазин."}, {"role": "assistant", "content": "Какая у вас присоединённая мощность?"}]
result = rag_api.generate_answer("Около 900 кВт, можно на оптовый рынок?", history=history)
system = server.last["messages"][0]["content"]
check(result["degraded"] == [] and result["answer"] and result["sources"], "обычный ответ: текст, источники, без деградации")
check("Никогда не утверждай, что заявка" in system, "режим сайта: нет ложного подтверждения заявки")
check(set(result["timings_ms"]) == {"retrieval", "generation", "total"}, "время в ответе")

long_history = [{"role": "user", "content": "Параметры объекта: регион — Москва, мощность 900 кВт"}]
for i in range(25):
    long_history += [{"role": "user", "content": f"вопрос номер {i}"}, {"role": "assistant", "content": f"ответ номер {i}"}]
rag_api.generate_answer("итоговый вопрос про тарифы", history=long_history)
last = server.last["messages"][-1]["content"]
check("Параметры объекта" in last and "итоговый вопрос" in last and "вопрос номер 24" in last and "вопрос номер 3\n" not in last, "история обрезается, параметры объекта сохраняются")

answers = {}


def ask(i):
    answers[i] = rag_api.generate_answer(f"уникальныйтокен{i} про солнечные панели", history=[{"role": "user", "content": f"прошлаяреплика{i}"}, {"role": "assistant", "content": "уточните"}])["answer"]


threads = [threading.Thread(target=ask, args=(i,)) for i in range(4)]
[t.start() for t in threads]
[t.join() for t in threads]
check(len(answers) == 4 and not any(f"токен{j}" in answers[i] for i in answers for j in answers if i != j), "параллельные чаты не смешиваются")

server.mode = "norerank"
check("reranker_failed" in rag_api.generate_answer("сколько стоит станция")["degraded"], "отказ реранкера отражается в degraded")

server.mode = "chat500"
started = time.time()
try:
    rag_api.generate_answer("вопрос")
    check(False, "сбой генерации даёт исключение")
except Exception as exc:
    check("generation_failed" in str(exc) and time.time() - started < 20, "сбой генерации даёт исключение быстро")

server.mode, rag_api.BUDGET_SECONDS = "slow", 4
started = time.time()
try:
    rag_api.generate_answer("вопрос")
    check(False, "бюджет времени соблюдается")
except Exception:
    check(time.time() - started < 9, "бюджет времени соблюдается")
server.mode, rag_api.BUDGET_SECONDS = "ok", 45

rag_api._state.clear()
S.MAX_ATTEMPTS, S.TIMEOUT, server.mode = 1, 90, "emb500"
rag_api.init()
check("embeddings_unavailable" in rag_api.health()["degraded"] and rag_api.generate_answer("сколько стоит станция")["answer"], "без эмбеддингов работает BM25 с пометкой деградации")

print("\nИТОГ:", "все проверки пройдены" if not failed else f"провалено: {failed}")
sys.exit(1 if failed else 0)