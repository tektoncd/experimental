
# Readme for Generating Tekton Pipeline SDK

The guide shows how to generate the openapi model and swagger.json file from Tekton Pipeline types using `openapi-gen` and generate Tekton Pipeline Python SDK Client for the Python object models using `openapi-generator`. Also show how to upload the Tekton Pipeline SDK to Pypi.

## Update openapi spec and swagger file.

Download `tektoncd/pipeline` repository, and execute the below script to generate openapi spec and swagger file.

```
./hack/update-openapigen.sh
```
After executing, the `openapi_generated.go` and `swagger.json` are refreshed under `pkg/apis/pipeline/v1beta1/`.

And then copy the `pkg/apis/pipeline/v1beta1/swagger.json` to the `sdk/hack` in this repo. If not copy, the `sdk-gen.sh` will download from github directly.

## Generate Tekton Pipeline Python SDK

Execute the script `/sdk/hack/sdk-gen.sh` to install openapi-generator and generate Tekton Pipeline Python SDK.

```
./sdk/hack/sdk-gen.sh
```
After the script execution, the Tekton Pipeline Python SDK is generated in the `./sdk/python` directory. Some files such as [README](../python/README.md) and setup.py need to be merged manually after the script execution.

## (Optional) Refresh Python SDK in the Pypi

Navigate to `sdk/python/tekton` directory.

1. Install `twine`:

   ```bash
   pip install twine
   ```

2. Update the Tekton Pipeline Python SDK version in the [setup.py](../python/setup.py).

3. Create some distributions in the normal way:

    ```bash
    python setup.py sdist bdist_wheel
    ```

4. Upload with twine to [Test PyPI](https://packaging.python.org/guides/using-testpypi/) and verify things look right. `Twine` will automatically prompt for your username and password:
    ```bash
    twine upload --repository-url https://test.pypi.org/legacy/ dist/*
    username: ...
    password:
    ...
    ```

5. Upload to [PyPI](https://pypi.org/project/tekton-pipeline/):
    ```bash
    twine upload dist/*
    ```
