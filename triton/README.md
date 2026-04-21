# Triton Setup

This directory contains the first Triton model repository for InferFlow's deferred Triton backend path.

## Layout

- `qwen3_0_6b/config.pbtxt`: Triton model configuration
- `qwen3_0_6b/1/model.py`: Python backend implementation for prompt-to-text generation

## Runtime Assumptions

- Triton runs as a separate service or container.
- The recommended image is built from [Dockerfile.triton](../Dockerfile.triton), which extends the official Triton image with `torch` and `transformers`.
- The Python backend downloads the Hugging Face model at runtime using `MODEL_ID`, which defaults to `Qwen/Qwen3-0.6B`.

## Notes

- The current primary runtime path is EKS + vLLM, not Triton.
- The Kubernetes manifests mount this repository into the Triton pod via a ConfigMap for the lightweight source files; model weights are not checked into git.
- If Triton is reintroduced later, install the appropriate NVIDIA GPU support on the target cluster before scheduling the pod.
