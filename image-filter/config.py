import os

# --- Model Configuration ---
# A stable, public CLIP model for zero-shot image classification
MODEL_ID = os.getenv("MODEL_ID", "openai/clip-vit-base-patch32")

# Prompts used for binary classification
POSITIVE_PROMPT = os.getenv(
    "POSITIVE_PROMPT",
    "a photo of cultural heritage monument",
)
NEGATIVE_PROMPT = os.getenv(
    "NEGATIVE_PROMPT",
    "a photo of garbage or irrelevant content",
)

# --- Inference Configuration ---
# The confidence threshold for classifying an image as 'accepted'.
CONFIDENCE_THRESHOLD = float(os.getenv("CONFIDENCE_THRESHOLD", 0.05))

# --- Image Preprocessing ---
# The target size for the model input
IMAGE_WIDTH = 224
IMAGE_HEIGHT = 224

# The minimum allowed image size. Images smaller than this will be rejected.
MIN_IMAGE_WIDTH = 100
MIN_IMAGE_HEIGHT = 100

# --- Server Configuration ---
# The host and port for the FastAPI server.
APP_HOST = "0.0.0.0"
APP_PORT = 8000
