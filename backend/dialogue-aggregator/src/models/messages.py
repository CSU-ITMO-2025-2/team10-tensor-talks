"""Message models"""

from pydantic import BaseModel, Field
from datetime import datetime
from typing import Optional, Dict, Any, List
from enum import Enum

from .dialogue import DialogueState


class MessageRole(str, Enum):
    """Роли сообщений"""

    USER = "user"
    ASSISTANT = "assistant"
    SYSTEM = "system"


class Message(BaseModel):
    """Модель сообщения"""

    message_id: str
    role: MessageRole
    content: str
    timestamp: datetime
    processed_at: Optional[datetime] = None

    class Config:
        json_encoders = {datetime: lambda v: v.isoformat()}
        use_enum_values = True


class MessageFullPayload(BaseModel):
    """Payload для messages.full.data"""

    chat_id: str
    message_id: str
    role: MessageRole
    content: str
    metadata: Dict[str, Any]
    embeddings: Optional[List[float]] = None
    source: str
    timestamp: datetime
    processed_at: datetime

    class Config:
        json_encoders = {datetime: lambda v: v.isoformat()}
        use_enum_values = True


class HistoryEventPayload(BaseModel):
    """Payload для history.full.events"""

    chat_id: str
    dialogue_state: DialogueState
    message: Optional[Message] = None
    snapshot: Optional[Dict[str, Any]] = None

    class Config:
        json_encoders = {datetime: lambda v: v.isoformat()}

