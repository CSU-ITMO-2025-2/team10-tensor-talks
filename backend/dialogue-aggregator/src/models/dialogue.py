"""Dialogue state models"""

from pydantic import BaseModel, Field
from datetime import datetime
from typing import Dict, Any
from enum import Enum


class DialogueStatus(str, Enum):
    """Статусы диалога"""

    ACTIVE = "active"
    FINISHED = "finished"
    ERROR = "error"
    PAUSED = "paused"


class DialogueFlags(BaseModel):
    """Флаги состояния диалога"""

    is_finished: bool = False
    awaiting_llm: bool = False
    awaiting_user: bool = False
    has_error: bool = False


class DialogueState(BaseModel):
    """Состояние диалога в Redis"""

    chat_id: str
    user_id: str
    session_id: str
    dialogue_type: str
    status: DialogueStatus
    started_at: datetime
    last_activity: datetime
    messages_count: int = 0
    state_version: int = 0
    flags: DialogueFlags = Field(default_factory=DialogueFlags)
    metadata: Dict[str, Any] = Field(default_factory=dict)

    class Config:
        json_encoders = {datetime: lambda v: v.isoformat()}
        use_enum_values = True

