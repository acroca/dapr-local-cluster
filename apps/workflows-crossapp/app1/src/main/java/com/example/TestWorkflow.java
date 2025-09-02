package com.example;

import io.dapr.workflows.Workflow;
import io.dapr.workflows.WorkflowStub;
import io.dapr.workflows.WorkflowTaskOptions;

public class TestWorkflow implements Workflow {
  @Override
  public WorkflowStub create() {
    return ctx -> {
      Integer number = ctx.callActivity("TestActivity2", null, new WorkflowTaskOptions("workflows-crossapp2"), Integer.class).await();
      ctx.getLogger().info("Workflow completed with number: " + number);
      ctx.complete(number);
    };
  }
}
