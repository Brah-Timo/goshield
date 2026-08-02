"""GoShield AI Service — FastAPI application factory."""
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse, Response
from prometheus_fastapi_instrumentator import Instrumentator

from app.api.routes import router
from app.core.config import settings
from app.core.logging import configure_logging, logger
from app.models.fraud_model import FraudModel


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan: load model on startup, clean up on shutdown."""
    configure_logging(settings.log_level)
    logger.info("ai-service-py starting", version=settings.version, env=settings.environment)

    # Load the fraud model (non-blocking background would be ideal in production)
    model = FraudModel(
        model_path=settings.model_path,
        preprocessor_path=settings.preprocessor_path,
    )
    model.load()
    app.state.model = model
    app.state.settings = settings

    logger.info("ai-service-py ready")
    yield

    logger.info("ai-service-py shutting down")
    app.state.model = None


def create_app() -> FastAPI:
    app = FastAPI(
        title="GoShield AI Inference Service",
        description=(
            "XGBoost-based insurance fraud detection with SHAP explainability. "
            "Every prediction includes per-feature attribution scores for full transparency."
        ),
        version=settings.version,
        docs_url="/docs",
        redoc_url="/redoc",
        lifespan=lifespan,
    )

    # CORS
    app.add_middleware(
        CORSMiddleware,
        allow_origins=settings.cors_origins,
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )

    # Prometheus metrics
    Instrumentator(
        should_group_status_codes=False,
        should_respect_env_var=False,
        should_instrument_requests_inprogress=True,
    ).instrument(app).expose(app, endpoint="/metrics")

    # Routes
    app.include_router(router, prefix="/api/v1")

    # ── Root landing endpoint ───────────────────────────────────────────────────
    @app.get("/", include_in_schema=False)
    def root():
        """Service status — shown when opening the service in a browser."""
        model = getattr(app.state, "model", None)
        return JSONResponse({
            "service": settings.service_name,
            "status": "ok",
            "version": settings.version,
            "environment": settings.environment,
            "model": settings.model_version,
            "model_loaded": model.is_loaded() if model else False,
            "docs": "/docs",
            "health": "/api/v1/health",
            "readyz": "/api/v1/readyz",
        })

    # Suppress browser favicon 404 log spam.
    @app.get("/favicon.ico", include_in_schema=False)
    def favicon():
        return Response(status_code=204)

    return app


app = create_app()
