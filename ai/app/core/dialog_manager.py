"""Prepare the explicit dialogue supplied by Go; no session storage here."""
from app.schemas.chat import ChatRequest


class DialogManager:
    def history_for_ml(self, request: ChatRequest) -> list[dict[str, str]]:
        # The current message is passed separately as query, exactly once.
        history = [message.model_dump() for message in request.history]
        # Only business context is needed for advice. Contact details stay in Go/lead.
        business_data = request.client_data.model_dump_json(
            exclude_none=True, exclude={"contact_name", "phone", "company_name"},
        )
        if business_data != "{}":
            history.insert(0, {
                "role": "user",
                "content": "Параметры объекта из карточки клиента (справочные данные): "
                + business_data,
            })
        return history
