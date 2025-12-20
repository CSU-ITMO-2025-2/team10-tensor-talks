"""Event models for Kafka messages"""

from pydantic import BaseModel, Field
from datetime import datetime
from typing import Optional, Dict, Any, List
from enum import Enum


class EventType(str, Enum):
    """Типы событий"""

    # Publication events
    DIALOGUE_STARTED = "dialogue.started"
    USER_MESSAGE_NEW = "user.message.new"
    DIALOGUE_CONTINUED = "dialogue.continued"
    USER_ACTION = "user.action"

    # Messages events
    MESSAGE_USER_SENT = "message.user.sent"
    MESSAGE_ASSISTANT_GENERATED = "message.assistant.generated"
    MESSAGE_SYSTEM_CREATED = "message.system.created"

    # Generated phrases
    PHRASE_AGENT_GENERATED = "phrase.agent.generated"

    # Full data events
    MESSAGE_FULL = "message.full"
    HISTORY_EVENT = "history.event"


class KafkaEvent(BaseModel):
    """Базовая модель события Kafka"""

    event_id: str = Field(..., description="Уникальный ID события (UUID)")
    event_type: EventType = Field(..., description="Тип события")
    timestamp: datetime = Field(..., description="Время создания события (ISO 8601 UTC)")
    service: str = Field(..., description="Имя сервиса, создавшего событие")
    version: str = Field(..., description="Версия сервиса")
    payload: Dict[str, Any] = Field(..., description="Данные события")
    metadata: Optional[Dict[str, Any]] = Field(
        None, description="Дополнительные метаданные"
    )

    class Config:
        json_encoders = {datetime: lambda v: v.isoformat()}
        use_enum_values = True


# Payload models for specific event types
class DialogueStartedPayload(BaseModel):
    """Payload для dialogue.started"""

    chat_id: str
    user_id: str
    session_id: str
    dialogue_type: str = "ml_interview"
    started_at: datetime

    class Config:
        json_encoders = {datetime: lambda v: v.isoformat()}


class UserMessageNewPayload(BaseModel):
    """Payload для user.message.new"""

    chat_id: str
    user_id: str
    message_id: str
    content: str
    timestamp: datetime

    class Config:
        json_encoders = {datetime: lambda v: v.isoformat()}


class PhraseAgentGeneratedPayload(BaseModel):
    """Payload для phrase.agent.generated"""

    chat_id: str
    message_id: str
    generated_text: str
    confidence: Optional[float] = None
    intermediate_steps: Optional[List[Dict[str, Any]]] = None
    metadata: Optional[Dict[str, Any]] = None
    timestamp: datetime

    class Config:
        json_encoders = {datetime: lambda v: v.isoformat()}


class MessageUserSentPayload(BaseModel):
    """Payload для message.user.sent"""

    chat_id: str
    message_id: str
    user_id: str
    content: str
    timestamp: datetime

    class Config:
        json_encoders = {datetime: lambda v: v.isoformat()}

