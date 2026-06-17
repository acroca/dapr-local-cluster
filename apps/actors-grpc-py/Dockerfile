FROM python:3.11-slim

WORKDIR /app

# git is needed to install dapr from the GitHub branch in requirements.txt
RUN apt-get update && apt-get install -y --no-install-recommends git && rm -rf /var/lib/apt/lists/*

# Copy requirements first for better caching
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY src .

# Set the default port
ENV APP_PORT=6011

# Expose the port
EXPOSE 6011

CMD ["python", "main.py"]
