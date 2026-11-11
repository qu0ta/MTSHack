from app.schemas.chat import ChatRequest


class DialogManager:
    def build_search_query(self, request: ChatRequest) -> str:
        previous_question = next(
            (
                message.content
                for message in reversed(request.history)
                if message.role == "user"
            ),
            None,
        )

        if previous_question is None:
            return request.message

        return (
            f"Предыдущая реплика клиента: {previous_question}\n"
            f"Текущий вопрос: {request.message}"
        )