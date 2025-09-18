FROM python:3.13-slim

WORKDIR /app

RUN apt-get update && apt-get install -y git

# Copy requirements first for better caching
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

RUN pip install "git+https://github.com/acroca/dapr-python-sdk.git@cross-app"
RUN pip install "git+https://github.com/acroca/dapr-python-sdk.git@cross-app#subdirectory=ext/dapr-ext-workflow"
RUN pip install "git+https://github.com/acroca/dapr-python-sdk.git@cross-app#subdirectory=ext/dapr-ext-grpc"
RUN pip install "git+https://github.com/acroca/dapr-durabletask-python.git@cross-app"

# Copy application code
COPY main.py .

# Set the default port
ENV APP_PORT=6010

# Expose the port
EXPOSE 6010

CMD ["python", "main.py"]
