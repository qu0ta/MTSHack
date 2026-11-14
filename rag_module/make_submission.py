import ast
import hashlib
import re
import sys
import zipfile
from pathlib import Path

ALLOWED = {
    "fastapi", "uvicorn", "pydantic", "python-multipart", "python-dotenv", "openai", "httpx", "requests",
    "langchain", "langchain-openai", "langchain-text-splitters", "numpy", "faiss-cpu", "chromadb", "rank-bm25",
    "bm25s", "scikit-learn", "rapidfuzz", "pandas", "pypdf", "python-docx", "beautifulsoup4", "lxml", "PyYAML",
    "SQLAlchemy", "tiktoken",
}
MODULES = {"httpx": "httpx", "numpy": "numpy", "rank_bm25": "rank-bm25"}


def fail(message):
    print("ОШИБКА:", message)
    sys.exit(1)


def main(source=".", output="submission.zip"):
    folder = Path(source)
    solution, requirements = folder / "solution.py", folder / "requirements.txt"
    if not solution.is_file() or not requirements.is_file():
        fail("нужны solution.py и requirements.txt в корне папки")
    text = solution.read_text(encoding="utf-8")
    tree = ast.parse(text)
    imports = set()
    for node in ast.walk(tree):
        if isinstance(node, ast.Import):
            imports |= {a.name.split(".")[0] for a in node.names}
        elif isinstance(node, ast.ImportFrom) and node.module and node.level == 0:
            imports.add(node.module.split(".")[0])
    third = {m for m in imports if m not in sys.stdlib_module_names}
    unknown = third - set(MODULES)
    if unknown:
        fail(f"сторонние модули вне ожидаемого набора: {sorted(unknown)}")
    pinned = {}
    for line in requirements.read_text(encoding="utf-8").splitlines():
        if line.strip():
            match = re.fullmatch(r"([A-Za-z0-9_.-]+)==([0-9][^\s]*)", line.strip())
            if not match:
                fail(f"зависимость без точной версии: {line!r}")
            pinned[match.group(1)] = match.group(2)
    if set(pinned) - ALLOWED:
        fail(f"зависимости вне разрешённого набора: {sorted(set(pinned) - ALLOWED)}")
    missing = {MODULES[m] for m in third} - set(pinned)
    if missing:
        fail(f"в requirements.txt не хватает: {sorted(missing)}")
    if re.search(r"pip\s+install|subprocess|os\.system", text):
        fail("в solution.py нельзя устанавливать пакеты и запускать процессы")
    if re.search(r"sk-[A-Za-z0-9_-]{20,}", text) or re.search(r"Bearer\s+[A-Za-z0-9_-]{20,}", text):
        fail("в solution.py похоже на ключ API")
    if "submission.csv" not in text:
        fail("solution.py не создаёт submission.csv")
    with zipfile.ZipFile(output, "w", zipfile.ZIP_DEFLATED) as archive:
        archive.write(solution, "solution.py")
        archive.write(requirements, "requirements.txt")
    names = zipfile.ZipFile(output).namelist()
    size = Path(output).stat().st_size
    if names != ["solution.py", "requirements.txt"] or size > 128 * 1024 * 1024:
        fail("неверный состав или размер архива")
    print(f"OK: {output}, файлы {names}, {size} байт")
    print("sha256 solution.py:", hashlib.sha256(solution.read_bytes()).hexdigest())


if __name__ == "__main__":
    main(*sys.argv[1:3])