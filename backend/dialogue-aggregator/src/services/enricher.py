"""Event enrichment service"""

from datetime import datetime
from typing import Dict, Any

from ..models.messages import Message, MessageFullPayload, MessageRole
from ..models.dialogue import DialogueState
from ..logger import get_logger

logger = get_logger(__name__)


class EventEnricher:
    """Enriches messages with metadata and context"""

    def enrich_message(
        self, message: Message, dialogue_state: DialogueState
    ) -> MessageFullPayload:
        """Enrich message with dialogue context and metadata"""
        # Build dialogue context
        dialogue_context = self._build_dialogue_context(dialogue_state)

        # Build metadata
        metadata = {
            "user_id": dialogue_state.user_id,
            "message_index": dialogue_state.messages_count,
            "dialogue_context": dialogue_context,
        }

        # Determine source
        source = "user_input" if message.role == MessageRole.USER else "assistant_response"

        enriched = MessageFullPayload(
            chat_id=dialogue_state.chat_id,
            message_id=message.message_id,
            role=message.role,
            content=message.content,
            metadata=metadata,
            embeddings=None,  # Can be added later
            source=source,
            timestamp=message.timestamp,
            processed_at=datetime.utcnow(),
        )

        logger.debug(
            "Message enriched",
            chat_id=dialogue_state.chat_id,
            message_id=message.message_id,
            role=message.role,
        )

        return enriched

    def _build_dialogue_context(self, state: DialogueState) -> Dict[str, Any]:
        """Build dialogue context from state"""
        return {
            "total_messages": state.messages_count,
            "dialogue_type": state.dialogue_type,
            "status": state.status,
            "topic": state.metadata.get("topic"),
            "difficulty": state.metadata.get("difficulty"),
            "current_question_index": state.metadata.get("current_question_index"),
        }

    def calculate_message_index(self, chat_id: str, role: MessageRole) -> int:
        """Calculate message index (can be enhanced with Redis lookup)"""
        # This is a simplified version
        # In production, might need to query Redis for exact count
        return 0

