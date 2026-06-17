import asyncio
from datetime import datetime, timedelta
import random
import logging
import os
from time import sleep
# from opentelemetry import trace
# from opentelemetry.sdk.resources import Resource
# from opentelemetry.sdk.trace import TracerProvider
# from opentelemetry.sdk.trace.export import BatchSpanProcessor
# from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
# from opentelemetry.semconv.resource import ResourceAttributes

# resource = Resource.create({
#     ResourceAttributes.SERVICE_NAME: "workflows-py",
#     ResourceAttributes.SERVICE_VERSION: "1.0.0",
# })

# tracer_provider = TracerProvider(resource=resource)
# trace.set_tracer_provider(tracer_provider)

# exporter = OTLPSpanExporter(
#     endpoint="http://otel-collector-opentelemetry-collector.default.svc.cluster.local:4317",
#     # headers={"uptrace-dsn": "workflows-py"},
#     timeout=30,
# )

# span_processor = BatchSpanProcessor(
#     exporter,
#     max_queue_size=1000,
#     max_export_batch_size=1000,
# )
# tracer_provider.add_span_processor(span_processor)

# tracer = trace.get_tracer(__name__)


from flask import Flask, request, jsonify
# from opentelemetry.instrumentation.flask import FlaskInstrumentor
from dapr.ext.workflow import DaprWorkflowClient, DaprWorkflowContext, RetryPolicy, WorkflowActivityContext, WorkflowRuntime, when_all

wfr = WorkflowRuntime()
wfClient = DaprWorkflowClient()

logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger('Workflows')

app = Flask(__name__)
# FlaskInstrumentor().instrument_app(app)

@app.route('/healthz', methods=['GET'])
def health_check():
    """Health check endpoint"""
    return jsonify({"status": "healthy", "timestamp": datetime.now().isoformat()}), 200

@app.route('/continue', methods=['POST'])
def continue_workflow():
    """Continue a workflow instance"""
    try:
        instance_id = request.args.get('instance_id')
        print(f"Continuing workflow with instance ID: {instance_id}")
        wfClient.raise_workflow_event(instance_id=instance_id, event_name='event')
        return jsonify({"status": "ok"}), 200
    except Exception as e:
        logger.error(f"Error continuing workflow: {str(e)}")
        return jsonify({"error": f"Failed to continue workflow: {str(e)}"}), 500

@app.route('/start', methods=['POST'])
def start_workflow():
    """Start a new workflow instance"""

    try:
        # Get input from request body or use a default counter
        request_data = request.get_json() if request.is_json else {}
        workflow_input = request_data.get('input', datetime.now().isoformat())

        logger.info(f"Starting workflow with input: {workflow_input}")
        instance_id = wfClient.schedule_new_workflow(workflow=test_workflow, input=workflow_input)

        logger.info(f"Workflow started with instance ID: {instance_id}")

        # Wait for workflow completion with timeout
        try:
            state = wfClient.wait_for_workflow_completion(instance_id=instance_id, timeout_in_seconds=300)
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

@wfr.workflow
def test_workflow(ctx: DaprWorkflowContext, wf_input: str):
    logger.debug(f'Workflow test_workflow started. Input: {wf_input}')
    numbers = []

    tasks = [
        ctx.call_child_workflow(child_workflow, input="", retry_policy=RetryPolicy(
            first_retry_interval=timedelta(seconds=0),
            max_number_of_attempts=3,
            max_retry_interval=timedelta(seconds=0),
            retry_timeout=timedelta(seconds=3),
        )) for _ in range(2)
    ]
    numbers = yield when_all(tasks)

    return "Workflow completed with numbers: " + " ".join([str(n) for n in numbers])

attempts = {}

@wfr.workflow
def child_workflow(ctx: DaprWorkflowContext, wf_input: str):
    logger.info(f"Executing child workflow with ID: {ctx.instance_id}")
    sleep(1)
    if attempts.get(ctx.instance_id, 0) < 2:
        attempts[ctx.instance_id] = attempts.get(ctx.instance_id, 0) + 1
        logger.debug(f'Child workflow {ctx.instance_id} failed on attempt {attempts[ctx.instance_id]}')
        raise ValueError(f'Simulated failure on attempt {attempts[ctx.instance_id]}')
    number = random.randint(0, 100000)
    return number

@wfr.activity
def random_number_generator(ctx: WorkflowActivityContext):
    sleep(10)
    number = random.randint(0, 100000)
    return number

def main():
    wfr.start()
    app_port = int(os.getenv('APP_PORT', 6005))
    # Start Flask server
    logger.info(f"Starting HTTP server on port {app_port}")
    app.run(host='0.0.0.0', port=app_port, debug=False)

if __name__ == '__main__':
    main()
