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

CHUNK_SIZE = 800
CHUNK_OVERLAP = 150

HEADERS_TO_SPLIT_ON = [
    ("#", "Header1"),
    ("##", "Header2"),
    ("###", "Header3"),
]


def extract_knowledge_base() -> str:
    """
    Locate and prepare the knowledge base folder.
    """
    data_dir = os.getenv("DATA_DIR", "data")
    kb_zip_path = os.path.join(data_dir, "knowledge-base.zip")
    extract_path = "/tmp/kb_extracted"

    if os.path.exists(kb_zip_path):
        print(f"[ingest] Extracting knowledge base from: {kb_zip_path}")
        if os.path.exists(extract_path):
            shutil.rmtree(extract_path)
        os.makedirs(extract_path, exist_ok=True)

        with zipfile.ZipFile(kb_zip_path, "r") as zip_ref:
            zip_ref.extractall(extract_path)

        print(f"[ingest] Extracted to: {extract_path}")
        return extract_path

    print(f"[ingest] No ZIP found at {kb_zip_path}. Falling back to local folder.")
    return LOCAL_KB_PATH


def split_documents(raw_folder: str) -> list:
    """
    Split all Markdown and text files into smaller chunks.
    """
    markdown_splitter = MarkdownHeaderTextSplitter(
        headers_to_split_on=HEADERS_TO_SPLIT_ON
    )
    text_splitter = RecursiveCharacterTextSplitter(
        chunk_size=CHUNK_SIZE,
        chunk_overlap=CHUNK_OVERLAP,
    )

    all_chunks = []

    files = sorted(
        glob.glob(f"{raw_folder}/**/*.md", recursive=True)
        + glob.glob(f"{raw_folder}/**/*.txt", recursive=True)
    )

    if not files:
        print(f"[ingest] WARNING: no .md or .txt files found in {raw_folder}")
        return []

    for file_path in files:
        filename = os.path.basename(file_path)
        print(f"[ingest] Processing: {filename}")

        with open(file_path, "r", encoding="utf-8") as f:
            text = f.read()

        md_chunks = markdown_splitter.split_text(text)

        final_chunks = text_splitter.split_documents(md_chunks)

        for chunk in final_chunks:
            chunk.metadata["source"] = filename
            if "Чего утверждать нельзя" in chunk.page_content or "Не подходит" in chunk.page_content:
                chunk.metadata["type"] = "restriction"
            else:
                chunk.metadata["type"] = "info"

            all_chunks.append(chunk)

    return all_chunks


def save_chunks(chunks: list, output_path: str) -> None:
    """
    Save chunks to a JSONL file.
    """
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

    print(f"[ingest] Saved {len(chunks)} chunks to: {output_path}")


def main():
    raw_folder = extract_knowledge_base()
    chunks = split_documents(raw_folder)

    if not chunks:
        print("[ingest] ERROR: no chunks were produced. Check your input files.")
        return

    save_chunks(chunks, OUTPUT_PATH)

    # Quick sanity check.
    print("\n[ingest] --- First chunk preview ---")
    print(chunks[0].page_content[:300])
    print("[ingest] Metadata:", chunks[0].metadata)


if __name__ == "__main__":
    main()