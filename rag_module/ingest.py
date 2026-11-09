"""
Knowledge Base Ingestion Script.
Splits documents into smaller, denser chunks for precise retrieval.
"""

import os
import glob
import json
import zipfile
import shutil
from langchain_text_splitters import (
    MarkdownHeaderTextSplitter,
    RecursiveCharacterTextSplitter,
)

LOCAL_KB_PATH = "rag_module/data/raw"
OUTPUT_PATH = "rag_module/data/chunks.jsonl"

# Smaller chunks = less noise for BM25
CHUNK_SIZE = 450
CHUNK_OVERLAP = 60

HEADERS_TO_SPLIT_ON = [
    ("#", "Header1"),
    ("##", "Header2"),
    ("###", "Header3"),
]


def extract_knowledge_base() -> str:
    data_dir = os.getenv("DATA_DIR", "data")
    kb_zip_path = os.path.join(data_dir, "knowledge-base.zip")
    extract_path = "/tmp/kb_extracted"

    if os.path.exists(kb_zip_path):
        print(f"[ingest] Extracting: {kb_zip_path}")
        if os.path.exists(extract_path):
            shutil.rmtree(extract_path)
        os.makedirs(extract_path, exist_ok=True)
        with zipfile.ZipFile(kb_zip_path, "r") as zip_ref:
            zip_ref.extractall(extract_path)
        return extract_path

    print(f"[ingest] No ZIP, using local folder: {LOCAL_KB_PATH}")
    return LOCAL_KB_PATH


def split_documents(raw_folder: str) -> list:
    md_splitter = MarkdownHeaderTextSplitter(headers_to_split_on=HEADERS_TO_SPLIT_ON)
    txt_splitter = RecursiveCharacterTextSplitter(
        chunk_size=CHUNK_SIZE, chunk_overlap=CHUNK_OVERLAP
    )

    all_chunks = []
    files = sorted(
        glob.glob(f"{raw_folder}/**/*.md", recursive=True)
        + glob.glob(f"{raw_folder}/**/*.txt", recursive=True)
    )

    for file_path in files:
        filename = os.path.basename(file_path)
        with open(file_path, "r", encoding="utf-8") as f:
            text = f.read()

        for chunk in txt_splitter.split_documents(md_splitter.split_text(text)):
            chunk.metadata["source"] = filename
            if "Чего утверждать нельзя" in chunk.page_content or "Не подходит" in chunk.page_content:
                chunk.metadata["type"] = "restriction"
            else:
                chunk.metadata["type"] = "info"
            all_chunks.append(chunk)

    return all_chunks


def save_chunks(chunks: list, output_path: str) -> None:
    os.makedirs(os.path.dirname(output_path), exist_ok=True)
    with open(output_path, "w", encoding="utf-8") as f:
        for chunk in chunks:
            f.write(
                json.dumps(
                    {"text": chunk.page_content, "metadata": chunk.metadata},
                    ensure_ascii=False,
                )
                + "\n"
            )
    print(f"[ingest] Saved {len(chunks)} chunks → {output_path}")


def main():
    raw_folder = extract_knowledge_base()
    chunks = split_documents(raw_folder)
    if not chunks:
        print("[ingest] ERROR: no chunks produced.")
        return
    save_chunks(chunks, OUTPUT_PATH)


if __name__ == "__main__":
    main()