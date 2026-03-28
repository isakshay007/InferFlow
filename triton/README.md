# Triton Setup

This directory contains the first Triton model repository for InferFlow's AWS GPU path.

## Layout

- `qwen3_0_6b/config.pbtxt`: Triton model configuration
- `qwen3_0_6b/1/model.py`: Python backend implementation for prompt-to-text generation

## Runtime Assumptions

- Triton runs as a separate service or container.
- The recommended image is built from [Dockerfile.triton](/C:/Users/ajinf/Documents/CS%206650/InferFlow/Dockerfile.triton), which extends the official Triton image with `torch` and `transformers`.
- The Python backend downloads the Hugging Face model at runtime using `MODEL_ID`, which defaults to `Qwen/Qwen3-0.6B`.

## AWS Notes

- The first supported real runtime target is an AWS GPU environment.
- The Kubernetes manifests mount this repository into the Triton pod via a ConfigMap for the lightweight source files; model weights are not checked into git.
- Ensure the NVIDIA device plugin is installed on the EKS cluster before scheduling the Triton pod.
