FROM python:3.11-slim
WORKDIR /service

ARG KNOWLEDGE_BASE_URL="https://drive.google.com/uc?export=download&id=18jMUdDN10xJHdmGfjjrKY9HPjd7LsGWG"
ARG KNOWLEDGE_BASE_SHA256="d5f1bbba0094d1eb7e4918bd7a1d37c621ae5cd498010ca2cba041ddddafd66a"

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

RUN python -c "import hashlib, pathlib, urllib.request; p = pathlib.Path('/opt/mtshack/knowledge-base.zip'); p.parent.mkdir(parents=True, exist_ok=True); urllib.request.urlretrieve('${KNOWLEDGE_BASE_URL}', p); actual = hashlib.sha256(p.read_bytes()).hexdigest(); expected = '${KNOWLEDGE_BASE_SHA256}'.lower(); assert actual == expected, f'knowledge-base.zip checksum mismatch: expected {expected}, got {actual}'"

COPY app ./app

ENV KB_PATH=/opt/mtshack/knowledge-base.zip

EXPOSE 8000
CMD ["python", "-m", "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
