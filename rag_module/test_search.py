
import os
import json
import numpy as np

from rag_module import retriever as retriever_module

def mock_get_embeddings(texts):
    """Returns random embeddings for local testing."""
    return np.random.rand(len(texts), 1024).astype(np.float32)

retriever_module.get_embeddings = mock_get_embeddings

from rag_module.retriever import Retriever


def main():
    print("=" * 60)
    print("LOCAL RETRIEVER TEST (Mock Embeddings)")
    print("=" * 60)

    # 1. Initialize retriever
    print("\n[1] Initializing Retriever...")
    r = Retriever(chunks_path="rag_module/data/chunks.jsonl", use_faiss=False)
    print(f"    ✓ Loaded {len(r.chunks)} chunks")

    # 2. Test queries
    test_queries = [
        "У нас старая шиферная крыша. Стоит вообще рассматривать солнечные панели?",
        "Можно сначала поставить солнечную станцию на 60 кВт, а потом расширить?",
        "В счётчиках стоят симки другого оператора. Для вашей платформы их надо менять?",
        "Нам предлагают компенсировать реактивную энергию. Сначала надо что-то измерить?",
    ]

    print("\n[2] Testing retrieval on demo questions...")
    for i, query in enumerate(test_queries, 1):
        print("\n" + "-" * 60)
        print(f"Query {i}: {query}")
        print("-" * 60)

        results = r.retrieve_documents(query, top_k=3)

        for j, res in enumerate(results, 1):
            preview = res['text'][:150].replace('\n', ' ')
            print(f"\n  Result {j}:")
            print(f"    Source: {res['source']}")
            print(f"    Score:  {res['score']:.4f}")
            print(f"    Text:   {preview}...")

    print("\n" + "=" * 60)
    print("TEST COMPLETE")
    print("=" * 60)


if __name__ == "__main__":
    main()