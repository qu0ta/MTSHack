from typing import Annotated, Literal

from pydantic import BaseModel, Field, StrictBool, StringConstraints

MessageText = Annotated[str, StringConstraints(strip_whitespace=True, min_length=1, max_length=10000)]
Quantity = Annotated[float, Field(ge=0, allow_inf_nan=False)]

class ChatMessage(BaseModel):
    role: Literal["user", "assistant"]
    content: MessageText

class ClientData(BaseModel):
    company_name: str | None = None
    contact_name: str | None = None
    phone: str | None = None
    industry: str | None = None
    region: str | None = None
    connected_power_kw: Quantity | None = None
    monthly_consumption_kwh: Quantity | None = None

class Lead(BaseModel):
    contact_name: str
    phone: str
    company_name: str
    summary: str

class ChatRequest(BaseModel):
    session_id: str
    message: MessageText
    history: list[ChatMessage] | None = Field(default=None, max_length=100)
    client_data: ClientData = Field(default_factory=ClientData)
    lead_requested: StrictBool = False

class ChatResponse(BaseModel):
    answer: str
    client_data: ClientData
    sources: list[str]
    lead: Lead | None
    response_time_ms: int

