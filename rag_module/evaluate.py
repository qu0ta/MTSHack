import os
import csv
import sys
import numpy as np
from rag_module import retriever as retriever_module

def mock_get_embeddings(texts):
    return np.random.rand(len(texts), 1024).astype(np.float32)

GROUND_TRUTH = {
    1:  ["23-solar.md"],
    2:  ["23-solar.md"],
    3:  ["23-solar.md"],
    4:  ["23-solar.md"],
    5:  ["22-askue.md"],
    6:  ["22-askue.md"],
    7:  ["22-askue.md"],
    8:  ["25-meter-reading.md"],
    9:  ["25-meter-reading.md"],
    10: ["14-tariffs.md", "11-price-categories.md"],
    11: ["17-faq.md"],
    12: ["21-orem.md"],
    13: [
        "03-calculation-and-requests.md",
        "21-orem.md",
        "26-general.md",
        "20-portfolio.md",
        "14-tariffs.md",
    ],
    14: ["23-solar.md"],
    15: ["22-askue.md"],
    16: ["24-losses.md"],
    17: ["11-price-categories.md", "14-tariffs.md"],
    18: [
        "03-calculation-and-requests.md",
        "22-askue.md",
        "26-general.md",
    ],
}


def load_questions(path: str) -> list:
    """Load demo_questions.csv (handles BOM)."""
    rows = []
    with open(path, "r", encoding="utf-8-sig") as f:
        reader = csv.DictReader(f)
        for row in reader:
            rows.append({"id": int(row["id"]), "question": row["question"]})
    return rows


def find_questions_file() -> str:
    """Search for demo_questions.csv in common locations."""
    candidates = [
        "data/demo_questions.csv",
        "rag_module/data/demo_questions.csv",
        "demo_questions.csv",
        "../data/demo_questions.csv",
    ]
    for p in candidates:
        if os.path.exists(p):
            return p
    raise FileNotFoundError(
        "demo_questions.csv not found. Put it in data/ or rag_module/data/"
    )


def main():
    
    retriever_module.get_embeddings = mock_get_embeddings

    print("Loading Retriever (BM25 only)...")
    r = retriever_module.Retriever(
        chunks_path="rag_module/data/chunks.jsonl",
        use_faiss=False,
    )

    qpath = find_questions_file()
    questions = load_questions(qpath)
    print(f"Loaded {len(questions)} questions from {qpath}\n")

    hits_at_1 = 0
    hits_at_3 = 0
    hits_at_5 = 0
    misses = []

    for q in questions:
        qid = q["id"]
        question = q["question"]
        expected = set(GROUND_TRUTH.get(qid, []))

        results = r.retrieve_documents(question, top_k=5)
        sources = [res["source"] for res in results]

        in_top1 = any(s in expected for s in sources[:1])
        in_top3 = any(s in expected for s in sources[:3])
        in_top5 = any(s in expected for s in sources[:5])

        hits_at_1 += int(in_top1)
        hits_at_3 += int(in_top3)
        hits_at_5 += int(in_top5)

        status = "✓" if in_top3 else "✗"
        print(f"[{status}] Q{qid:02d}  top1={sources[0] if sources else '-'}")

        if not in_top3:
            misses.append((qid, expected, sources[:3]))

    n = len(questions)
    print("\n" + "=" * 60)
    print("RESULTS")
    print("=" * 60)
    print(f"Accuracy@1: {hits_at_1}/{n} = {hits_at_1 / n:.1%}")
    print(f"Accuracy@3: {hits_at_3}/{n} = {hits_at_3 / n:.1%}")
    print(f"Accuracy@5: {hits_at_5}/{n} = {hits_at_5 / n:.1%}")

    if misses:
        print("\nMISSES (correct source not in top-3):")
        for qid, expected, got in misses:
            print(f"  Q{qid:02d}  expected={sorted(expected)}  got={got}")


if __name__ == "__main__":
    main()