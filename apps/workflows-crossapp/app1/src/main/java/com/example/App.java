package com.example;

import io.dapr.workflows.client.DaprWorkflowClient;
import io.dapr.workflows.runtime.WorkflowRuntimeBuilder;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.http.MediaType;
import org.springframework.web.bind.annotation.*;

import java.time.Duration;
import java.time.OffsetDateTime;

@SpringBootApplication
public class App {

  public static void main(String[] args) {

    SpringApplication.run(App.class, args);
  }

}

@RestController
class ApiController {

  private final DaprWorkflowClient workflowClient;

  ApiController() {
    var builder = new WorkflowRuntimeBuilder();
    builder.registerWorkflow(TestWorkflow.class);
    builder.registerActivity(TestActivity.class);
    var workflowRuntime = builder.build();
    workflowRuntime.start(false);

    // Start workflow runtime via builder and register workflow & activity
    this.workflowClient = new DaprWorkflowClient();
  }

  static class HealthResponse {
    public String status;
    public String timestamp;
  }

  static class WorkflowRequest {
    public String input;
  }

  static class WorkflowResponse {
    public String status;
    public String instance_id;
    public String result;
    public String message;
    public String error;
  }

  @GetMapping(path = "/healthz", produces = MediaType.APPLICATION_JSON_VALUE)
  public HealthResponse health() {
    HealthResponse r = new HealthResponse();
    r.status = "healthy";
    r.timestamp = OffsetDateTime.now().toString();
    return r;
  }

  @PostMapping(path = "/start", consumes = MediaType.APPLICATION_JSON_VALUE, produces = MediaType.APPLICATION_JSON_VALUE)
  public WorkflowResponse start(@RequestBody(required = false) WorkflowRequest req) {
    WorkflowResponse resp = new WorkflowResponse();
    try {
      String instanceId = this.workflowClient.scheduleNewWorkflow(TestWorkflow.class);
      resp.instance_id = instanceId;

      // Wait up to 30s for completion
      this.workflowClient.waitForInstanceCompletion(instanceId, Duration.ofSeconds(30), true);
      resp.status = "completed";
      return resp;
    } catch (Exception e) {
      resp.status = "failed";
      resp.error = "Failed to start or await workflow: " + e.getMessage();
      return resp;
    }
  }
}

