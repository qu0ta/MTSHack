"""Async boundary around the synchronous, externally owned ML pipeline."""
import asyncio
from collections.abc import Callable
from typing import Any

from pydantic import BaseModel, ConfigDict, Field, ValidationError, field_validator


class MLUnavailable(Exception):
    """Initialization or generation failed."""


class MLInvalidResponse(Exception):
    """The ML module returned a broken contract."""


class MLBusy(Exception):
    """All workers are still processing previous requests."""


class MLTimeout(Exception):
    """The HTTP waiting budget expired; the synchronous worker may continue."""


class MLAnswer(BaseModel):
    model_config = ConfigDict(strict=True)

    answer: str = Field(min_length=1, max_length=10000)
    sources: list[str]
    degraded: list[str] = Field(default_factory=list)
    timings_ms: dict[str, float] = Field(default_factory=dict)

    @field_validator("timings_ms")
    @classmethod
    def valid_timings(cls, values: dict[str, float]) -> dict[str, float]:
        import math
        if any(not math.isfinite(value) or value < 0 for value in values.values()):
            raise ValueError("Invalid ML timings")
        return values

    @field_validator("answer")
    @classmethod
    def nonempty_answer(cls, value: str) -> str:
        if not value.strip():
            raise ValueError("Empty ML answer")
        return value.strip()

    @field_validator("sources")
    @classmethod
    def unique_sources(cls, values: list[str]) -> list[str]:
        return list(dict.fromkeys(value.strip() for value in values if value.strip()))


class RAGAdapter:
    def __init__(
        self,
        generate_function: Callable[..., dict[str, Any]],
        init_function: Callable[[], None],
        *,
        timeout_seconds: float = 55.0,
        max_concurrency: int = 2,
        health_function: Callable[[], dict[str, Any]] | None = None,
    ) -> None:
        if timeout_seconds <= 0 or max_concurrency < 1:
            raise ValueError("Timeout and concurrency must be positive")
        self._generate = generate_function
        self._init = init_function
        self._timeout = timeout_seconds
        self._limit = max_concurrency
        self._ready = False
        self._init_lock = asyncio.Lock()
        self._tasks: set[asyncio.Task[MLAnswer]] = set()
        self._health = health_function
        self._startup_degraded: list[str] = []

    async def initialize(self) -> None:
        async with self._init_lock:
            if self._ready:
                return
            try:
                await asyncio.to_thread(self._init)
                if self._health is not None:
                    health = await asyncio.to_thread(self._health)
                    if health.get("ready") is not True:
                        raise ValueError("ML health is not ready")
                    flags = health.get("degraded", [])
                    if not isinstance(flags, list) or any(not isinstance(flag, str) for flag in flags):
                        raise ValueError("Invalid health degradation flags")
                    self._startup_degraded = list(flags)
            except Exception as exc:
                raise MLUnavailable("ML initialization failed") from exc
            self._ready = True

    def health(self) -> dict[str, Any]:
        return {"ready": self._ready, "degraded": list(self._startup_degraded),
                "active_requests": len(self._tasks), "capacity": self._limit}

    async def _run(self, query: str, history: list[dict[str, str]]) -> MLAnswer:
        try:
            result = await asyncio.to_thread(
                self._generate, query=query, history=history,
            )
        except Exception as exc:
            raise MLUnavailable("ML generation failed") from exc
        try:
            return MLAnswer.model_validate(result)
        except ValidationError as exc:
            raise MLInvalidResponse("Invalid ML response") from exc

    def _finished(self, task: asyncio.Task[MLAnswer]) -> None:
        self._tasks.discard(task)
        # Consume late exceptions when the HTTP caller has already timed out.
        if not task.cancelled():
            task.exception()

    async def generate(self, query: str, history: list[dict[str, str]]) -> MLAnswer:
        if not self._ready:
            raise MLUnavailable("ML service is not initialized")
        if len(self._tasks) >= self._limit:
            raise MLBusy("ML workers are busy")
        task = asyncio.create_task(self._run(query, history))
        self._tasks.add(task)
        task.add_done_callback(self._finished)
        try:
            # Timeout/caller cancellation must not free a still-running thread's slot.
            return await asyncio.wait_for(asyncio.shield(task), timeout=self._timeout)
        except TimeoutError as exc:
            raise MLTimeout("ML response timed out") from exc
