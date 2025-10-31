from datetime import datetime
from time import sleep
import logging
import os
import threading
import random

from flask import Flask, jsonify
from dapr.clients import DaprClient
from dapr.clients.http.client import DaprHttpClient
from dapr.ext.grpc import App, BindingRequest
from dapr.serializers import DefaultJSONSerializer

logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger('Bindings')

web_app = Flask(__name__)
bindings_app = App()

@web_app.route('/healthz', methods=['GET'])
def health_check():
    """Health check endpoint"""
    return jsonify({"status": "healthy", "timestamp": datetime.now().isoformat()}), 200

@web_app.route('/start', methods=['POST'])
def start():
    """Start"""

    # with DaprClient() as d:
    d = DaprHttpClient(DefaultJSONSerializer())
    req_data = "HELLO"

    print(f'Sending message: {req_data}', flush=True)
    resp = d.invoke_binding('testbinding', 'create', req_data)
    print(f'Response: {resp}', flush=True)

    return jsonify({"status": "OK"}), 200


@bindings_app.binding('testbinding')
def binding(request: BindingRequest):
    print(f'Received message: {request.text()}', flush=True)

def main():
    # Run both servers in separate threads
    threads = []

    def run_bindings_app():
        port = int(os.getenv('APP_PORT', 50051))
        print(f'Running bindings app on port {port}', flush=True)
        bindings_app.run(port)

    def run_web_app():
        port = int(os.getenv('WEB_PORT', 6005))
        print(f'Running web app on port {port}', flush=True)
        web_app.run(host='0.0.0.0', port=port, debug=False)

    t2 = threading.Thread(target=run_web_app, daemon=True)
    t1 = threading.Thread(target=run_bindings_app, daemon=True)

    t1.start()
    t2.start()

    # Keep the main thread alive as long as one of the child threads is alive
    t1.join()
    t2.join()

if __name__ == '__main__':
    main()
