"""Health check endpoints"""

from fastapi import FastAPI, status
from fastapi.responses import JSONResponse, Response
from typing import Optional

from ..config import settings
from ..logger import get_logger

logger = get_logger(__name__)


class HealthChecker:
    """Health check manager"""

    def __init__(self):
        self.kafka_client = None
        self.redis_client = None

    def set_kafka_client(self, kafka_client):
        """Set Kafka client for health checks"""
        self.kafka_client = kafka_client

    def set_redis_client(self, redis_client):
        """Set Redis client for health checks"""
        self.redis_client = redis_client

    async def check_kafka(self) -> tuple[bool, dict]:
        """Check Kafka connectivity"""
        if self.kafka_client is None:
            return False, {"error": "Kafka client not initialized"}

        try:
            # Simple connectivity check: try to flush producer
            # This will attempt to connect and verify broker availability
            self.kafka_client.flush(timeout=5.0)
            return True, {"status": "healthy"}
        except Exception as e:
            logger.error("Kafka health check failed", error=str(e))
            return False, {"error": str(e)}

    async def check_redis(self) -> tuple[bool, dict]:
        """Check Redis connectivity"""
        if self.redis_client is None:
            return False, {"error": "Redis client not initialized"}

        try:
            # Simple ping
            result = self.redis_client.ping()
            if result:
                return True, {"status": "healthy"}
            return False, {"error": "Redis ping failed"}
        except Exception as e:
            logger.error("Redis health check failed", error=str(e))
            return False, {"error": str(e)}

    async def check_overall(self) -> tuple[bool, dict]:
        """Check overall health"""
        kafka_healthy, kafka_info = await self.check_kafka()
        redis_healthy, redis_info = await self.check_redis()

        overall_healthy = kafka_healthy and redis_healthy

        return overall_healthy, {
            "status": "healthy" if overall_healthy else "unhealthy",
            "kafka": kafka_info,
            "redis": redis_info,
        }


# Global health checker instance
health_checker = HealthChecker()


def create_health_app() -> FastAPI:
    """Create FastAPI app for health checks"""
    app = FastAPI(title="Dialogue Aggregator Health")

    @app.get("/health")
    async def health():
        """Overall health check"""
        healthy, info = await health_checker.check_overall()
        status_code = (
            status.HTTP_200_OK if healthy else status.HTTP_503_SERVICE_UNAVAILABLE
        )
        return JSONResponse(status_code=status_code, content=info)

    @app.get("/health/kafka")
    async def health_kafka():
        """Kafka health check"""
        healthy, info = await health_checker.check_kafka()
        status_code = (
            status.HTTP_200_OK if healthy else status.HTTP_503_SERVICE_UNAVAILABLE
        )
        return JSONResponse(status_code=status_code, content=info)

    @app.get("/health/redis")
    async def health_redis():
        """Redis health check"""
        healthy, info = await health_checker.check_redis()
        status_code = (
            status.HTTP_200_OK if healthy else status.HTTP_503_SERVICE_UNAVAILABLE
        )
        return JSONResponse(status_code=status_code, content=info)

    @app.get("/ready")
    async def ready():
        """Readiness probe"""
        # Service is ready if we can check both dependencies
        healthy, _ = await health_checker.check_overall()
        status_code = (
            status.HTTP_200_OK if healthy else status.HTTP_503_SERVICE_UNAVAILABLE
        )
        return JSONResponse(
            status_code=status_code,
            content={"status": "ready" if healthy else "not ready"},
        )

    @app.get("/metrics")
    async def metrics():
        """Prometheus metrics endpoint (redirects to metrics server)"""
        from prometheus_client import generate_latest, CONTENT_TYPE_LATEST

        from ..metrics import get_metrics_collector

        metrics_collector = get_metrics_collector()
        return Response(
            content=generate_latest(metrics_collector.registry),
            media_type=CONTENT_TYPE_LATEST,
        )

    return app

