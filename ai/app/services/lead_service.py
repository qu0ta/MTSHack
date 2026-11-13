"""Prepare a lead from backend data; do not infer contacts or persist anything."""
import re

from app.schemas.chat import ChatRequest, ClientData, Lead


class LeadService:
    CONTACTS = {
        "contact_name": "имя контактного лица",
        "phone": "телефон",
        "company_name": "название компании",
    }

    def missing_fields(self, data: ClientData) -> list[str]:
        return [label for field, label in self.CONTACTS.items() if not (getattr(data, field) or "").strip()]

    def summarize(self, request: ChatRequest) -> str:
        # Extract the stated task; never infer it from the assistant's recommendations.
        candidates = []
        purpose = re.compile(r"хоч|нуж|интерес|планир|задач|цель|сниз|налад|уч[её]т|потер|солнеч|оптов|расход", re.I)
        contact_only = re.compile(
            r"^(?:да|нет|спасибо|хорошо|ок|соглас\w*|хочу заяв\w*|(?:прошу |хочу )?(?:оформ\w*|остав\w*|отправ\w*) заяв\w*|"
            r"(?:прошу )?свяж\w* со мной|(?:хочу |нужен )?(?:менеджер\w*|специалист\w*))[.! ]*$", re.I,
        )
        utterances = [message.content for message in request.history if message.role == "user"] + [request.message]
        for text in utterances:
            for sentence in re.split(r"(?<=[.!?])\s+|\n+", text):
                sentence = " ".join(sentence.split())
                if sentence and purpose.search(sentence) and not contact_only.fullmatch(sentence):
                    candidates.append(sentence)
        if candidates:
            task = candidates[-1]
            if len(task) > 400:
                task = task[:397].rsplit(" ", 1)[0] + "…"
            heading = f"Запрос клиента: «{task}»"
        else:
            heading = "Задача клиента не уточнена в переписке; выяснить при звонке."
        details = []
        for field, label in (
            ("industry", "отрасль"), ("region", "регион"),
            ("connected_power_kw", "мощность, кВт"),
            ("monthly_consumption_kwh", "потребление, кВт·ч/мес"),
        ):
            value = getattr(request.client_data, field)
            if value is not None:
                details.append(f"{label}: {str(value)[:120]}")
        parts = [heading]
        if details:
            parts.append("Данные из карточки: " + "; ".join(details) + ".")
        parts.append("Следующий шаг: связаться с клиентом по указанному телефону и уточнить решение задачи.")
        return "\n".join(parts)

    def prepare(self, request: ChatRequest) -> Lead | None:
        if not request.lead_requested or self.missing_fields(request.client_data):
            return None
        data = request.client_data
        return Lead(
            contact_name=data.contact_name.strip(),
            phone=data.phone.strip(),
            company_name=data.company_name.strip(),
            summary=self.summarize(request),
        )
