"""Main application entry point"""

import signal
import threading
import uvicorn
from fastapi import FastAPI

from .config import settings
from .logger import setup_logger, get_logger
from .metrics import get_metrics_collector
from .redis_client import create_redis_client
from .kafka import create_kafka_producer, create_kafka_consumer
from .services import (
    StateManager,
    EventEnricher,
    EventRouter,
    EventHandlers,
)
from .health import create_health_app, health_checker
from .models.events import EventType

logger = get_logger(__name__)

# Global instances
redis_client = None
kafka_producer = None
consumers = []
event_router = None
running = True


def setup_components():
    """Initialize all components"""
    global redis_client, kafka_producer, event_router

    logger.info("Initializing components...")

    # Redis client
    redis_client = create_redis_client()
    health_checker.set_redis_client(redis_client.client)

    # Kafka producer
    kafka_producer = create_kafka_producer()
    health_checker.set_kafka_client(kafka_producer.producer)

    # Services
    state_manager = StateManager(redis_client)
    enricher = EventEnricher()

    # Event handlers
    handlers = EventHandlers(
        state_manager=state_manager,
        enricher=enricher,
        kafka_producer=kafka_producer,
        redis_client=redis_client,
    )

    # Event router
    event_router = EventRouter()
    event_router.register_handler(
        EventType.DIALOGUE_STARTED, handlers.handle_dialogue_started
    )
    event_router.register_handler(
        EventType.USER_MESSAGE_NEW, handlers.handle_user_message_new
    )
    event_router.register_handler(
        EventType.PHRASE_AGENT_GENERATED, handlers.handle_phrase_agent_generated
    )
    event_router.register_handler(
        EventType.MESSAGE_USER_SENT, handlers.handle_message_user_sent
    )

    logger.info("Components initialized successfully")


def consume_events(consumer, consumer_name: str):
    """Consume events from Kafka topic"""
    global running
    logger.info(f"Starting consumer: {consumer_name}")

    while running:
        try:
            events = consumer.poll(timeout=1.0)

            for event in events:
                try:
                    event_router.route_event(event)
                    consumer.commit()
                except Exception as e:
                    logger.error(
                        "Failed to process event",
                        consumer=consumer_name,
                        event_id=event.event_id if hasattr(event, 'event_id') else 'unknown',
                        error=str(e),
                        exc_info=True,
                    )
                    # Commit anyway to avoid reprocessing
                    consumer.commit()

        except Exception as e:
            logger.error(
                "Consumer error",
                consumer=consumer_name,
                error=str(e),
                exc_info=True,
            )
            if not running:
                break
            import time
            time.sleep(1)

    logger.info(f"Consumer stopped: {consumer_name}")


def start_consumers():
    """Start all Kafka consumers"""
    global consumers

    # Publication events consumer
    publication_consumer = create_kafka_consumer(
        topic=settings.kafka_topic_publication,
        group_id=settings.kafka_consumer_group_publication,
    )
    consumers.append(publication_consumer)
    thread1 = threading.Thread(
        target=consume_events,
        args=(publication_consumer, "publication"),
        daemon=True,
    )
    thread1.start()

    # Messages events consumer
    messages_consumer = create_kafka_consumer(
        topic=settings.kafka_topic_messages,
        group_id=settings.kafka_consumer_group_messages,
    )
    consumers.append(messages_consumer)
    thread2 = threading.Thread(
        target=consume_events,
        args=(messages_consumer, "messages"),
        daemon=True,
    )
    thread2.start()

    # Generated phrases consumer
    generated_consumer = create_kafka_consumer(
        topic=settings.kafka_topic_generated,
        group_id=settings.kafka_consumer_group_generated,
    )
    consumers.append(generated_consumer)
    thread3 = threading.Thread(
        target=consume_events,
        args=(generated_consumer, "generated"),
        daemon=True,
    )
    thread3.start()

    logger.info("All consumers started")


def shutdown():
    """Graceful shutdown"""
    global running, consumers, kafka_producer, redis_client

    logger.info("Shutting down...")
    running = False

    # Close consumers
    for consumer in consumers:
        try:
            consumer.close()
        except Exception as e:
            logger.error("Error closing consumer", error=str(e))

    # Close producer
    if kafka_producer:
        try:
            kafka_producer.close()
        except Exception as e:
            logger.error("Error closing producer", error=str(e))

    # Close Redis
    if redis_client:
        try:
            redis_client.close()
        except Exception as e:
            logger.error("Error closing Redis client", error=str(e))

    logger.info("Shutdown complete")


def signal_handler(signum, frame):
    """Handle shutdown signals"""
    logger.info(f"Received signal {signum}, initiating shutdown...")
    shutdown()


def create_app() -> FastAPI:
    """Create FastAPI application"""
    health_app = create_health_app()
    
    @health_app.on_event("startup")
    async def startup_event():
        """Startup event handler"""
        setup_logger()
        logger.info(f"Starting {settings.service_name} v{settings.service_version}")
        
        # Start metrics server
        metrics_collector = get_metrics_collector()
        metrics_collector.start_metrics_server()
        logger.info(f"Metrics server started on port {settings.metrics_port}")
        
        # Setup components
        setup_components()
        
        # Start consumers
        start_consumers()
        
        # Register signal handlers
        signal.signal(signal.SIGINT, signal_handler)
        signal.signal(signal.SIGTERM, signal_handler)
        
        logger.info("Application started successfully")
    
    @health_app.on_event("shutdown")
    async def shutdown_event():
        """Shutdown event handler"""
        shutdown()
    
    return health_app


def main():
    """Main entry point"""
    app = create_app()

    # Run with uvicorn on different port (metrics use 9091)
    uvicorn.run(
        app,
        host="0.0.0.0",
        port=8085,  # Different port from metrics (9091)
        log_config=None,  # Use our structured logging
    )


if __name__ == "__main__":
    main()

