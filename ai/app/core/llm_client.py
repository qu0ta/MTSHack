import os 
import httpx

class LLMClient:
    def __init__(self):
        self.base_url = os.getenv("LLM_BASE_URL", "").rstrip("/")
        self.api_key = os.getenv("LLM_API_KEY", "")
        self.model = os.getenv("LLM_MODEL", "")
        if not self.base_url or not self.api_key or not self.model:
            raise ValueError(
                "Для подключения LLM задайте переменные "
                "LLM_BASE_URL, LLM_API_KEY и LLM_MODEL."
            )
        
    async def generate(self, messages: list[dict[str, str]]) -> str:
        try:
            async with httpx.AsyncClient(timeout=60.0) as client:
                response = await client.post(
                    f"{self.base_url}/chat/completions",
                    headers={
                        "Authorization": f"Bearer {self.api_key}",
                    },
                    json={
                        "model": self.model,
                        "messages": messages,
                        "temperature": 0.0,
                    },
                )

                response.raise_for_status()

                data = response.json()
                answer = data["choices"][0]["message"]["content"]

                if not isinstance(answer, str) or not answer.strip():
                    raise LLMError(
                        "LLM вернула пустой или некорректный ответ."
                    )

                return answer.strip()

        except httpx.HTTPError as exc:
            raise LLMError("Ошибка обращения к LLM.") from exc

        except (ValueError, KeyError, IndexError, TypeError) as exc:
            raise LLMError("Некорректный формат ответа LLM.") from exc

class LLMError(Exception):
    pass

