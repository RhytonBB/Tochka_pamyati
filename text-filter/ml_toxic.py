from pathlib import Path


class ToxicityClassifier:
    def __init__(self, model_path: str, vocab_path: str, max_seq_len: int = 128):
        from ml_vocab_preprocessor import TextPreprocessor
        import torch

        self._torch = torch
        import torch.nn.functional as F
        self._F = F
        self.preprocessor = TextPreprocessor(vocab_path=vocab_path, max_len=max_seq_len)
        self.model = torch.jit.load(model_path)
        self.model.eval()

    def predict(self, text: str, threshold: float = 0.5):
        input_tensor = self.preprocessor.process(text)

        with self._torch.no_grad():
            logits = self.model(input_tensor)
            probabilities = self._F.softmax(logits, dim=1).squeeze()

        if probabilities.dim() == 0:
            score_toxic = float(probabilities.item())
        else:
            score_toxic = float(probabilities[1].item())

        return {
            "is_toxic": score_toxic > threshold,
            "toxic_score": score_toxic,
        }


def resolve_default_model_paths() -> tuple[str, str]:
    # Use relative paths to avoid non-ASCII issues in absolute paths on Windows
    model_path = "models/v1.0/solo_cnn_int8.pth"
    vocab_path = "models/v1.0/vocab.json"
    return model_path, vocab_path
