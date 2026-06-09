from fastapi import FastAPI, File, UploadFile, HTTPException
from fastapi.responses import JSONResponse
import uvicorn
import time

import config
import model
import preprocess

# --- App Initialization ---
app = FastAPI(
    title="Cultural Heritage Image Filter",
    description="A microservice to detect if an image contains cultural heritage content.",
    version="1.0.0"
)

@app.on_event("startup")
async def startup_event():
    """Load the ML model on startup."""
    model.load_model()

# --- API Endpoints ---

@app.get("/health", tags=["Monitoring"])
async def health_check():
    """Endpoint to check if the service is running."""
    return {"status": "ok"}

@app.post("/filter", tags=["Filtering"])
async def filter_image(file: UploadFile = File(...)):
    """
    Receives an image, processes it, and classifies it as 'Accepted' or 'Garbage'.
    """
    start_time = time.time()

    # 1. Read image bytes
    image_bytes = await file.read()

    # 2. Preprocess the image
    preprocess_start = time.time()
    processed_image, error_message = preprocess.process_image(image_bytes)
    preprocess_time = time.time() - preprocess_start

    if error_message:
        print(f"Preprocessing error: {error_message}")
        raise HTTPException(status_code=400, detail=error_message)

    # 3. Perform inference
    inference_start = time.time()
    try:
        predicted_label, confidence = model.predict(processed_image)
    except RuntimeError as e:
        raise HTTPException(status_code=503, detail=str(e))
    inference_time = time.time() - inference_start

    if predicted_label is None:
        raise HTTPException(status_code=500, detail="Model inference failed.")

    # 4. Determine status based on threshold
    final_status = "garbage"
    message = "Image does not contain cultural heritage content"
    if predicted_label.lower() == 'accepted' and confidence >= config.CONFIDENCE_THRESHOLD:
        final_status = "accepted"
        message = "Image contains cultural heritage content"

    total_time = time.time() - start_time

    # Logging
    print(f"Request finished in {total_time:.4f}s | Preprocessing: {preprocess_time:.4f}s | Inference: {inference_time:.4f}s")
    print(f"Result: {final_status.upper()} with confidence {confidence:.2f}")

    return JSONResponse(content={
        "status": final_status,
        "confidence": confidence,
        "message": message
    })

# --- Main Execution ---
if __name__ == "__main__":
    uvicorn.run(
        "main:app",
        host=config.APP_HOST,
        port=config.APP_PORT,
        reload=True
    )
