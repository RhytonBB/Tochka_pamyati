from PIL import Image
import io
import config

def process_image(image_bytes: bytes):
    """
    Processes raw image bytes into a model-ready tensor.

    Args:
        image_bytes: The raw bytes of the image file.

    Returns:
        A tuple of (PIL.Image.Image, str). The image is validated and converted,
        or None if an error occurred. The string is an error message, or empty.
    """
    try:
        image = Image.open(io.BytesIO(image_bytes)).convert("RGB")
    except Exception as e:
        return None, f"Failed to decode image: {e}"

    # Check if the image is too small
    if image.width < config.MIN_IMAGE_WIDTH or image.height < config.MIN_IMAGE_HEIGHT:
        return None, f"Image is too small ({image.width}x{image.height}). Minimum is {config.MIN_IMAGE_WIDTH}x{config.MIN_IMAGE_HEIGHT}."

    return image, ""
