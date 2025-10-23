FROM python:3.14-slim

WORKDIR /app
RUN apt-get update && apt-get install -y git

# Copy requirements first for better caching
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
# RUN pip install git+https://github.com/filintod/durabletask-python.git@filinto/fix-continue-as-new-lint

# Copy application code
COPY main.py .

# Set the default port
ENV APP_PORT=6021

# Expose the port
EXPOSE 6021

CMD ["python", "main.py"]
