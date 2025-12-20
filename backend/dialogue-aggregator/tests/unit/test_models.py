"""Unit tests for models"""

import pytest
from datetime import datetime
from uuid import uuid4

from src.models.events import (
    KafkaEvent,
    EventType,
    DialogueStartedPayload,
    UserMessageNewPayload,
)
from src.models.dialogue import DialogueState, DialogueStatus, DialogueFlags
from src.models.messages import Message, MessageRole


def test_kafka_event_creation():
    """Test KafkaEvent model creation"""
    event = KafkaEvent(
        event_id=str(uuid4()),
        event_type=EventType.DIALOGUE_STARTED,
        timestamp=datetime.utcnow(),
        service="test-service",
        version="1.0.0",
        payload={"chat_id": "test-chat"},
    )

    assert event.event_type == EventType.DIALOGUE_STARTED
    assert event.service == "test-service"
    assert "chat_id" in event.payload


def test_dialogue_state_creation():
    """Test DialogueState model creation"""
    state = DialogueState(
        chat_id="test-chat",
        user_id="test-user",
        session_id="test-session",
        dialogue_type="ml_interview",
        status=DialogueStatus.ACTIVE,
        started_at=datetime.utcnow(),
        last_activity=datetime.utcnow(),
    )

    assert state.chat_id == "test-chat"
    assert state.status == DialogueStatus.ACTIVE
    assert state.messages_count == 0
    assert state.state_version == 0


def test_message_creation():
    """Test Message model creation"""
    message = Message(
        message_id=str(uuid4()),
        role=MessageRole.USER,
        content="Test message",
        timestamp=datetime.utcnow(),
    )

    assert message.role == MessageRole.USER
    assert message.content == "Test message"

