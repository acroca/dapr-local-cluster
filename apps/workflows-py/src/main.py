from datetime import datetime
import random
import logging
import os
from opentelemetry import trace
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.semconv.resource import ResourceAttributes

resource = Resource.create({
    ResourceAttributes.SERVICE_NAME: "workflows-py",
    ResourceAttributes.SERVICE_VERSION: "1.0.0",
})

tracer_provider = TracerProvider(resource=resource)
trace.set_tracer_provider(tracer_provider)

exporter = OTLPSpanExporter(
    endpoint="http://otel-collector-opentelemetry-collector.default.svc.cluster.local:4317",
    # headers={"uptrace-dsn": "workflows-py"},
    timeout=30,
)

span_processor = BatchSpanProcessor(
    exporter,
    max_queue_size=1000,
    max_export_batch_size=1000,
)
tracer_provider.add_span_processor(span_processor)

tracer = trace.get_tracer(__name__)


from flask import Flask, request, jsonify
from opentelemetry.instrumentation.flask import FlaskInstrumentor
from dapr.ext.workflow import DaprWorkflowClient, DaprWorkflowContext, WorkflowActivityContext, WorkflowRuntime

wfr = WorkflowRuntime()
wfClient = DaprWorkflowClient()

logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger('Workflows')

app = Flask(__name__)
FlaskInstrumentor().instrument_app(app)

@app.route('/healthz', methods=['GET'])
def health_check():
    """Health check endpoint"""
    return jsonify({"status": "healthy", "timestamp": datetime.now().isoformat()}), 200

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

@wfr.workflow
def test_workflow(ctx: DaprWorkflowContext, wf_input: str):
    logger.debug(f'Workflow test_workflow started. Input: {wf_input}')
    numbers = []
    number = yield ctx.call_activity(random_number_generator)
    numbers.append(str(number))
    logger.debug(f'Workflow test_workflow completed. Number: {number}')

    return "Workflow completed with numbers: " + " ".join(numbers)

@wfr.activity
def random_number_generator(ctx: WorkflowActivityContext):
    logger.debug('Random number activity started')
    with tracer.start_as_current_span(name='IN-ACTIVITY'):
        number = random.randint(0, 100000)
    logger.debug(f'Random number activity completed. Number: {number}')
    return number

def main():
    wfr.start()
    app_port = int(os.getenv('APP_PORT', 6005))
    # Start Flask server
    logger.info(f"Starting HTTP server on port {app_port}")
    app.run(host='0.0.0.0', port=app_port, debug=False)

if __name__ == '__main__':
    main()
