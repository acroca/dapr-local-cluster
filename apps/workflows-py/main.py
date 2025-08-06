from datetime import datetime
import random
from time import sleep
import logging
import os
import signal
import sys
import threading

from flask import Flask, request, jsonify
from dapr.ext.workflow import DaprWorkflowClient, WorkflowRuntime, DaprWorkflowContext, WorkflowActivityContext

workflow_name = "test_workflow"
workflow_runtime = WorkflowRuntime()
logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger('Workflows')

app = Flask(__name__)
wfClient = None

@app.route('/healthz', methods=['GET'])
def health_check():
    """Health check endpoint"""
    return jsonify({"status": "healthy", "timestamp": datetime.now().isoformat()}), 200

@app.route('/start', methods=['POST'])
def start_workflow():
    """Start a new workflow instance"""
    global wfClient
    if wfClient is None:
        wfClient = DaprWorkflowClient()

    try:
        # Get input from request body or use a default counter
        request_data = request.get_json() if request.is_json else {}
        workflow_input = request_data.get('input', datetime.now().isoformat())

        logger.info(f"Starting workflow with input: {workflow_input}")
        instance_id = wfClient.schedule_new_workflow(workflow=test_workflow, input=workflow_input)

        logger.info(f"Workflow started with instance ID: {instance_id}")

        # Wait for workflow completion with timeout
        try:
            state = wfClient.wait_for_workflow_completion(instance_id=instance_id, timeout_in_seconds=30)
            if not state:
                logger.error("Workflow not found!")
                return jsonify({"error": "Workflow not found", "instance_id": instance_id}), 404
            elif state.runtime_status.name == 'COMPLETED':
                logger.info(f'Workflow completed! Result: {state.serialized_output}')
                return jsonify({
                    "status": "completed",
                    "instance_id": instance_id,
                    "result": state.serialized_output
                }), 200
            else:
                logger.error(f'Workflow failed! Status: {state.runtime_status.name}')
                return jsonify({
                    "status": "failed",
                    "instance_id": instance_id,
                    "runtime_status": state.runtime_status.name
                }), 500
        except TimeoutError:
            logger.error('Workflow timed out!')
            return jsonify({
                "status": "timeout",
                "instance_id": instance_id,
                "message": "Workflow execution timed out"
            }), 408

    except Exception as e:
        logger.error(f"Error starting workflow: {str(e)}")
        return jsonify({"error": f"Failed to start workflow: {str(e)}"}), 500

@workflow_runtime.workflow(name=workflow_name)
def test_workflow(ctx: DaprWorkflowContext, wf_input: str):
    logger.debug(f'Workflow {workflow_name} started. Input: {wf_input}')
    number = yield ctx.call_activity(random_number_generator)
    logger.debug(f'Workflow {workflow_name} completed. Number: {number}')
    return "Workflow completed with number: " + str(number)

@workflow_runtime.activity(name="random_number_generator")
def random_number_generator(ctx: WorkflowActivityContext):
    logger.debug(f'Random number activity started')
    number = random.randint(0, 100000)
    logger.debug(f'Random number activity completed. Number: {number}')
    return number

def main():
    workflow_runtime.start()
    app_port = os.getenv('APP_PORT', 6005)
    # Start Flask server
    logger.info(f"Starting HTTP server on port {app_port}")
    app.run(host='0.0.0.0', port=app_port, debug=False)

if __name__ == '__main__':
    main()
