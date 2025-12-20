"""Event handlers"""

from datetime import datetime
from uuid import uuid4

from ..models.events import (
    KafkaEvent,
    EventType,
    DialogueStartedPayload,
    UserMessageNewPayload,
    PhraseAgentGeneratedPayload,
    MessageUserSentPayload,
)
from ..models.messages import Message, MessageRole
from ..models.dialogue import DialogueStatus
from ..services.state_manager import StateManager
from ..services.enricher import EventEnricher
from ..kafka import KafkaProducer
from ..redis_client import RedisClient
from ..config import settings
from ..logger import get_logger
from ..metrics import get_metrics_collector

logger = get_logger(__name__)


class EventHandlers:
    """Event handlers for different event types"""

    def __init__(
        self,
        state_manager: StateManager,
        enricher: EventEnricher,
        kafka_producer: KafkaProducer,
        redis_client: RedisClient,
    ):
        self.state_manager = state_manager
        self.enricher = enricher
        self.producer = kafka_producer
        self.redis = redis_client
        self.metrics = get_metrics_collector()

    def handle_dialogue_started(self, event: KafkaEvent) -> None:
        """Handle dialogue.started event"""
        with self.metrics.event_processing_duration.labels(
            event_type=EventType.DIALOGUE_STARTED.value
        ).time():
            logger.info(
                "Processing dialogue.started",
                event_id=event.event_id,
                chat_id=event.payload.get("chat_id"),
            )

            try:
                # Parse payload
                payload = DialogueStartedPayload(**event.payload)

                # Initialize dialogue
                state = self.state_manager.initialize_dialogue(
                    chat_id=payload.chat_id,
                    user_id=payload.user_id,
                    session_id=payload.session_id,
                    dialogue_type=payload.dialogue_type,
                    started_at=payload.started_at,
                )

                # Create system message
                system_message = Message(
                    message_id=str(uuid4()),
                    role=MessageRole.SYSTEM,
                    content=f"Dialogue started: {payload.dialogue_type}",
                    timestamp=payload.started_at,
                    processed_at=datetime.utcnow(),
                )

                # Add system message to dialogue
                self.state_manager.add_message_to_dialogue(
                    payload.chat_id, system_message
                )

                # Publish to history
                self._publish_history_event(
                    chat_id=payload.chat_id,
                    message=system_message,
                    state=state,
                    correlation_id=event.metadata.get("correlation_id")
                    if event.metadata
                    else None,
                )

                logger.info(
                    "Dialogue started processed",
                    event_id=event.event_id,
                    chat_id=payload.chat_id,
                )

            except Exception as e:
                logger.error(
                    "Failed to process dialogue.started",
                    event_id=event.event_id,
                    error=str(e),
                    exc_info=True,
                )
                raise

    def handle_user_message_new(self, event: KafkaEvent) -> None:
        """Handle user.message.new event"""
        with self.metrics.event_processing_duration.labels(
            event_type=EventType.USER_MESSAGE_NEW.value
        ).time():
            logger.info(
                "Processing user.message.new",
                event_id=event.event_id,
                chat_id=event.payload.get("chat_id"),
            )

            try:
                # Parse payload
                payload = UserMessageNewPayload(**event.payload)

                # Get dialogue state
                state = self.state_manager.get_dialogue_state(payload.chat_id)
                if not state:
                    raise ValueError(
                        f"Dialogue not found: {payload.chat_id}"
                    )

                if state.status != DialogueStatus.ACTIVE:
                    raise ValueError(
                        f"Dialogue is not active: {payload.chat_id}, status: {state.status}"
                    )

                # Create message
                message = Message(
                    message_id=payload.message_id,
                    role=MessageRole.USER,
                    content=payload.content,
                    timestamp=payload.timestamp,
                    processed_at=datetime.utcnow(),
                )

                # Add message to dialogue
                self.state_manager.add_message_to_dialogue(
                    payload.chat_id, message
                )

                # Update state flags
                self.state_manager.update_dialogue_state(
                    payload.chat_id,
                    {
                        "flags": state.flags.model_copy(
                            update={"awaiting_llm": True, "awaiting_user": False}
                        ),
                    },
                )

                # Get updated state
                updated_state = self.state_manager.get_dialogue_state(
                    payload.chat_id
                )

                # Enrich message
                enriched = self.enricher.enrich_message(message, updated_state)

                # Publish to messages.full.data
                self._publish_message_full_data(
                    enriched=enriched,
                    correlation_id=event.metadata.get("correlation_id")
                    if event.metadata
                    else None,
                )

                # Publish to history
                self._publish_history_event(
                    chat_id=payload.chat_id,
                    message=message,
                    state=updated_state,
                    correlation_id=event.metadata.get("correlation_id")
                    if event.metadata
                    else None,
                )

                logger.info(
                    "User message processed",
                    event_id=event.event_id,
                    chat_id=payload.chat_id,
                    message_id=payload.message_id,
                )

            except Exception as e:
                logger.error(
                    "Failed to process user.message.new",
                    event_id=event.event_id,
                    error=str(e),
                    exc_info=True,
                )
                raise

    def handle_phrase_agent_generated(self, event: KafkaEvent) -> None:
        """Handle phrase.agent.generated event"""
        with self.metrics.event_processing_duration.labels(
            event_type=EventType.PHRASE_AGENT_GENERATED.value
        ).time():
            logger.info(
                "Processing phrase.agent.generated",
                event_id=event.event_id,
                chat_id=event.payload.get("chat_id"),
            )

            try:
                # Parse payload
                payload = PhraseAgentGeneratedPayload(**event.payload)

                # Get dialogue state
                state = self.state_manager.get_dialogue_state(payload.chat_id)
                if not state:
                    raise ValueError(
                        f"Dialogue not found: {payload.chat_id}"
                    )

                # Create message
                message = Message(
                    message_id=payload.message_id,
                    role=MessageRole.ASSISTANT,
                    content=payload.generated_text,
                    timestamp=payload.timestamp,
                    processed_at=datetime.utcnow(),
                )

                # Add message to dialogue
                self.state_manager.add_message_to_dialogue(
                    payload.chat_id, message
                )

                # Update state flags
                self.state_manager.update_dialogue_state(
                    payload.chat_id,
                    {
                        "flags": state.flags.model_copy(
                            update={"awaiting_llm": False, "awaiting_user": True}
                        ),
                    },
                )

                # Get updated state
                updated_state = self.state_manager.get_dialogue_state(
                    payload.chat_id
                )

                # Publish to history
                self._publish_history_event(
                    chat_id=payload.chat_id,
                    message=message,
                    state=updated_state,
                    correlation_id=event.metadata.get("correlation_id")
                    if event.metadata
                    else None,
                )

                logger.info(
                    "Agent phrase processed",
                    event_id=event.event_id,
                    chat_id=payload.chat_id,
                    message_id=payload.message_id,
                )

            except Exception as e:
                logger.error(
                    "Failed to process phrase.agent.generated",
                    event_id=event.event_id,
                    error=str(e),
                    exc_info=True,
                )
                raise

    def handle_message_user_sent(self, event: KafkaEvent) -> None:
        """Handle message.user.sent event (replay/restore)"""
        # Check idempotency by message_id
        # For now, delegate to user_message_new handler
        # In production, should check if message already processed
        self.handle_user_message_new(event)

    def _publish_message_full_data(
        self, enriched: "MessageFullPayload", correlation_id: str = None
    ) -> None:
        """Publish enriched message to messages.full.data"""
        from ..models.events import KafkaEvent, EventType

        event = KafkaEvent(
            event_id=str(uuid4()),
            event_type=EventType.MESSAGE_FULL,
            timestamp=datetime.utcnow(),
            service=settings.service_name,
            version=settings.service_version,
            payload=enriched.model_dump(),
            metadata={"correlation_id": correlation_id} if correlation_id else None,
        )

        self.producer.publish(settings.kafka_topic_full_data, event)

    def _publish_history_event(
        self,
        chat_id: str,
        message: Message,
        state: "DialogueState",
        correlation_id: str = None,
    ) -> None:
        """Publish event to history.full.events"""
        from ..models.events import KafkaEvent, EventType
        from ..models.messages import HistoryEventPayload

        # Get recent messages for snapshot
        messages = self.redis.get_messages(chat_id, limit=10)

        payload = HistoryEventPayload(
            chat_id=chat_id,
            dialogue_state=state,
            message=message,
            snapshot={
                "chat_id": chat_id,
                "state_version": state.state_version,
                "recent_messages": [m.model_dump() for m in messages],
            },
        )

        event = KafkaEvent(
            event_id=str(uuid4()),
            event_type=EventType.HISTORY_EVENT,
            timestamp=datetime.utcnow(),
            service=settings.service_name,
            version=settings.service_version,
            payload=payload.model_dump(),
            metadata={"correlation_id": correlation_id} if correlation_id else None,
        )

        self.producer.publish(settings.kafka_topic_history, event)

