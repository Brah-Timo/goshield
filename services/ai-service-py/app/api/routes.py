"""FastAPI route definitions for the AI inference service."""
from fastapi import APIRouter, HTTPException, Depends, Request
from fastapi.responses import JSONResponse
import time

from app.schemas.inference import InferenceRequest, InferenceResponse, HealthResponse
from app.core.logging import logger

router = APIRouter()


def get_model(request: Request):
    """Dependency: retrieve the loaded FraudModel from app state."""
    model = request.app.state.model
    if not model or not model.is_loaded():
        raise HTTPException(status_code=503, detail="Model not ready")
    return model


@router.get("/health", response_model=HealthResponse, tags=["system"])
def health(request: Request):
    """Liveness probe — always returns 200 if service is up."""
    model = request.app.state.model
    return HealthResponse(
        status="ok",
        service="ai-service-py",
        version=request.app.state.settings.version,
        model_loaded=model.is_loaded() if model else False,
        model_version=model._version if model and model.is_loaded() else "none",
    )


@router.get("/readyz", tags=["system"])
def readiness(request: Request):
    """Readiness probe — 200 only when model is loaded."""
    model = request.app.state.model
    if not model or not model.is_loaded():
        return JSONResponse(
            status_code=503,
            content={"status": "not_ready", "reason": "model not loaded"},
        )
    return {"status": "ready"}


@router.post(
    "/analyze",
    response_model=InferenceResponse,
    tags=["inference"],
    summary="Analyze a claim for fraud",
    description="""
Accepts claim features, runs XGBoost inference, and returns:
- **fraud_score** (0.0–1.0): probability of fraud
- **risk_level**: LOW / MEDIUM / HIGH / CRITICAL
- **reason**: human-readable explanation
- **shap_values**: per-feature SHAP attribution for full explainability
    """,
)
def analyze_claim(
    req: InferenceRequest,
    model=Depends(get_model),
):
    """Primary inference endpoint. Returns fraud score + SHAP explanation."""
    try:
        logger.info(
            "inference request",
            claim_id=req.claim_id,
            amount=req.amount,
            claim_type=req.claim_type,
        )
        result = model.predict(req)
        logger.info(
            "inference complete",
            claim_id=req.claim_id,
            fraud_score=result.fraud_score,
            risk_level=result.risk_level,
            ms=result.processing_time_ms,
        )
        return result
    except Exception as exc:
        logger.error("inference error", claim_id=req.claim_id, error=str(exc))
        raise HTTPException(status_code=500, detail=f"Inference failed: {exc}") from exc


@router.post(
    "/batch-analyze",
    response_model=list[InferenceResponse],
    tags=["inference"],
    summary="Batch analyze multiple claims",
)
def batch_analyze(
    requests: list[InferenceRequest],
    model=Depends(get_model),
):
    """Analyze up to 50 claims in a single call."""
    if len(requests) > 50:
        raise HTTPException(
            status_code=400, detail="Batch size exceeds maximum of 50"
        )
    results = []
    for req in requests:
        try:
            results.append(model.predict(req))
        except Exception as exc:
            logger.error("batch inference error", claim_id=req.claim_id, error=str(exc))
            # Include a zero-score response on individual failure
            results.append(InferenceResponse(
                claim_id=req.claim_id,
                fraud_score=0.0,
                risk_level="LOW",
                reason=f"Inference error: {exc}",
                risk_factors=[],
                shap_values=[],
                model_version="error",
                confidence=0.0,
                processing_time_ms=0.0,
            ))
    return results
