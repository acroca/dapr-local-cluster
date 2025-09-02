package com.example;

import java.util.Random;

import io.dapr.workflows.WorkflowActivity;
import io.dapr.workflows.WorkflowActivityContext;

public class TestActivity implements WorkflowActivity {
  @Override
  public Object run(WorkflowActivityContext ctx) {
    return new Random().nextInt(100000);
  }
}

