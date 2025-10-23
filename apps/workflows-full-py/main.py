from datetime import datetime
import random
from time import sleep
import logging
import os
import signal
import sys
import threading
import time

from flask import Flask, request, jsonify
from dapr.ext.workflow import DaprWorkflowClient, WorkflowRuntime, DaprWorkflowContext, WorkflowActivityContext

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
        instance_id = wfClient.schedule_new_workflow(workflow=root_workflow, input=workflow_input)

        logger.info(f"Workflow started with instance ID: {instance_id}")

        # Wait for workflow completion with timeout
        try:
            state = wfClient.wait_for_workflow_completion(instance_id=instance_id, timeout_in_seconds=30)
            if not state:
                logger.error("Workflow not found!")
                return jsonify({"error": "Workflow not found", "instance_id": instance_id}), 404
            elif state.runtime_status.name == 'COMPLETED':
                logger.info(f'Workflow completed!')
                return jsonify({
                    "status": "completed",
                    "instance_id": instance_id,
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


@workflow_runtime.workflow
def root_workflow(ctx: DaprWorkflowContext, wf_input: str):
    n = yield ctx.call_activity(double_activity, input=4)
    if n != 8:
        raise Exception("Number is not 8")
    n = yield ctx.call_child_workflow(child_workflow_async_activities, input=4)
    if n != 16:
        raise Exception("Number is not 16")
    n = yield ctx.call_child_workflow(child_workflow_async_activities, input=5, app_id="workflows-full-py-2")
    if n != 20:
        raise Exception("Number is not 20")
    n = yield ctx.call_child_workflow(child_workflow_n_times, input={"n": 4, "times": 3})
    if n != 32:
        raise Exception("Number is not 32")
    n = yield ctx.call_child_workflow(child_workflow_n_times, input={"n": 5, "times": 3}, app_id="workflows-full-py-2")
    if n != 40:
        raise Exception("Number is not 40")
    return None

@workflow_runtime.workflow
def child_workflow_async_activities(ctx: DaprWorkflowContext, wf_input: int):
    n = wf_input
    now = time.time()
    a1 = ctx.call_activity(double_activity, input=n)
    a2 = ctx.call_activity(double_activity, input=n)
    n1 = yield a1
    n2 = yield a2
    if time.time() - now >= 2:
        raise Exception("Activities didn't run in parallel")
    return n1 + n2


@workflow_runtime.workflow
def child_workflow_n_times(ctx: DaprWorkflowContext, wf_input: dict[str, int]):
    n = wf_input["n"]
    times = wf_input["times"]
    n = yield ctx.call_activity(double_activity, input=n)
    if times > 1:
        ctx.continue_as_new(new_input={"n": n, "times": times - 1})
    return n

@workflow_runtime.activity
def double_activity(ctx: WorkflowActivityContext, input: int):
    sleep(1)
    return input * 2


def main():
    workflow_runtime.start()
    app_port = os.getenv('APP_PORT', 6021)
    # Start Flask server
    logger.info(f"Starting HTTP server on port {app_port}")
    app.run(host='0.0.0.0', port=app_port, debug=False)

if __name__ == '__main__':
    main()
