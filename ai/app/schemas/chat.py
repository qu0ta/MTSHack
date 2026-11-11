from typing import Literal 

from pydantic import BaseModel

class ChatMessage(BaseModel):
    role: Literal["user", "assistant"]
    content: str

class ClientData(BaseModel):
    company_name: str | None = None
    contact_name: str | None = None
    phone: str | None = None
    industry: str | None = None
    region: str | None = None
    connected_power_kw: float | None = None
    monthly_consumption_kwh: float | None = None

class Lead(BaseModel):
    contact_name: str
    phone: str
    company_name: str
    summary: str

class ChatRequest(BaseModel):
    session_id: str
    message: str
    history: list[ChatMessage]
    client_data: ClientData

class ChatResponse(BaseModel):
    answer: str
    client_data: ClientData
    sources: list[str]
    lead: Lead | None
    response_time_ms: int

