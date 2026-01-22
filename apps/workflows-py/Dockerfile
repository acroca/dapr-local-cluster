FROM python:3.11-slim

WORKDIR /app


# Copy requirements first for better caching
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

## Use the following to install the dependencies from the source code
RUN apt-get update && apt-get install -y git
COPY --from=python-sdk . /python-sdk
COPY --from=durabletask-python . /durabletask-python
RUN pip uninstall -y dapr || true && pip install --no-cache-dir /python-sdk
RUN pip uninstall -y dapr-ext-workflow || true && pip install --no-cache-dir /python-sdk/ext/dapr-ext-workflow
RUN pip uninstall -y durabletask-python || true && pip install --no-cache-dir /durabletask-python


# Copy application code
COPY src .

# Set the default port
ENV APP_PORT=6005

# Expose the port
EXPOSE 6005

CMD ["python", "main.py"]
