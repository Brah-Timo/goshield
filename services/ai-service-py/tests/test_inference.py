"""Unit tests for the AI inference service."""
import pytest
from app.models.fraud_model import FraudModel, _engineer_features, _risk_level
from app.schemas.inference import InferenceRequest


@pytest.fixture(scope="module")
def model():
    m = FraudModel(model_path="/nonexistent/model.joblib")
    m.load()  # falls back to synthetic model
    return m


def make_request(**kwargs) -> InferenceRequest:
    defaults = dict(
        claim_id="test-claim-001",
        amount=500.0,
        claim_type="HEALTH",
        description="Routine medical checkup",
        policy_number="POL-001",
        user_id="user-001",
        company_id="company-001",
        account_age_days=365,
        prior_claims_count=0,
        incident_date="2024-03-15",
    )
    defaults.update(kwargs)
    return InferenceRequest(**defaults)


def test_feature_engineering_shape():
    req = make_request()
    df = _engineer_features(req)
    assert df.shape == (1, 10), "Expected 10 feature columns"


def test_feature_is_new_account_true():
    req = make_request(account_age_days=30)
    df = _engineer_features(req)
    assert df["is_new_account"].iloc[0] == 1


def test_feature_is_new_account_false():
    req = make_request(account_age_days=200)
    df = _engineer_features(req)
    assert df["is_new_account"].iloc[0] == 0


def test_feature_high_amount_flag():
    req = make_request(amount=15_000)
    df = _engineer_features(req)
    assert df["high_amount_flag"].iloc[0] == 1


def test_risk_level_thresholds():
    assert _risk_level(0.95) == "CRITICAL"
    assert _risk_level(0.80) == "HIGH"
    assert _risk_level(0.60) == "MEDIUM"
    assert _risk_level(0.30) == "LOW"


def test_inference_low_risk_claim(model):
    req = make_request(
        amount=500, account_age_days=1000, prior_claims_count=0
    )
    result = model.predict(req)
    assert 0.0 <= result.fraud_score <= 1.0
    assert result.claim_id == req.claim_id
    assert result.model_version is not None
    assert len(result.shap_values) > 0


def test_inference_high_risk_claim(model):
    req = make_request(
        amount=95_000,
        account_age_days=7,
        prior_claims_count=8,
        claim_type="PROPERTY",
        incident_date="2024-12-31",
    )
    result = model.predict(req)
    # High-risk pattern should score above 0.5
    assert result.fraud_score > 0.5, f"Expected high fraud score, got {result.fraud_score}"


def test_inference_has_shap_explanation(model):
    req = make_request(amount=1000, account_age_days=500)
    result = model.predict(req)
    assert len(result.shap_values) > 0
    for sv in result.shap_values:
        assert sv.importance >= 0
        assert sv.direction in ("increases_risk", "decreases_risk")


def test_inference_processing_time(model):
    req = make_request()
    result = model.predict(req)
    assert result.processing_time_ms > 0
    assert result.processing_time_ms < 5000  # should be fast


def test_inference_risk_factors_list(model):
    req = make_request(amount=50_000, account_age_days=10, prior_claims_count=5)
    result = model.predict(req)
    assert isinstance(result.risk_factors, list)
