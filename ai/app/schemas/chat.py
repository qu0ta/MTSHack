from typing import Annotated, Literal

from pydantic import BaseModel, Field, StringConstraints, field_validator

MessageText = Annotated[str, StringConstraints(strip_whitespace=True, min_length=1, max_length=10000)]
OptionalText = Annotated[str, StringConstraints(min_length=1, max_length=500)] | None
NonnegativeNumber = Annotated[float, Field(ge=0, allow_inf_nan=False)] | None


class ChatMessage(BaseModel):
    role: Literal["user", "assistant"]
    content: MessageText


class ClientData(BaseModel):
    company_name: Annotated[str, StringConstraints(min_length=1, max_length=255)] | None = None
    contact_name: Annotated[str, StringConstraints(min_length=1, max_length=255)] | None = None
    phone: Annotated[str, StringConstraints(min_length=1, max_length=30)] | None = None
    industry: OptionalText = None
    region: OptionalText = None
    connected_power_kw: NonnegativeNumber = None
    monthly_consumption_kwh: NonnegativeNumber = None

    @field_validator("company_name", "contact_name", "phone", "industry", "region", mode="before")
    @classmethod
    def empty_backend_field_is_unknown(cls, value):
        # Go/form clients may send empty optional text instead of null.
        return None if isinstance(value, str) and not value.strip() else value


class Lead(BaseModel):
    contact_name: str
    phone: str
    company_name: str
    summary: str


class ChatRequest(BaseModel):
    session_id: Annotated[str, StringConstraints(strip_whitespace=True, min_length=1, max_length=200)]
    message: MessageText
    history: list[ChatMessage] = Field(default_factory=list, max_length=100)
    client_data: ClientData = Field(default_factory=ClientData)
    lead_requested: bool = Field(default=False, strict=True)

    @field_validator("history", mode="before")
    @classmethod
    def accept_empty_go_slice(cls, value):
        # encoding/json serializes a nil Go slice as null.
        return [] if value is None else value


class ChatResponse(BaseModel):
    answer: str
    client_data: ClientData
    sources: list[str]
    lead: Lead | None
    response_time_ms: int = Field(ge=0)
    degraded: list[str] = Field(default_factory=list)
    timings_ms: dict[str, float] = Field(default_factory=dict)
