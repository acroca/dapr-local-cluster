import json
import logging
import os
from cloudevents.sdk.event import v1
from dapr.ext.grpc import App
from dapr.clients.grpc._response import TopicEventResponse

# Configure logging
logging.basicConfig(level=logging.INFO)

# Create Dapr App
app = App()

@app.subscribe(pubsub_name='pubsub', topic='numbers')
def numbers_handler(event: v1.Event) -> TopicEventResponse:
    """Handle messages from the 'numbers' topic"""
    res = TopicEventResponse('success')
    try:
        # Parse the event data
        data = json.loads(event.Data())
        logging.info(f"Subscriber received sdfjhsaklfjhsalkfhsadkl: {data}")
    except Exception as e:
        logging.error(f"Error processing message: {e}")
        res = TopicEventResponse('retry')
    return res

if __name__ == '__main__':
    # Get app port from environment or use default
    app_port = int(os.getenv('APP_PORT', '6005'))

    logging.info(f"Starting subscriber on port {app_port}")

    app.register_health_check(lambda: print('Healthy'))
    # Run the app
    app.run(app_port)
