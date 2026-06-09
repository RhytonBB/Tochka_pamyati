from fastapi import FastAPI, HTTPException
from fastapi.responses import JSONResponse
from pydantic import BaseModel
import os
import time

try:
    from glin_profanity import Filter
except Exception:
    Filter = None

from ml_toxic import ToxicityClassifier, resolve_default_model_paths

app = FastAPI(title="Text Filter Service", version="1.0.0")

filter_instance = None
toxicity_model = None

PROFANITY_MARKERS = [
    "бля",
    "бляд",
    "блять",
    "сука",
    "сучк",
    "хуй",
    "хуе",
    "пизд",
    "еба",
    "ебл",
    "мудак",
    "гондон",
    "нахер",
    "нахуй",
    "похуй",
]


class FilterRequest(BaseModel):
    text: str


@app.on_event("startup")
async def startup_event():
    global filter_instance, toxicity_model
    if Filter is None:
        filter_instance = None
    else:
        filter_instance = Filter(
            {
                "languages": ["russian", "english"],
                "detect_leetspeak": True,
                "leetspeak_level": "aggressive",
                "normalize_unicode": True,
                "cache_results": True,
                "max_cache_size": 5000,
            }
        )

    model_path = os.getenv("TOXIC_MODEL_PATH", "").strip()
    vocab_path = os.getenv("TOXIC_VOCAB_PATH", "").strip()
    if not model_path or not vocab_path:
        model_path, vocab_path = resolve_default_model_paths()
    try:
        print(f"Loading toxicity model from {model_path}...")
        toxicity_model = ToxicityClassifier(model_path=model_path, vocab_path=vocab_path, max_seq_len=128)
        print("Toxicity model loaded successfully.")
    except Exception as e:
        print(f"Failed to load toxicity model: {e}")
        import traceback
        traceback.print_exc()
        toxicity_model = None


@app.get("/health")
async def health_check():
    return {"status": "ok", "toxicity_model_loaded": toxicity_model is not None, "profanity_loaded": filter_instance is not None}


def _get_bool(value):
    if isinstance(value, bool):
        return value
    if isinstance(value, dict):
        return bool(value.get("contains_profanity"))
    return bool(getattr(value, "contains_profanity", False))


def _predict_toxicity(text: str) -> dict:
    if toxicity_model is None:
        return {"is_toxic": False, "toxic_score": 0.0, "available": False}
    out = toxicity_model.predict(text, threshold=0.5)
    out["available"] = True
    return out


def _contains_profanity_fallback(text: str) -> bool:
    normalized = (text or "").lower().replace("ё", "е")
    for marker in PROFANITY_MARKERS:
        if marker in normalized:
            return True
    return False


@app.post("/filter")
async def filter_text(payload: FilterRequest):
    start = time.time()
    text = payload.text or ""
    if len(text) > 10000:
        raise HTTPException(status_code=400, detail="Text too long")

    categories: list[str] = []
    scores: dict = {}

    if filter_instance is None:
        categories.append("profanity_service_unavailable")
    else:
        result = filter_instance.check_profanity(text)
        contains = _get_bool(result)
        if contains:
            categories.append("profanity")

    if "profanity" not in categories and _contains_profanity_fallback(text):
        categories.append("profanity")

    toxicity = _predict_toxicity(text)
    if not toxicity.get("available", False):
        raise HTTPException(status_code=503, detail="toxicity_model_not_loaded")
    scores["toxicity"] = toxicity.get("toxic_score", 0.0)
    if toxicity.get("is_toxic"):
        categories.append("toxicity")

    status = "flagged" if categories else "clean"
    message = "Text is clean" if status == "clean" else "Text requires manual review"

    elapsed_ms = int((time.time() - start) * 1000)
    return JSONResponse(
        content={
            "status": status,
            "categories": categories,
            "scores": scores,
            "message": message,
            "dur_ms": elapsed_ms,
        }
    )

