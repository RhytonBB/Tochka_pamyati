from transformers import CLIPModel, CLIPProcessor
import torch
import config

# --- Globals ---
model = None
processor = None
load_error = None

def load_model():
    """
    Loads the pre-trained image classification model and feature extractor
    from Hugging Face. This function is called once at startup.
    """
    global model, processor, load_error
    if model is None:
        print(f"Loading model '{config.MODEL_ID}' for the first time...")
        try:
            processor = CLIPProcessor.from_pretrained(config.MODEL_ID)
            model = CLIPModel.from_pretrained(config.MODEL_ID)
            model.eval()  # Set the model to evaluation mode
            load_error = None
            print("Model loaded successfully.")
        except Exception as e:
            model = None
            processor = None
            load_error = (
                f"Failed to load model '{config.MODEL_ID}'. "
                f"Make sure this is a valid CLIP model. "
                f"Original error: {e}"
            )
            print(load_error)

def predict(image):
    """
    Performs inference on a preprocessed image tensor.

    Args:
        image_tensor: The processed image tensor with a batch dimension.

    Returns:
        A tuple of (str, float) representing the predicted label and its confidence score.
        Returns (None, 0.0) if an error occurs.
    """
    if model is None or processor is None:
        if load_error:
            raise RuntimeError(load_error)
        raise RuntimeError("Model has not been loaded. Call load_model() first.")

    with torch.no_grad():
        texts = [config.POSITIVE_PROMPT, config.NEGATIVE_PROMPT]
        inputs = processor(text=texts, images=image, return_tensors="pt", padding=True)
        outputs = model(**inputs)

        # Similarity of image to each text prompt
        logits_per_image = outputs.logits_per_image
        probs = logits_per_image.softmax(dim=1)[0]
        positive_conf = probs[0].item()
        negative_conf = probs[1].item()

        if positive_conf >= negative_conf:
            return "accepted", positive_conf
        return "garbage", negative_conf
