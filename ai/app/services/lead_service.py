from app.schemas.chat import ChatRequest, Lead


def prepare_lead(request: ChatRequest) -> tuple[str, Lead | None]:
    data = request.client_data
    name = (data.contact_name or "").strip()
    phone = (data.phone or "").strip()
    company = (data.company_name or "").strip()
    missing = [label for label, value in (("ФИО", name), ("телефон", phone), ("компанию", company)) if not value]
    if missing:
        return "Для заявки укажите " + ", ".join(missing) + ".", None

    questions = [item.content for item in request.history or [] if item.role == "user"]
    summary = "\n".join([*questions, request.message])
    return "Данные для заявки подготовлены.", Lead(
        contact_name=name, phone=phone, company_name=company, summary=summary
    )
