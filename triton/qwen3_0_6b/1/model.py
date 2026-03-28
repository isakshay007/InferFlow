import json
import os

import numpy as np
import torch
import triton_python_backend_utils as pb_utils
from transformers import AutoModelForCausalLM, AutoTokenizer


class TritonPythonModel:
    def initialize(self, args):
        model_config = json.loads(args["model_config"])
        self.model_name = model_config["name"]
        self.model_id = os.getenv("MODEL_ID", "Qwen/Qwen3-0.6B")
        self.device = "cuda" if torch.cuda.is_available() else "cpu"

        self.tokenizer = AutoTokenizer.from_pretrained(
            self.model_id,
            trust_remote_code=True,
        )
        torch_dtype = torch.float16 if self.device == "cuda" else torch.float32
        self.model = AutoModelForCausalLM.from_pretrained(
            self.model_id,
            trust_remote_code=True,
            torch_dtype=torch_dtype,
        )
        self.model.to(self.device)
        self.model.eval()

    def execute(self, requests):
        responses = []
        for request in requests:
            prompt_tensor = pb_utils.get_input_tensor_by_name(request, "prompt")
            max_new_tokens_tensor = pb_utils.get_input_tensor_by_name(
                request, "max_new_tokens"
            )

            prompt = prompt_tensor.as_numpy().reshape(-1)[0]
            if isinstance(prompt, bytes):
                prompt = prompt.decode("utf-8")

            max_new_tokens = int(max_new_tokens_tensor.as_numpy().reshape(-1)[0])

            inputs = self.tokenizer(prompt, return_tensors="pt").to(self.device)
            with torch.no_grad():
                output_ids = self.model.generate(
                    **inputs,
                    max_new_tokens=max_new_tokens,
                    do_sample=False,
                )

            generated_ids = output_ids[0][inputs["input_ids"].shape[1] :]
            generated_text = self.tokenizer.decode(
                generated_ids,
                skip_special_tokens=True,
            ).strip()

            output = np.array([generated_text], dtype=object)
            responses.append(
                pb_utils.InferenceResponse(
                    output_tensors=[pb_utils.Tensor("generated_text", output)]
                )
            )

        return responses

    def finalize(self):
        return
