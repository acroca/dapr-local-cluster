FROM python:3.11-slim

RUN apt-get update && apt-get install -y git

WORKDIR /app

# Copy requirements first for better caching
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY app.py .
COPY .env .

# Set the default port
ENV APP_PORT=8004

# Expose the port
EXPOSE 8004

CMD ["python", "app.py"]
