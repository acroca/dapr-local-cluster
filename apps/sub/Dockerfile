FROM python:3.11-slim

WORKDIR /app

# Copy requirements first for better caching
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY main.py .

# Set the default port
ENV APP_PORT=6005

# Expose the port
EXPOSE 6005

CMD ["python", "main.py"]
