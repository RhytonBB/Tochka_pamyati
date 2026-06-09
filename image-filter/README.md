# Image Filter Microservice

This microservice is a lightweight image classifier designed to detect cultural heritage content. It uses the `sbrzz/cultural-arts-shield-v0` model from Hugging Face and provides a simple FastAPI interface.

## Tech Stack

- Python 3.10+
- FastAPI
- PyTorch (CPU)
- Transformers by Hugging Face
- Pillow

## Setup and Installation

Follow these steps to set up and run the service locally.

### 1. Create a Virtual Environment

It is highly recommended to use a virtual environment to manage dependencies.

```bash
python -m venv venv
```

### 2. Activate the Virtual Environment

- **On Windows:**
  ```bash
  .\venv\Scripts\activate
  ```
- **On Linux/macOS:**
  ```bash
  source venv/bin/activate
  ```

### 3. Install Dependencies

Install all the required Python packages from `requirements.txt`.

```bash
pip install -r requirements.txt
```

## Running the Service

You can run the service in one of the following ways:

**Option 1: Using the run script (for Windows)**

Simply execute the `run.bat` script.

```bash
run.bat
```

**Option 2: Using uvicorn directly**

This command will start the server with hot-reloading enabled.

```bash
uvicorn main:app --host 0.0.0.0 --port 8000 --reload
```

The service will be available at `http://localhost:8000`.

## API Endpoints

### Health Check

- **Endpoint**: `GET /health`
- **Description**: Returns the operational status of the service.
- **Success Response**:
  ```json
  {
    "status": "ok"
  }
  ```

### Filter Image

- **Endpoint**: `POST /filter`
- **Description**: Analyzes an uploaded image.
- **Request**: `multipart/form-data` with a single field `file` containing the image.
- **Success Response (`accepted`):**
  ```json
  {
    "status": "accepted",
    "confidence": 0.87,
    "message": "Image contains cultural heritage content"
  }
  ```
- **Success Response (`garbage`):**
  ```json
  {
    "status": "garbage",
    "confidence": 0.92,
    "message": "Image does not contain cultural heritage content"
  }
  ```
- **Error Response**:
  ```json
  {
    "detail": "Error message here..."
  }
  ```

## Configuration

You can configure the service using environment variables:

- `CONFIDENCE_THRESHOLD`: The minimum confidence score (0.0 to 1.0) for an image to be classified as `accepted`. Defaults to `0.5`.
