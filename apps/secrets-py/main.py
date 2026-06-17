from datetime import datetime
import logging
import os
import threading
import time

from flask import Flask, jsonify
from dapr.clients import DaprClient

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger('secrets-py')

app = Flask(__name__)

# State store whose redisPassword metadata is sourced from a Kubernetes secret
# via secretKeyRef. The point of this app is to exercise the component so that,
# when the referenced secret changes and the component hot-reloads, we can see
# the new secret value take effect (calls flip OK <-> error).
STATE_STORE = os.getenv('STATE_STORE_NAME', 'secret-statestore')
STATE_KEY = os.getenv('STATE_KEY', 'mykey')
LOOP_INTERVAL = float(os.getenv('LOOP_INTERVAL_SECONDS', '5'))


def exercise_state_store():
    """Save then get a key against the secret-backed state store; return the value read."""
    with DaprClient() as d:
        value = datetime.now().isoformat()
        d.save_state(store_name=STATE_STORE, key=STATE_KEY, value=value)
        resp = d.get_state(store_name=STATE_STORE, key=STATE_KEY)
        return resp.data.decode('utf-8') if resp.data else None


@app.route('/healthz', methods=['GET'])
def health_check():
    """Health check endpoint"""
    return jsonify({"status": "healthy", "timestamp": datetime.now().isoformat()}), 200


@app.route('/invoke', methods=['GET', 'POST'])
def invoke():
    """Do a single save/get against the secret-backed state store."""
    try:
        got = exercise_state_store()
        logger.info(f"[invoke] OK store={STATE_STORE} key={STATE_KEY} value={got}")
        return jsonify({"status": "OK", "store": STATE_STORE, "key": STATE_KEY, "value": got}), 200
    except Exception as e:
        logger.error(f"[invoke] ERROR store={STATE_STORE}: {e}")
        return jsonify({"status": "ERROR", "store": STATE_STORE, "error": str(e)}), 500


def loop():
    """Continuously exercise the state store so secret/component reloads are observable."""
    logger.info(f"Starting state-store loop every {LOOP_INTERVAL}s against '{STATE_STORE}'")
    # Give the Dapr sidecar a moment to come up before the first call.
    time.sleep(5)
    while True:
        try:
            got = exercise_state_store()
            logger.info(f"[loop] OK store={STATE_STORE} key={STATE_KEY} value={got}")
        except Exception as e:
            logger.error(f"[loop] ERROR store={STATE_STORE}: {e}")
        time.sleep(LOOP_INTERVAL)


if __name__ == '__main__':
    port = int(os.getenv('APP_PORT', '6005'))
    threading.Thread(target=loop, daemon=True).start()
    logger.info(f"Starting secrets-py on port {port}")
    app.run(host='0.0.0.0', port=port, debug=False)
