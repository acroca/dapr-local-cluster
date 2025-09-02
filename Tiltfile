load('ext://helm_resource', 'helm_resource', 'helm_repo')

k8s_kind('Namespace')
k8s_kind('Secret')
k8s_kind('Role')
k8s_kind('RoleBinding')
k8s_kind('Resiliency')
k8s_kind('Subscription')
k8s_kind('HTTPEndpoint')
k8s_kind('Service')
k8s_kind('Configuration')
k8s_kind('Component')

run_e2e = False
# run_e2e = True
helm_repo('bitnami', 'https://charts.bitnami.com/bitnami')
helm_repo('openzipkin', 'https://zipkin.io/zipkin-helm')

if run_e2e:
  load_dynamic('e2e/Tiltfile')
else:
  helm_resource('redis', 'bitnami/redis',
              flags=[
                  '--set', 'architecture=standalone',
                  '--set', 'auth.enabled=false',
                  '--set', 'master.resources.requests.memory=512Mi',
                  '--set', 'master.resources.requests.cpu=200m',
                  '--set', 'master.resources.limits.memory=1024Mi',
                  '--set', 'master.resources.limits.cpu=200m',
                ],
              resource_deps=['bitnami'],
              labels=['core'])
  k8s_yaml("manifests/redis_insight.yaml")
  k8s_resource(workload='redisinsight', resource_deps=['redis'], labels=['core'], port_forwards=['5540:5540'], links="http://localhost:5540/0/browser")

  helm_resource('zipkin', 'openzipkin/zipkin',
              flags=['--set', 'zipkin.storage.type=mem'],
              resource_deps=['openzipkin'],
              labels=['core'],
              port_forwards=['9411:9411'])

  dapr_cli_version = "1.15"
  # dapr_cli_version = "dev" # use ../dapr instead of a release

  if dapr_cli_version == "dev":
    local_resource('dapr-images',
                  dir='../dapr',
                  cmd='mise exec dapr@1.15 -- dapr uninstall -k && make build docker-push',
                  env={
                    'HA_MODE': 'true',
                    'DAPR_REGISTRY': 'localhost:5001/dapr',
                    'DAPR_TAG': 'dev',
                    'DAPR_TEST_NAMESPACE': 'default',
                    'DAPR_NAMESPACE': 'default',
                    'TARGET_OS': 'linux',
                    'TARGET_ARCH': 'arm64',
                    'GOOS': 'linux',
                    'GOARCH': 'arm64',
                    'LOG_LEVEL': 'debug'
                  },
                  labels=['core'])

    helm_resource('dapr', '../dapr/charts/dapr',
                release_name='dapr',
                flags=[
                    '--set', 'global.ha.enabled=true',
                    '--set', 'global.tag=dev-linux-arm64',
                    '--set', 'global.registry=localhost:5001/dapr',
                    '--set', 'global.logAsJson=true',
                    '--set', 'global.daprControlPlaneOs=linux',
                    '--set', 'global.daprControlPlaneArch=arm64',
                    '--set', 'dapr_placement.logLevel=debug',
                    '--set', 'dapr_sentry.logLevel=debug',
                    '--set', 'dapr_sidecar_injector.sidecarImagePullPolicy=Always',
                    '--set', 'global.imagePullPolicy=Always',
                    '--set', 'global.imagePullSecrets=',
                    '--set', 'global.mtls.enabled=true',
                    '--set', 'dapr_placement.cluster.forceInMemoryLog=true',
                    '--set', 'dapr_scheduler.replicaCount=1',
                    '--set', 'dapr_scheduler.cluster.storageSize=100Mi',
                    '--set', 'dapr_scheduler.cluster.inMemoryStorage=false',
                    '--set', 'dapr_scheduler.logLevel=debug',
                  ],
                resource_deps=['dapr-images'],
                labels=['core'])
  else:
    # runtime_version = "latest"
    # runtime_version = "1.13.6"
    # runtime_version = "1.14.4"
    runtime_version = "1.16.0-rc.5"
    local_resource('dapr',
                  cmd='''
                    mise exec dapr@%s -- dapr uninstall -k -n default && \
                    mise exec dapr@%s -- dapr init -k -n default --runtime-version %s --wait
                  ''' % (dapr_cli_version, dapr_cli_version, runtime_version),
                  labels=['core'])

  k8s_yaml("manifests/config.yaml")
  k8s_resource(workload='daprconfig', resource_deps=['dapr'], labels=['components'], pod_readiness="ignore")
  k8s_yaml("manifests/component_pubsub.yaml")
  k8s_resource(workload='pubsub', resource_deps=['dapr', 'redis'], labels=['components'], pod_readiness="ignore")
  k8s_yaml("manifests/component_state.yaml")
  k8s_resource(workload='statestore', resource_deps=['dapr', 'redis'], labels=['components'], pod_readiness="ignore")
  k8s_yaml("manifests/component_workflowstate.yaml")
  k8s_resource(workload='workflowstatestore', resource_deps=['dapr', 'redis'], labels=['components'], pod_readiness="ignore")

  # load_dynamic('apps/actors-go/Tiltfile')
  # load_dynamic('apps/pub/Tiltfile')
  # load_dynamic('apps/sub/Tiltfile')
  # load_dynamic('apps/workflows-py/Tiltfile')
  # load_dynamic('apps/workflows-crossapp/Tiltfile')
  # load_dynamic('apps/workflows-go/Tiltfile')
  # load_dynamic('apps/workflows-stress/Tiltfile')
  # load_dynamic('apps/dapr-agents/Tiltfile')
  # load_dynamic('apps/tracing-dotnet/Tiltfile')



